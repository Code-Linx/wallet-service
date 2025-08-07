package repositories

import (
	"github.com/Code-Linx/wallet-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository interface defines user repository methods
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uuid.UUID) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetAll() ([]models.User, error)
}

// WalletRepository interface defines wallet repository methods
type WalletRepository interface {
	Create(wallet *models.Wallet) error
	GetByUserID(userID uuid.UUID) (*models.Wallet, error)
	UpdateBalance(userID uuid.UUID, newBalance int64) error
	GetByID(id uuid.UUID) (*models.Wallet, error)
	GetAllWallets() ([]models.Wallet, error)
}

// TransactionRepository interface defines transaction repository methods
type TransactionRepository interface {
	Create(transaction *models.Transaction) error
	GetByReference(reference string) (*models.Transaction, error)
	GetByUserID(userID uuid.UUID, limit, offset int) ([]models.Transaction, int64, error)
	GetUserTransactionSum(userID uuid.UUID) (int64, error)
	Update(transaction *models.Transaction) error
}

// userRepository implements UserRepository
type userRepository struct {
	db *gorm.DB
}

// walletRepository implements WalletRepository
type walletRepository struct {
	db *gorm.DB
}

// transactionRepository implements TransactionRepository
type transactionRepository struct {
	db *gorm.DB
}

// Repositories holds all repository instances
type Repositories struct {
	User        UserRepository
	Wallet      WalletRepository
	Transaction TransactionRepository
	DB          *gorm.DB
}

// NewRepositories creates new repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:        &userRepository{db: db},
		Wallet:      &walletRepository{db: db},
		Transaction: &transactionRepository{db: db},
		DB:          db,
	}
}

// User Repository Implementation

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Wallet").First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Wallet").First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Preload("Wallet").Find(&users).Error
	return users, err
}

// Wallet Repository Implementation

func (r *walletRepository) Create(wallet *models.Wallet) error {
	return r.db.Create(wallet).Error
}

func (r *walletRepository) GetByUserID(userID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.First(&wallet, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) UpdateBalance(userID uuid.UUID, newBalance int64) error {
	return r.db.Model(&models.Wallet{}).Where("user_id = ?", userID).Update("balance", newBalance).Error
}

func (r *walletRepository) GetByID(id uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.First(&wallet, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) GetAllWallets() ([]models.Wallet, error) {
	var wallets []models.Wallet
	err := r.db.Preload("User").Find(&wallets).Error
	return wallets, err
}

// Transaction Repository Implementation

func (r *transactionRepository) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *transactionRepository) GetByReference(reference string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.First(&transaction, "reference = ?", reference).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionRepository) GetByUserID(userID uuid.UUID, limit, offset int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	// Get total count
	if err := r.db.Model(&models.Transaction{}).Where("user_id = ? OR from_user_id = ? OR to_user_id = ?", userID, userID, userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.Where("user_id = ? OR from_user_id = ? OR to_user_id = ?", userID, userID, userID).
		Preload("User").
		Preload("FromUser").
		Preload("ToUser").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	return transactions, total, err
}

func (r *transactionRepository) GetUserTransactionSum(userID uuid.UUID) (int64, error) {
	var result struct {
		Sum int64
	}

	// Calculate sum: credits minus debits for the user
	query := `
		SELECT COALESCE(SUM(
			CASE 
				WHEN type = 'credit' THEN amount
				WHEN type = 'debit' AND user_id = ? THEN -amount
				WHEN type = 'transfer' AND from_user_id = ? THEN -amount
				WHEN type = 'transfer' AND to_user_id = ? THEN amount
				ELSE 0
			END
		), 0) as sum
		FROM transactions 
		WHERE (user_id = ? OR from_user_id = ? OR to_user_id = ?) 
		AND status = 'completed'
	`

	err := r.db.Raw(query, userID, userID, userID, userID, userID, userID).Scan(&result).Error
	return result.Sum, err
}

func (r *transactionRepository) Update(transaction *models.Transaction) error {
	return r.db.Save(transaction).Error
}

// BeginTransaction starts a new database transaction
func (repos *Repositories) BeginTransaction() *gorm.DB {
	return repos.DB.Begin()
}

// WithTransaction creates repositories with transaction context
func (repos *Repositories) WithTransaction(tx *gorm.DB) *Repositories {
	return &Repositories{
		User:        &userRepository{db: tx},
		Wallet:      &walletRepository{db: tx},
		Transaction: &transactionRepository{db: tx},
		DB:          tx,
	}
}
