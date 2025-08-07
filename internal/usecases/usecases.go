package usecases

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Code-Linx/wallet-service/internal/models"
	"github.com/Code-Linx/wallet-service/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrTransactionExists = errors.New("transaction with this reference already exists")
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrSameUser          = errors.New("cannot transfer to the same user")
)

// UserUseCase interface
type UserUseCase interface {
	CreateUser(name, email string) (*models.User, error)
	GetUserByID(id uuid.UUID) (*models.User, error)
}

// WalletUseCase interface
type WalletUseCase interface {
	FundWallet(userID uuid.UUID, amount int64, reference string) (*models.Transaction, error)
	WithdrawFunds(userID uuid.UUID, amount int64, reference string) (*models.Transaction, error)
	TransferFunds(fromUserID, toUserID uuid.UUID, amount int64, reference string) (*models.Transaction, error)
	GetTransactionHistory(userID uuid.UUID, page, pageSize int) ([]models.Transaction, int64, error)
}

// ReconciliationUseCase interface
type ReconciliationUseCase interface {
	RunReconciliation() ([]models.ReconciliationResult, error)
}

// UseCases holds all use case instances
type UseCases struct {
	User           UserUseCase
	Wallet         WalletUseCase
	Reconciliation ReconciliationUseCase
}

// userUseCase implements UserUseCase
type userUseCase struct {
	repos *repositories.Repositories
}

// walletUseCase implements WalletUseCase
type walletUseCase struct {
	repos *repositories.Repositories
}

// reconciliationUseCase implements ReconciliationUseCase
type reconciliationUseCase struct {
	repos *repositories.Repositories
}

// NewUseCases creates new use case instances
func NewUseCases(repos *repositories.Repositories) *UseCases {
	return &UseCases{
		User:           &userUseCase{repos: repos},
		Wallet:         &walletUseCase{repos: repos},
		Reconciliation: &reconciliationUseCase{repos: repos},
	}
}

// User Use Case Implementation

func (uc *userUseCase) CreateUser(name, email string) (*models.User, error) {
	// Check if user already exists
	existingUser, err := uc.repos.User.GetByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Start transaction
	tx := uc.repos.BeginTransaction()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txRepos := uc.repos.WithTransaction(tx)

	// Create user
	user := &models.User{
		Name:  name,
		Email: email,
	}

	if err := txRepos.User.Create(user); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create wallet for user
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 0,
	}

	if err := txRepos.Wallet.Create(wallet); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load user with wallet
	return uc.repos.User.GetByID(user.ID)
}

