package handlers

import (
	"net/http"

	"github.com/Code-Linx/wallet-service/internal/usecases"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	useCases *usecases.UseCases
}

func (h *Handlers) SetupRouter(handlers *Handlers) *gin.Engine {
	router := gin.Default()

	api := router.Group("/api/v1")

	{
		// User routes
	api.POST("/users", handlers.CreateUser)
	api.GET("/users/:id", handlers.GetUser)

	// Wallet routes
	api.POST("/users/:id/wallet/fund", handlers.FundWallet)
	api.POST("/users/:id/wallet/withdraw", handlers.WithdrawFunds)
	api.POST("/users/:id/wallet/transfer", handlers.TransferFunds)
	api.GET("/users/:id/wallet/transactions", handlers.GetTransactionHistory)

	// Reconciliation route
	api.POST("/reconciliation/run", handlers.RunReconciliation)

	// Health check route
	api.GET("/health", handlers.HealthCheck)
	}

	return router
}

// NewHandlers creates new handler instances
func NewHandlers(useCases *usecases.UseCases) *Handlers {
	return &Handlers{
		useCases: useCases,
	}
}

// Request/Response DTOs

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type FundWalletRequest struct {
	Amount    int64  `json:"amount" binding:"required,min=1"`
	Reference string `json:"reference" binding:"required"`
}

type WithdrawFundsRequest struct {
	Amount    int64  `json:"amount" binding:"required,min=1"`
	Reference string `json:"reference" binding:"required"`
}

type TransferFundsRequest struct {
	ToUserID  string `json:"to_user_id" binding:"required"`
	Amount    int64  `json:"amount" binding:"required,min=1"`
	Reference string `json:"reference" binding:"required"`
}

type TransactionHistoryQuery struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"page_size,default=10" binding:"min=1,max=100"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// Helper functions

func successResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func errorResponse(c *gin.Context, statusCode int, message string, err error) {
	response := APIResponse{
		Success: false,
		Message: message,
	}
	if err != nil {
		response.Error = err.Error()
	}
	c.JSON(statusCode, response)
}

func paginatedResponse(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := PaginatedResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Transactions retrieved successfully",
		Data:    response,
	})
}

// User Handlers

func (h *Handlers) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	user, err := h.useCases.User.CreateUser(req.Name, req.Email)
	if err != nil {
		if err == usecases.ErrUserAlreadyExists {
			errorResponse(c, http.StatusConflict, "User already exists", err)
			return
		}
		errorResponse(c, http.StatusInternalServerError, "Failed to create user", err)
		return
	}

	successResponse(c, "User created successfully", user)
}

func (h *Handlers) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	user, err := h.useCases.User.GetUserByID(userID)
	if err != nil {
		if err == usecases.ErrUserNotFound {
			errorResponse(c, http.StatusNotFound, "User not found", err)
			return
		}
		errorResponse(c, http.StatusInternalServerError, "Failed to get user", err)
		return
	}

	successResponse(c, "User retrieved successfully", user)
}

// Wallet Handlers

func (h *Handlers) FundWallet(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var req FundWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	transaction, err := h.useCases.Wallet.FundWallet(userID, req.Amount, req.Reference)
	if err != nil {
		if err == usecases.ErrUserNotFound {
			errorResponse(c, http.StatusNotFound, "User not found", err)
			return
		}
		if err == usecases.ErrInvalidAmount {
			errorResponse(c, http.StatusBadRequest, "Invalid amount", err)
			return
		}
		errorResponse(c, http.StatusInternalServerError, "Failed to fund wallet", err)
		return
	}

	successResponse(c, "Wallet funded successfully", transaction)
}

func (h *Handlers) WithdrawFunds(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var req WithdrawFundsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	transaction, err := h.useCases.Wallet.WithdrawFunds(userID, req.Amount, req.Reference)
	if err != nil {
		if err == usecases.ErrUserNotFound {
			errorResponse(c, http.StatusNotFound, "User not found", err)
			return
		}
		if err == usecases.ErrInvalidAmount {
			errorResponse(c, http.StatusBadRequest, "Invalid amount", err)
			return
		}
		if err == usecases.ErrInsufficientFunds {
			errorResponse(c, http.StatusBadRequest, "Insufficient funds", err)
			return
		}
		errorResponse(c, http.StatusInternalServerError, "Failed to withdraw funds", err)
		return
	}

	successResponse(c, "Funds withdrawn successfully", transaction)
}

func (h *Handlers) TransferFunds(c *gin.Context) {
	fromUserIDStr := c.Param("id")
	fromUserID, err := uuid.Parse(fromUserIDStr)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var req TransferFundsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	toUserID, err := uuid.Parse(req.ToUserID)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid recipient user ID", err)
		return
	}

	transaction, err := h.useCases.Wallet.TransferFunds(fromUserID, toUserID, req.Amount, req.Reference)
	if err != nil {
		if err == usecases.ErrUserNotFound {
			errorResponse(c, http.StatusNotFound, "User not found", err)
			return
		}
		if err == usecases.ErrInvalidAmount {
			errorResponse(c, http.StatusBadRequest, "Invalid amount", err)
			return
		}
		if err == usecases.ErrInsufficientFunds {
			errorResponse(c, http.StatusBadRequest, "Insufficient funds", err)
			return
		}
		if err == usecases.ErrSameUser {
			errorResponse(c, http.StatusBadRequest, "Cannot transfer to the same user", err)
			return
		}
		errorResponse(c, http.StatusInternalServerError, "Failed to transfer funds", err)
		return
	}

	successResponse(c, "Funds transferred successfully", transaction)
}

func (h *Handlers) GetTransactionHistory(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var query TransactionHistoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid query parameters", err)
		return
	}

	transactions, total, err := h.useCases.Wallet.GetTransactionHistory(userID, query.Page, query.PageSize)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to get transaction history", err)
		return
	}

	paginatedResponse(c, transactions, query.Page, query.PageSize, total)
}

// Reconciliation Handlers

func (h *Handlers) RunReconciliation(c *gin.Context) {
	results, err := h.useCases.Reconciliation.RunReconciliation()
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to run reconciliation", err)
		return
	}

	successResponse(c, "Reconciliation completed successfully", results)
}

// Health Check Handler

func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "wallet-service",
		"version": "1.0.0",
	})
}
