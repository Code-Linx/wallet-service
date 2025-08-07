package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Code-Linx/wallet-service/internal/config"
	"github.com/Code-Linx/wallet-service/internal/handlers"
	"github.com/Code-Linx/wallet-service/internal/repositories"
	"github.com/Code-Linx/wallet-service/internal/usecases"
	"github.com/Code-Linx/wallet-service/pkg/database"
	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type APITestSuite struct {
	suite.Suite
	router  *gin.Engine
	db      *gorm.DB
	cleanup func()
}

func (suite *APITestSuite) SetupSuite() {
	// Load .env.test
	_ = godotenv.Load(".env.test")

	// Set test environment
	os.Setenv("APP_ENV", "test")

	gin.SetMode(gin.TestMode)

	// Load config
	cfg, err := config.Load()
	suite.Require().NoError(err)

	// Connect to test DB
	db, err := database.NewConnection(cfg)
	suite.Require().NoError(err)
	suite.db = db

	// Run migrations
	err = database.AutoMigrate(db)
	suite.Require().NoError(err)

	// Init app
	repos := repositories.NewRepositories(db)
	useCases := usecases.NewUseCases(repos)
	handlersInstance := handlers.NewHandlers(useCases)
	suite.router = handlersInstance.SetupRouter(handlersInstance)

	// Cleanup setup
	suite.cleanup = func() {
		db.Exec("DELETE FROM transactions")
		db.Exec("DELETE FROM wallets")
		db.Exec("DELETE FROM users")
	}
}

func (suite *APITestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
	sqlDB, _ := suite.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

func (suite *APITestSuite) SetupTest() {
	// Clean up before each test
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite *APITestSuite) TestHealthCheck() {
	
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

func (suite *APITestSuite) TestCreateUser() {
	userPayload := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	jsonPayload, _ := json.Marshal(userPayload)
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response handlers.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "User created successfully", response.Message)

	// Check that user data is returned
	userData := response.Data.(map[string]interface{})
	assert.NotNil(suite.T(), userData["id"])
	assert.Equal(suite.T(), "John Doe", userData["name"])
	assert.Equal(suite.T(), "john@example.com", userData["email"])
}

func (suite *APITestSuite) TestCreateUserDuplicate() {
	// Create first user
	userPayload := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	jsonPayload, _ := json.Marshal(userPayload)
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Try to create duplicate user
	req2, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(jsonPayload))
	req2.Header.Set("Content-Type", "application/json")

	resp2 := httptest.NewRecorder()
	suite.router.ServeHTTP(resp2, req2)
	assert.Equal(suite.T(), http.StatusConflict, resp2.Code)
}

func (suite *APITestSuite) makeWalletRequest(method, url string, payload map[string]interface{}, expectedStatus int) *httptest.ResponseRecorder {
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), expectedStatus, resp.Code)
	return resp
}

