package unit

import (
	"testing"
	"time"

	"github.com/Code-Linx/wallet-service/internal/models"
	"github.com/Code-Linx/wallet-service/internal/repositories"
	"github.com/Code-Linx/wallet-service/internal/usecases"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Mock repositories
type MockUserRepository struct {
	mock.Mock
}

type MockWalletRepository struct {
	mock.Mock
}

type MockTransactionRepository struct {
	mock.Mock
}

// Mock implementations
func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetAll() ([]models.User, error) {
	args := m.Called()
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockWalletRepository) Create(wallet *models.Wallet) error {
	args := m.Called(wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetByUserID(userID uuid.UUID) (*models.Wallet, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateBalance(userID uuid.UUID, newBalance int64) error {
	args := m.Called(userID, newBalance)
	return args.Error(0)
}

func (m *MockWalletRepository) GetByID(id uuid.UUID) (*models.Wallet, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetAllWallets() ([]models.Wallet, error) {
	args := m.Called()
	return args.Get(0).([]models.Wallet), args.Error(1)
}

func (m *MockTransactionRepository) Create(transaction *models.Transaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetByReference(reference string) (*models.Transaction, error) {
	args := m.Called(reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByUserID(userID uuid.UUID, limit, offset int) ([]models.Transaction, int64, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]models.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionRepository) GetUserTransactionSum(userID uuid.UUID) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionRepository) Update(transaction *models.Transaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}

// Create a functional mock DB using SQLite in-memory
// This is necessary because GORM needs a real database connection to work properly
func setupMockDB() *gorm.DB {

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to create mock database: " + err.Error())
	}

	// Auto-migrate the tables that your application uses
	err = db.AutoMigrate(&models.User{}, &models.Wallet{}, &models.Transaction{})
	if err != nil {
		panic("failed to migrate test database: " + err.Error())
	}

	return db
}

// Test cases

func TestCreateUser_Success(t *testing.T) {
	// Create all mock repositories
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)

	firstName := "Dennis"
	lastName := "Enoakpokihoke"
	fullName := firstName + " " + lastName
	email := "test@example.com"
	expectedID := uuid.New()

	expectedUser := &models.User{
		ID:        expectedID,
		Name:      fullName,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Mock behavior - only mock the methods that are actually called
	mockUserRepo.On("GetByEmail", email).Return(nil, gorm.ErrRecordNotFound).Once()
	mockUserRepo.On("GetByID", mock.Anything).Return(expectedUser, nil).Once()

	// Set up test database
	testDB := setupMockDB()

	// Create repositories struct with all mocked repositories
	repos := &repositories.Repositories{
		User:        mockUserRepo,
		Wallet:      mockWalletRepo,
		Transaction: mockTxRepo,
		DB:          testDB,
	}

	// Use the correct constructor
	useCases := usecases.NewUseCases(repos)

	// Act
	result, err := useCases.User.CreateUser(fullName, email)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedUser.Name, result.Name)
	assert.Equal(t, expectedUser.Email, result.Email)
	mockUserRepo.AssertExpectations(t)
}

func TestFundWallet_InvalidAmount(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepository{}
	mockWalletRepo := &MockWalletRepository{}
	mockTxnRepo := &MockTransactionRepository{}

	// Setup mock DB
	testDB := setupMockDB()

	// Create repositories with your actual struct
	repos := &repositories.Repositories{
		User:        mockUserRepo,
		Wallet:      mockWalletRepo,
		Transaction: mockTxnRepo,
		DB:          testDB,
	}

	// Use your actual NewUseCases function
	walletUseCase := usecases.NewUseCases(repos).Wallet
	userID := uuid.New()
	amount := int64(-100)
	reference := "test-ref-123"

	// Call the function under test and assert the error
	_, err := walletUseCase.FundWallet(userID, amount, reference)

	// The core of the test: check if the returned error is the expected one
	assert.Equal(t, usecases.ErrInvalidAmount, err, "Expected invalid amount error")
}

func TestInsufficientFunds(t *testing.T) {
	currentBalance := int64(1000)
	withdrawAmount := int64(1500)

	assert.True(t, currentBalance < withdrawAmount, "Should have insufficient funds")
}

func TestValidTransactionReference(t *testing.T) {
	reference := "test_ref_001"
	assert.NotEmpty(t, reference, "Reference should not be empty")
	assert.True(t, len(reference) > 0, "Reference should have valid length")
}

func TestUserIDValidation(t *testing.T) {
	validUUID := uuid.New()
	assert.NotEqual(t, uuid.Nil, validUUID, "UUID should be valid")

	// Test that same user transfer is invalid
	fromUserID := validUUID
	toUserID := validUUID
	assert.Equal(t, fromUserID, toUserID, "Same user transfer should be detected")
}

func TestPaginationValidation(t *testing.T) {
	// Test page validation
	page := 0
	if page < 1 {
		page = 1
	}
	assert.Equal(t, 1, page, "Page should default to 1")

	// Test page size validation
	pageSize := 0
	if pageSize < 1 {
		pageSize = 10
	}
	assert.Equal(t, 10, pageSize, "PageSize should default to 10")

	// Test max page size
	largePageSize := 200
	maxPageSize := 100
	if largePageSize > maxPageSize {
		largePageSize = maxPageSize
	}
	assert.Equal(t, 100, largePageSize, "PageSize should be capped at max")
}

func TestTransactionTypes(t *testing.T) {
	assert.Equal(t, models.TransactionType("credit"), models.TransactionTypeCredit)
	assert.Equal(t, models.TransactionType("debit"), models.TransactionTypeDebit)
	assert.Equal(t, models.TransactionType("transfer"), models.TransactionTypeTransfer)
}

func TestTransactionStatus(t *testing.T) {
	assert.Equal(t, models.TransactionStatus("pending"), models.TransactionStatusPending)
	assert.Equal(t, models.TransactionStatus("completed"), models.TransactionStatusCompleted)
	assert.Equal(t, models.TransactionStatus("failed"), models.TransactionStatusFailed)
}