func (uc *userUseCase) GetUserByID(id uuid.UUID) (*models.User, error) {
	user, err := uc.repos.User.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// Wallet Use Case Implementation

func (uc *walletUseCase) FundWallet(userID uuid.UUID, amount int64, reference string) (*models.Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Check if transaction already exists (idempotency)
	existingTxn, err := uc.repos.Transaction.GetByReference(reference)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing transaction: %w", err)
	}
	if existingTxn != nil {
		return existingTxn, nil // Return existing transaction
	}

	// Check if user exists
	user, err := uc.repos.User.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Start transaction
	tx := uc.repos.BeginTransaction()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txRepos := uc.repos.WithTransaction(tx)

	// Get current wallet
	wallet, err := txRepos.Wallet.GetByUserID(userID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create transaction record
	transaction := &models.Transaction{
		UserID:      userID,
		Type:        models.TransactionTypeCredit,
		Amount:      amount,
		Description: "Wallet funding",
		Status:      models.TransactionStatusCompleted,
		Reference:   reference,
	}

	if err := txRepos.Transaction.Create(transaction); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update wallet balance
	newBalance := wallet.Balance + amount
	if err := txRepos.Wallet.UpdateBalance(userID, newBalance); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load transaction with user data
	transaction.User = *user
	return transaction, nil
}

func (uc *walletUseCase) WithdrawFunds(userID uuid.UUID, amount int64, reference string) (*models.Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Check if transaction already exists (idempotency)
	existingTxn, err := uc.repos.Transaction.GetByReference(reference)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing transaction: %w", err)
	}
	if existingTxn != nil {
		return existingTxn, nil // Return existing transaction
	}

	// Check if user exists
	user, err := uc.repos.User.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Start transaction
	tx := uc.repos.BeginTransaction()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txRepos := uc.repos.WithTransaction(tx)

	// Get current wallet
	wallet, err := txRepos.Wallet.GetByUserID(userID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Check sufficient funds
	if wallet.Balance < amount {
		tx.Rollback()
		return nil, ErrInsufficientFunds
	}

	// Create transaction record
	transaction := &models.Transaction{
		UserID:      userID,
		Type:        models.TransactionTypeDebit,
		Amount:      amount,
		Description: "Wallet withdrawal",
		Status:      models.TransactionStatusCompleted,
		Reference:   reference,
	}

	if err := txRepos.Transaction.Create(transaction); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update wallet balance
	newBalance := wallet.Balance - amount
	if err := txRepos.Wallet.UpdateBalance(userID, newBalance); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load transaction with user data
	transaction.User = *user
	return transaction, nil
}

func (uc *walletUseCase) TransferFunds(fromUserID, toUserID uuid.UUID, amount int64, reference string) (*models.Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	if fromUserID == toUserID {
		return nil, ErrSameUser
	}

	// Check if transaction already exists (idempotency)
	existingTxn, err := uc.repos.Transaction.GetByReference(reference)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing transaction: %w", err)
	}
	if existingTxn != nil {
		return existingTxn, nil // Return existing transaction
	}

	// Check if both users exist
	fromUser, err := uc.repos.User.GetByID(fromUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sender not found")
		}
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}

	toUser, err := uc.repos.User.GetByID(toUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("recipient not found")
		}
		return nil, fmt.Errorf("failed to get recipient: %w", err)
	}

	// Start transaction
	tx := uc.repos.BeginTransaction()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txRepos := uc.repos.WithTransaction(tx)

	// Get sender's wallet
	fromWallet, err := txRepos.Wallet.GetByUserID(fromUserID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get sender wallet: %w", err)
	}

	// Check sufficient funds
	if fromWallet.Balance < amount {
		tx.Rollback()
		return nil, ErrInsufficientFunds
	}

	// Get recipient's wallet
	toWallet, err := txRepos.Wallet.GetByUserID(toUserID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get recipient wallet: %w", err)
	}

	// Create transaction record
	transaction := &models.Transaction{
		UserID:      fromUserID,
		Type:        models.TransactionTypeTransfer,
		Amount:      amount,
		Description: fmt.Sprintf("Transfer to %s", toUser.Name),
		Status:      models.TransactionStatusCompleted,
		Reference:   reference,
		FromUserID:  &fromUserID,
		ToUserID:    &toUserID,
	}

	if err := txRepos.Transaction.Create(transaction); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update sender's wallet balance
	newFromBalance := fromWallet.Balance - amount
	if err := txRepos.Wallet.UpdateBalance(fromUserID, newFromBalance); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update sender wallet balance: %w", err)
	}

	// Update recipient's wallet balance
	newToBalance := toWallet.Balance + amount
	if err := txRepos.Wallet.UpdateBalance(toUserID, newToBalance); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update recipient wallet balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load transaction with user data
	transaction.User = *fromUser
	transaction.FromUser = fromUser
	transaction.ToUser = toUser
	return transaction, nil
}

func (uc *walletUseCase) GetTransactionHistory(userID uuid.UUID, page, pageSize int) ([]models.Transaction, int64, error) {
	// Validate page and pageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	return uc.repos.Transaction.GetByUserID(userID, pageSize, offset)
}

// Reconciliation Use Case Implementation

func (uc *reconciliationUseCase) RunReconciliation() ([]models.ReconciliationResult, error) {
	log.Println("Starting reconciliation process...")

	// Get all wallets
	wallets, err := uc.repos.Wallet.GetAllWallets()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallets: %w", err)
	}

	var results []models.ReconciliationResult

	for _, wallet := range wallets {
		// Calculate actual balance from transactions
		calculatedBalance, err := uc.repos.Transaction.GetUserTransactionSum(wallet.UserID)
		if err != nil {
			log.Printf("Failed to calculate balance for user %s: %v", wallet.UserID, err)
			continue
		}

		// Compare with stored balance
		difference := wallet.Balance - calculatedBalance
		hasMismatch := difference != 0

		result := models.ReconciliationResult{
			UserID:            wallet.UserID,
			StoredBalance:     wallet.Balance,
			CalculatedBalance: calculatedBalance,
			Difference:        difference,
			HasMismatch:       hasMismatch,
			CheckedAt:         time.Now(),
		}

		results = append(results, result)

		// Log mismatches
		if hasMismatch {
			log.Printf("MISMATCH DETECTED - User: %s, Stored: %d, Calculated: %d, Difference: %d",
				wallet.UserID, wallet.Balance, calculatedBalance, difference)
		}
	}

	log.Printf("Reconciliation completed. Checked %d wallets, found %d mismatches",
		len(results), countMismatches(results))

	return results, nil
}

// Helper function to count mismatches
func countMismatches(results []models.ReconciliationResult) int {
	count := 0
	for _, result := range results {
		if result.HasMismatch {
			count++
		}
	}
	return count
}
