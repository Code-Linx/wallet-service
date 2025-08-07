package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	App      AppConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string
	Port string
}

// AppConfig holds application configuration
type AppConfig struct {
	Env             string
	JWTSecret       string
	DefaultPageSize int
	MaxPageSize     int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	defaultPageSize := 10
	maxPageSize := 100

	if val := os.Getenv("DEFAULT_PAGE_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			defaultPageSize = parsed
		}
	}

	if val := os.Getenv("MAX_PAGE_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxPageSize = parsed
		}
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "wallet_service"),
		},
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		App: AppConfig{
			Env:             getEnv("APP_ENV", "development"),
			JWTSecret:       getEnv("JWT_SECRET", "default_secret"),
			DefaultPageSize: defaultPageSize,
			MaxPageSize:     maxPageSize,
		},
	}

	// Validate required fields
	if config.Database.Name == "" {
		return nil, fmt.Errorf("database name is required")
	}

	return config, nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
	)
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