func (suite *APITestSuite) TestFullWalletFlow() {
	// Step 1: Create two users
	user1ID := suite.createTestUser("John Doe", "john@example.com")
	user2ID := suite.createTestUser("Jane Smith", "jane@example.com")

	// Step 2: Fund user1's wallet - UPDATED: Added /wallet/ to match your router
	fundPayload := map[string]interface{}{
		"amount":    10000,
		"reference": "fund_001",
	}
	suite.makeWalletRequest("POST", fmt.Sprintf("/api/v1/users/%s/wallet/fund", user1ID), fundPayload, http.StatusOK)

	// Step 3: Check user1's balance
	user1Data := suite.getUser(user1ID)
	wallet1 := user1Data["wallet"].(map[string]interface{})
	assert.Equal(suite.T(), float64(10000), wallet1["balance"])

	// Step 4: Withdraw from user1's wallet - 
	withdrawPayload := map[string]interface{}{
		"amount":    2000,
		"reference": "withdraw_001",
	}
	suite.makeWalletRequest("POST", fmt.Sprintf("/api/v1/users/%s/wallet/withdraw", user1ID), withdrawPayload, http.StatusOK)

	// Step 5: Transfer from user1 to user2 - 
	transferPayload := map[string]interface{}{
		"to_user_id": user2ID,
		"amount":     3000,
		"reference":  "transfer_001",
	}
	suite.makeWalletRequest("POST", fmt.Sprintf("/api/v1/users/%s/wallet/transfer", user1ID), transferPayload, http.StatusOK)

	// Step 6: Check final balances
	user1Final := suite.getUser(user1ID)
	user2Final := suite.getUser(user2ID)

	wallet1Final := user1Final["wallet"].(map[string]interface{})
	wallet2Final := user2Final["wallet"].(map[string]interface{})

	assert.Equal(suite.T(), float64(5000), wallet1Final["balance"]) // 10000 - 2000 - 3000
	assert.Equal(suite.T(), float64(3000), wallet2Final["balance"]) // 0 + 3000

	// Step 7: Test transaction history - 
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s/wallet/transactions?page=1&page_size=10", user1ID), nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var historyResponse handlers.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &historyResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), historyResponse.Success)

	// Step 8: Run reconciliation
	req, _ = http.NewRequest("POST", "/api/v1/reconciliation/run", nil)
	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var reconciliationResponse handlers.APIResponse
	err = json.Unmarshal(resp.Body.Bytes(), &reconciliationResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), reconciliationResponse.Success)
}

func (suite *APITestSuite) TestIdempotency() {
	// Create user
	userID := suite.createTestUser("Test User", "test@example.com")

	// Fund wallet twice with same reference - 
	fundPayload := map[string]interface{}{
		"amount":    5000,
		"reference": "idempotent_fund_001",
	}

	// First request
	resp1 := suite.makeWalletRequest("POST", fmt.Sprintf("/api/v1/users/%s/wallet/fund", userID), fundPayload, http.StatusOK)

	// Second request with same reference
	resp2 := suite.makeWalletRequest("POST", fmt.Sprintf("/api/v1/users/%s/wallet/fund", userID), fundPayload, http.StatusOK)

	// Both should return the same transaction
	var response1, response2 handlers.APIResponse
	json.Unmarshal(resp1.Body.Bytes(), &response1)
	json.Unmarshal(resp2.Body.Bytes(), &response2)

	txn1 := response1.Data.(map[string]interface{})
	txn2 := response2.Data.(map[string]interface{})

	assert.Equal(suite.T(), txn1["id"], txn2["id"])

	// Check that balance is only credited once
	userData := suite.getUser(userID)
	wallet := userData["wallet"].(map[string]interface{})
	assert.Equal(suite.T(), float64(5000), wallet["balance"])
}

func (suite *APITestSuite) TestInsufficientFunds() {
	// Create user
	userID := suite.createTestUser("Poor User", "poor@example.com")

	// Try to withdraw without funds - 
	withdrawPayload := map[string]interface{}{
		"amount":    1000,
		"reference": "withdraw_insufficient",
	}

	suite.makeWalletRequest("POST", fmt.Sprintf("/api/v1/users/%s/wallet/withdraw", userID), withdrawPayload, http.StatusBadRequest)
}

// Helper methods
func (suite *APITestSuite) createTestUser(name, email string) string {
	userPayload := map[string]string{
		"name":  name,
		"email": email,
	}

	jsonPayload, _ := json.Marshal(userPayload)
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	var response handlers.APIResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	userData := response.Data.(map[string]interface{})
	return userData["id"].(string)
}

func (suite *APITestSuite) getUser(userID string) map[string]interface{} {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", userID), nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	var response handlers.APIResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	return response.Data.(map[string]interface{})
}

// Run the test suite
func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func TestMain(m *testing.M) {
	// Load .env.test from project root
	err := godotenv.Load(filepath.Join("..", "..", ".env.test"))
	if err != nil {
		fmt.Println("❌ Failed to load .env.test:", err)
	} else {
		fmt.Println("✅ .env.test loaded")
	}

	os.Setenv("APP_ENV", "test")

	code := m.Run()
	os.Exit(code)
}
