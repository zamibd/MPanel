package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/zamibd/MPanel/database/model"
	"github.com/zamibd/MPanel/logger"
	"github.com/zamibd/MPanel/util/common"
)

// dbDump is the JSON structure used for portable backup/restore.
type dbDump struct {
	Settings  []model.Setting  `json:"settings"`
	Tls       []model.Tls      `json:"tls"`
	Inbounds  []model.Inbound  `json:"inbounds"`
	Outbounds []model.Outbound `json:"outbounds"`
	Endpoints []model.Endpoint `json:"endpoints"`
	Users     []model.User     `json:"users"`
	Clients   []model.Client   `json:"clients"`
	Stats     []model.Stats    `json:"stats,omitempty"`
	Changes   []model.Changes  `json:"changes,omitempty"`
}

// GetDb exports the current database as a JSON backup.
// The `exclude` parameter is a comma-separated list of table names to skip
// (supported values: "changes", "stats").
func GetDb(exclude string) ([]byte, error) {
	excludeChanges, excludeStats := false, false
	for _, table := range strings.Split(exclude, ",") {
		switch strings.TrimSpace(table) {
		case "changes":
			excludeChanges = true
		case "stats":
			excludeStats = true
		}
	}

	dump := dbDump{}

	if err := db.Model(&model.Setting{}).Find(&dump.Settings).Error; err != nil {
		return nil, fmt.Errorf("scan settings: %w", err)
	}
	if err := db.Model(&model.Tls{}).Find(&dump.Tls).Error; err != nil {
		return nil, fmt.Errorf("scan tls: %w", err)
	}
	if err := db.Model(&model.Inbound{}).Find(&dump.Inbounds).Error; err != nil {
		return nil, fmt.Errorf("scan inbounds: %w", err)
	}
	if err := db.Model(&model.Outbound{}).Find(&dump.Outbounds).Error; err != nil {
		return nil, fmt.Errorf("scan outbounds: %w", err)
	}
	if err := db.Model(&model.Endpoint{}).Find(&dump.Endpoints).Error; err != nil {
		return nil, fmt.Errorf("scan endpoints: %w", err)
	}
	if err := db.Model(&model.User{}).Find(&dump.Users).Error; err != nil {
		return nil, fmt.Errorf("scan users: %w", err)
	}
	if err := db.Model(&model.Client{}).Find(&dump.Clients).Error; err != nil {
		return nil, fmt.Errorf("scan clients: %w", err)
	}
	if !excludeStats {
		if err := db.Model(&model.Stats{}).Find(&dump.Stats).Error; err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
	}
	if !excludeChanges {
		if err := db.Model(&model.Changes{}).Find(&dump.Changes).Error; err != nil {
			return nil, fmt.Errorf("scan changes: %w", err)
		}
	}

	out, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal dump: %w", err)
	}
	return out, nil
}

// ImportDB restores the database from a JSON backup produced by GetDb.
// It truncates all tables and re-inserts every record inside a transaction.
func ImportDB(file io.Reader) error {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return common.NewErrorf("read backup file: %v", err)
	}

	var dump dbDump
	if err := json.Unmarshal(buf.Bytes(), &dump); err != nil {
		return common.NewErrorf("invalid backup format (expected JSON): %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		return common.NewErrorf("begin transaction: %v", tx.Error)
	}

	// Delete in reverse dependency order to avoid FK violations
	tables := []interface{}{
		&model.Changes{}, &model.Stats{}, &model.Tokens{},
		&model.Client{}, &model.User{}, &model.Endpoint{},
		&model.Outbound{}, &model.Inbound{}, &model.Tls{}, &model.Setting{},
	}
	for _, t := range tables {
		if err := tx.Where("1 = 1").Delete(t).Error; err != nil {
			tx.Rollback()
			return common.NewErrorf("truncate table: %v", err)
		}
	}

	var firstErr error
	checkErr := func(err error, label string) {
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("restore %s: %w", label, err)
		}
	}

	if len(dump.Settings) > 0 {
		checkErr(tx.Create(&dump.Settings).Error, "settings")
	}
	if len(dump.Tls) > 0 {
		checkErr(tx.Create(&dump.Tls).Error, "tls")
	}
	if len(dump.Inbounds) > 0 {
		checkErr(tx.Create(&dump.Inbounds).Error, "inbounds")
	}
	if len(dump.Outbounds) > 0 {
		checkErr(tx.Create(&dump.Outbounds).Error, "outbounds")
	}
	if len(dump.Endpoints) > 0 {
		checkErr(tx.Create(&dump.Endpoints).Error, "endpoints")
	}
	if len(dump.Users) > 0 {
		checkErr(tx.Create(&dump.Users).Error, "users")
	}
	if len(dump.Clients) > 0 {
		checkErr(tx.Create(&dump.Clients).Error, "clients")
	}
	if len(dump.Stats) > 0 {
		checkErr(tx.Create(&dump.Stats).Error, "stats")
	}
	if len(dump.Changes) > 0 {
		checkErr(tx.Create(&dump.Changes).Error, "changes")
	}

	if firstErr != nil {
		tx.Rollback()
		return common.NewErrorf("import failed: %v", firstErr)
	}

	if err := tx.Commit().Error; err != nil {
		return common.NewErrorf("commit: %v", err)
	}

	if err := SendSighup(); err != nil {
		return common.NewErrorf("restart app: %v", err)
	}
	return nil
}

// IsSQLiteDB is retained for API compatibility but always returns false
// because this build uses PostgreSQL for storage.
func IsSQLiteDB(_ io.Reader) (bool, error) {
	return false, nil
}

// GetBackupFileName returns a timestamp-stamped filename for download purposes.
func GetBackupFileName() string {
	return fmt.Sprintf("mpanel_backup_%s.json", time.Now().Format("20060102-150405"))
}

// GetDbFilePath is kept so that any shell scripts that call `./mpanel backup` still compile.
func GetDbFilePath() string {
	dir, _ := os.Getwd()
	return fmt.Sprintf("%s/mpanel_backup_%s.json", dir, time.Now().Format("20060102-150405"))
}

func SendSighup() error {
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}

	go func() {
		time.Sleep(3 * time.Second)
		if runtime.GOOS == "windows" {
			err = process.Kill()
		} else {
			err = process.Signal(syscall.SIGHUP)
		}
		if err != nil {
			logger.Error("send signal SIGHUP failed:", err)
		}
	}()
	return nil
}
