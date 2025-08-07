# Wallet Service API

A RESTful wallet service API built with Go, Gin, and MySQL that provides wallet operations with reconciliation logic.

## Features

- ✅ User creation and management
- ✅ Wallet funding operations
- ✅ Wallet withdrawal operations (with idempotency)
- ✅ Funds transfer between users
- ✅ Transaction history with pagination
- ✅ Reconciliation system
- ✅ Concurrent-safe operations
- ✅ Clean Architecture implementation
- ✅ Comprehensive testing

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: MySQL 8.0+
- **ORM**: GORM
- **Testing**: Testify
- **Architecture**: Clean Architecture

## Project Structure

```
wallet-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration management
│   ├── handlers/
│   │   ├── handlers.go            # HTTP handlers
│   │   └── router.go              # Route definitions
│   ├── usecases/
│   │   └── usecases.go           # Business logic
│   ├── repositories/
│   │   └── repositories.go       # Data access layer
│   ├── models/
│   │   └── models.go             # Data models
│   └── middleware/
│       └── middleware.go         # HTTP middleware
├── pkg/
│   └── database/
│       └── database.go           # Database connection
├── tests/
│   ├── unit/
│   │   └── usecases_test.go      # Unit tests
│   └── integration/
│       └── api_test.go           # Integration tests
├── docs/
├── .env                          # Environment variables
├── .env.example                  # Environment template
├── go.mod
├── go.sum
└── README.md
```

## Setup Instructions

### Prerequisites

- Go 1.21 or higher
- MySQL 8.0 or higher
- MySQL Workbench (optional, for database management)

### 1. Clone the Repository

```bash
git clone https://github.com/Code-Linx/wallet-service
cd wallet-service
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Database Setup

#### Using MySQL Workbench:

1. Open MySQL Workbench
2. Connect to your MySQL server
3. Create a new database:

```sql
CREATE DATABASE wallet_service CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

#### Using Command Line:

```bash
mysql -u root -p
CREATE DATABASE wallet_service CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
EXIT;
```

### 4. Environment Configuration

Copy the example environment file and configure it:

```bash
cp .env.example .env
```

Edit `.env` with your actual database credentials:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_mysql_password
DB_NAME=wallet_service

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost

# Application Configuration
APP_ENV=development
JWT_SECRET=your_jwt_secret_key_here

# Pagination
DEFAULT_PAGE_SIZE=10
MAX_PAGE_SIZE=100
```

### 5. Run the Application

```bash
go run cmd/server/main.go
```

You should see output like:

```
2024/08/07 15:30:00 Database connection established successfully
2024/08/07 15:30:00 Running database migrations...
2024/08/07 15:30:00 Database migrations completed successfully
2024/08/07 15:30:00 Starting server on localhost:8080
2024/08/07 15:30:00 Environment: development
2024/08/07 15:30:00 Database: wallet_service
```

## API Documentation

### Base URL

```
http://localhost:8080/api/v1
```

### Endpoints

#### 1. Health Check

```http
GET /health
```

**Response:**

```json
{
  "status": "healthy",
  "service": "wallet-service",
  "version": "1.0.0"
}
```

#### 2. Create User

```http
POST /api/v1/users
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com"
}
```

**Response:**

```json
{
  "success": true,
  "message": "User created successfully",
  "data": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com",
    "created_at": "2024-08-07T15:30:00Z",
    "wallet": {
      "id": "uuid",
      "user_id": "uuid",
      "balance": 0,
      "created_at": "2024-08-07T15:30:00Z"
    }
  }
}
```

#### 3. Get User

```http
GET /api/v1/users/{user_id}
```

#### 4. Fund Wallet

```http
POST /api/v1/users/{user_id}/fund
Content-Type: application/json

{
  "amount": 10000,
  "reference": "fund_001"
}
```

**Note**: Amount is in smallest currency unit (cents for USD)

#### 5. Withdraw Funds

```http
POST /api/v1/users/{user_id}/withdraw
Content-Type: application/json

{
  "amount": 2000,
  "reference": "withdraw_001"
}
```

**Features**:

- ✅ Idempotent (same reference won't create duplicate transactions)
- ✅ Balance validation (prevents negative balances)

#### 6. Transfer Funds

```http
POST /api/v1/users/{from_user_id}/transfer
Content-Type: application/json

