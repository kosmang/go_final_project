package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func init() {
	dbPath := os.Getenv("TODO_DBFILE")
	if dbPath == "" {
		dbPath = "scheduler.db"
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		return
	}
	dbFile := filepath.Join(currentDir, dbPath)

	_, err = os.Stat(dbFile)
	install := os.IsNotExist(err)

	DB, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	if install {
		createTable := `
		CREATE TABLE IF NOT EXISTS scheduler(
    		id INTEGER PRIMARY KEY AUTOINCREMENT,
    		date CHAR(8) NOT NULL DEFAULT "",
    		title VARCHAR(64) NOT NULL DEFAULT "",
    		comment TEXT,
    		repeat VARCHAR(128) NOT NULL DEFAULT ""
		);
		CREATE INDEX schedule_date ON scheduler (date);
		`

		_, err = DB.Exec(createTable)
		if err != nil {
			log.Fatal(err)
		}
	}
}
