package config

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed version
var version string

//go:embed name
var name string

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func GetVersion() string {
	return strings.TrimSpace(version)
}

func GetName() string {
	return strings.TrimSpace(name)
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("MPANEL_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

func IsDebug() bool {
	return os.Getenv("MPANEL_DEBUG") == "true"
}

// GetDBDSN returns a PostgreSQL connection DSN built from environment variables:
//   POSTGRES_HOST     (default: postgres)
//   POSTGRES_PORT     (default: 5432)
//   POSTGRES_USER     (default: mpanel)
//   POSTGRES_PASSWORD (required)
//   POSTGRES_DB       (default: mpanel)
//   POSTGRES_SSLMODE  (default: disable)
func GetDBDSN() string {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "postgres"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "mpanel"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "mpanel"
	}
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	tz := os.Getenv("POSTGRES_TIMEZONE")
	if tz == "" {
		tz = "Asia/Dhaka"
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host, port, user, password, dbname, sslmode, tz,
	)
}

// GetDBPath is kept for backward compatibility; it now returns the PostgreSQL DSN.
func GetDBPath() string {
	return GetDBDSN()
}
