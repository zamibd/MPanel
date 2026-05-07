package migration

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/zamibd/MPanel/config"
	"github.com/zamibd/MPanel/database/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func MigrateDb() {
	dsn := config.GetDBDSN()

	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatal("Failed to parse database DSN: ", err)
		return
	}

	sqlDB := stdlib.OpenDB(*connConfig)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}))
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
		return
	}

	// On a fresh database no tables exist yet. Run AutoMigrate first so that
	// all tables are created before we query or mutate them.
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
		log.Fatal("AutoMigrate failed: ", err)
		return
	}

	// Read the stored schema version (safe now that settings table exists).
	dbVersion := ""
	db.Raw("SELECT value FROM settings WHERE key = ?", "version").Scan(&dbVersion)

	currentVersion := config.GetVersion()
	fmt.Println("Current version:", currentVersion, "\nDatabase version:", dbVersion)

	if currentVersion == dbVersion {
		fmt.Println("Database is up to date, no need to migrate")
		return
	}

	fmt.Println("Start migrating database...")

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Before 1.2
	if dbVersion == "" {
		err = to1_1(tx)
		if err != nil {
			tx.Rollback()
			log.Fatal("Migration to 1.1 failed: ", err)
			return
		}
		err = to1_2(tx)
		if err != nil {
			tx.Rollback()
			log.Fatal("Migration to 1.2 failed: ", err)
			return
		}
		dbVersion = "1.2"
	}

	// Before 1.3
	if dbVersion[0:3] == "1.2" {
		err = to1_3(tx)
		if err != nil {
			tx.Rollback()
			log.Fatal("Migration to 1.3 failed: ", err)
			return
		}
	}

	// Set version
	err = tx.Exec("UPDATE settings SET value = ? WHERE key = ?", currentVersion, "version").Error
	if err != nil {
		tx.Rollback()
		log.Fatal("Update version failed: ", err)
		return
	}

	tx.Commit()
	fmt.Println("Migration done!")
}
