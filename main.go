package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func main() {
	r := chi.NewRouter()
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

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

	db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

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

		_, err = db.Exec(createTable)
		if err != nil {
			log.Fatal(err)
		}
	}

	webDir := "./web"
	fileServer := http.FileServer(http.Dir(webDir))
	r.Handle("/*", fileServer)

	r.Get("/api/nextdate", handleNextDate)

	r.Get("/api/tasks", handleGetTasks)
	r.Get("/api/task", handleGetTask)
	r.Post("/api/task", handleTask)
	r.Put("/api/task", handleUpdateTask)

	r.Post("/api/task/done", handleTaskDone)
	r.Delete("/api/task", handleTaskDelete)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
