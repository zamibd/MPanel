package database

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/zamibd/MPanel/config"
	"github.com/zamibd/MPanel/database/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func initUser() error {
	var count int64
	err := db.Model(&model.User{}).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		user := &model.User{
			Username: "admin",
			Password: "admin",
		}
		return db.Create(user).Error
	}
	return nil
}

// OpenDB opens a PostgreSQL connection using pgx/v5 stdlib and wires it into GORM.
// The dsn parameter is a libpq-style connection string, e.g.:
//
//	"host=localhost port=5432 user=mpanel password=secret dbname=mpanel sslmode=disable"
func OpenDB(dsn string) error {
	var gormLogger logger.Interface
	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	// Parse the DSN with pgx so we get full pgx connection control.
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}

	// Open a *sql.DB backed by pgx/v5 stdlib.
	sqlDB := stdlib.OpenDB(*connConfig)

	// Hand the already-opened *sql.DB to GORM so it doesn't re-parse the DSN.
	db, err = gorm.Open(
		postgres.New(postgres.Config{Conn: sqlDB}),
		&gorm.Config{Logger: gormLogger},
	)
	if err != nil {
		return err
	}

	// Configure the underlying connection pool.
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	if config.IsDebug() {
		db = db.Debug()
	}
	return nil
}

func InitDB(dsn string) error {
	err := OpenDB(dsn)
	if err != nil {
		return err
	}

	// Drop problematic constraints created by earlier versions
	if db.Migrator().HasConstraint(&model.Inbound{}, "fk_inbounds_tls") {
		db.Migrator().DropConstraint(&model.Inbound{}, "fk_inbounds_tls")
	}
	if db.Migrator().HasConstraint(&model.Service{}, "fk_services_tls") {
		db.Migrator().DropConstraint(&model.Service{}, "fk_services_tls")
	}

	// Default Outbounds
	if !db.Migrator().HasTable(&model.Outbound{}) {
		db.Migrator().CreateTable(&model.Outbound{})
		defaultOutbound := []model.Outbound{
			{Type: "direct", Tag: "direct", Options: json.RawMessage(`{}`)},
		}
		db.Create(&defaultOutbound)
	}

	err = db.AutoMigrate(
		&model.Setting{},
		&model.Tls{},
		&model.Inbound{},
		&model.Outbound{},
		&model.Service{},
		&model.Endpoint{},
		&model.User{},
		&model.Tokens{},
		&model.Stats{},
		&model.Client{},
		&model.Changes{},
	)
	if err != nil {
		return err
	}
	return initUser()
}

func GetDB() *gorm.DB {
	return db
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
