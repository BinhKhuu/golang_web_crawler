package dbstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

const QueryTimeout = 5 * time.Second

type DBStorageService struct {
	DB *sql.DB
}

func GetConnectionString() (string, error) {
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return "", errors.New("DB_USER environment variable is not set")
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return "", errors.New("DB_PASSWORD environment variable is not set")
	}
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return "", errors.New("DB_HOST environment variable is not set")
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		return "", errors.New("DB_PORT environment variable is not set")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return "", errors.New("DB_NAME environment variable is not set")
	}
	dbSslmode := os.Getenv("DB_SSLMODE")
	if dbSslmode == "" {
		return "", errors.New("DB_SSLMODE environment variable is not set")
	}

	hostPort := net.JoinHostPort(dbHost, dbPort)
	return fmt.Sprintf(`postgres://%s:%s@%s/%s?sslmode=%s`, dbUser, dbPassword, hostPort, dbName, dbSslmode), nil
}

func SetupDatabase() (*sql.DB, error) {
	conStr, err := GetConnectionString()
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	if err != nil {
		return nil, fmt.Errorf("failed to load database settings: %w", err)
	}
	conn, err := sql.Open("postgres", conStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return conn, nil
}
