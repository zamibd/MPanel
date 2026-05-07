package migration

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zamibd/MPanel/database/model"

	"gorm.io/gorm"
)

func migrateClientSchema(db *gorm.DB) error {
	// Use information_schema.columns instead of SQLite's PRAGMA table_info
	type colInfo struct {
		ColumnName string
		DataType   string
	}
	var cols []colInfo
	err := db.Raw(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'clients'
		  AND column_name IN ('config', 'inbounds', 'links')
	`).Scan(&cols).Error
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, col := range cols {
		// In Postgres, old text columns would have data_type = 'text';
		// the new schema uses 'jsonb'. Migrate any residual text values.
		if col.DataType == "text" {
			fmt.Printf("Column %s has type TEXT\n", col.ColumnName)
			oldData := make([]struct {
				Id   uint
				Data string
			}, 0)
			db.Model(model.Client{}).Select("id", col.ColumnName+" as data").Scan(&oldData)
			for _, data := range oldData {
				var newData []byte
				switch col.ColumnName {
				case "inbounds":
					inbounds := strings.Split(data.Data, ",")
					newData, _ = json.MarshalIndent(inbounds, "", "  ")
				case "config":
					jsonData := map[string]interface{}{}
					json.Unmarshal([]byte(data.Data), &jsonData)
					newData, _ = json.MarshalIndent(jsonData, "", "  ")
				case "links":
					jsonData := make([]interface{}, 0)
					json.Unmarshal([]byte(data.Data), &jsonData)
					newData, _ = json.MarshalIndent(jsonData, "", "  ")
				}
				err = db.Model(model.Client{}).Where("id = ?", data.Id).UpdateColumn(col.ColumnName, newData).Error
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deleteOldWebSecret(db *gorm.DB) error {
	return db.Exec("DELETE FROM settings WHERE key = ?", "webSecret").Error
}

func changesObj(db *gorm.DB) error {
	// PostgreSQL: concatenate and cast to bytea instead of SQLite BLOB
	return db.Exec(`
		UPDATE changes
		SET obj = ('"' || obj::text || '"')::bytea
		WHERE actor = ? AND obj::text NOT LIKE ?
	`, "DepleteJob", "\"%\"").Error
}

func to1_1(db *gorm.DB) error {
	err := migrateClientSchema(db)
	if err != nil {
		return err
	}
	err = deleteOldWebSecret(db)
	if err != nil {
		return err
	}
	err = changesObj(db)
	if err != nil {
		return err
	}
	return nil
}