{
  "to_user_id": "recipient_user_id",
  "amount": 3000,
  "reference": "transfer_001"
}
```

**Features**:

- ✅ Idempotent
- ✅ Atomic transactions
- ✅ Balance validation
- ✅ User validation

#### 7. Transaction History

```http
GET /api/v1/users/{user_id}/transactions?page=1&page_size=10
```

**Query Parameters**:

- `page` (default: 1, min: 1)
- `page_size` (default: 10, min: 1, max: 100)

**Response:**

```json
{
  "success": true,
  "message": "Transactions retrieved successfully",
  "data": {
    "data": [...],
    "page": 1,
    "page_size": 10,
    "total": 25,
    "total_pages": 3
  }
}
```

#### 8. Run Reconciliation

```http
POST /api/v1/reconciliation/run
```

**Response:**

```json
{
  "success": true,
  "message": "Reconciliation completed successfully",
  "data": [
    {
      "user_id": "uuid",
      "stored_balance": 5000,
      "calculated_balance": 5000,
      "difference": 0,
      "has_mismatch": false,
      "checked_at": "2024-08-07T15:30:00Z"
    }
  ]
}
```

## Testing

### Run Unit Tests

```bash
go test ./tests/unit/... -v
```

### Run Integration Tests

```bash
# Create test database first
mysql -u root -p -e "CREATE DATABASE wallet_service_test;"

# Run integration tests
go test ./tests/integration/... -v
```

### Run All Tests

```bash
go test ./... -v
```

## Testing with Postman

Import the provided Postman collection and:

1. **Create Users**: Create two test users
2. **Fund Wallets**: Add funds to user wallets
3. **Test Withdrawals**: Test withdrawal operations
4. **Test Transfers**: Transfer funds between users
5. **View History**: Check transaction history
6. **Test Idempotency**: Try duplicate operations
7. **Run Reconciliation**: Verify wallet balances

## Key Features Implemented

### 1. Idempotency

- Withdraw and transfer operations use unique references
- Duplicate requests with same reference return original transaction
- Prevents accidental duplicate operations

### 2. Concurrency Safety

- Database transactions ensure atomic operations
- Proper locking mechanisms prevent race conditions
- All wallet operations are thread-safe

### 3. Clean Architecture

- **Handlers**: HTTP request/response handling
- **Use Cases**: Business logic implementation
- **Repositories**: Data access abstraction
- **Models**: Data structures and validation

### 4. Error Handling

- Comprehensive error responses
- Proper HTTP status codes
- Detailed error messages for debugging

### 5. Reconciliation

- Compares stored balances with calculated balances
- Identifies and logs discrepancies
- Can be run manually or scheduled

## Configuration

All configuration is managed through environment variables:

- **Database**: Connection settings
- **Server**: Host and port configuration
- **Application**: Environment and secrets
- **Pagination**: Default and maximum page sizes

## Production Considerations

1. **Security**:

   - Use proper JWT secrets
   - Implement rate limiting
   - Add input validation middleware
   - Use HTTPS in production

2. **Performance**:

   - Add database connection pooling
   - Implement caching for frequent queries
   - Add database indexes for optimization
   - Monitor query performance

3. **Monitoring**:

   - Add structured logging
   - Implement health checks
   - Add metrics collection
   - Set up alerting

4. **Deployment**:
   - Use Docker containers
   - Set up CI/CD pipelines
   - Configure auto-scaling
   - Implement database migrations

## Troubleshooting

### Common Issues

1. **Database Connection Error**:

   - Check MySQL is running
   - Verify credentials in `.env`
   - Ensure database exists

2. **Port Already in Use**:

   - Change `SERVER_PORT` in `.env`
   - Kill existing process: `lsof -ti:8080 | xargs kill`

3. **Migration Errors**:
   - Check database permissions
   - Verify MySQL version compatibility
   - Clear and recreate database if needed

### Logs

The application provides detailed logs for:

- Database connections
- Transaction operations
- Reconciliation results
- Error conditions

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details
