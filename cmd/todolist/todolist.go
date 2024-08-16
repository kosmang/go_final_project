package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/kosmang/go_final_project/cmd/database"
	"github.com/kosmang/go_final_project/internal/task"
	"github.com/kosmang/go_final_project/pkg/nextdate"
)

func main() {
	r := chi.NewRouter()
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	webDir := "./web"
	fileServer := http.FileServer(http.Dir(webDir))
	r.Handle("/*", fileServer)

	r.Get("/api/nextdate", nextdate.HandleNextDate)

	r.Get("/api/tasks", task.HandleGetTasks)
	r.Get("/api/task", task.HandleGetTask)
	r.Post("/api/task", task.HandleTask)
	r.Put("/api/task", task.HandleUpdateTask)

	r.Post("/api/task/done", task.HandleTaskDone)
	r.Delete("/api/task", task.HandleTaskDelete)

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
	defer database.DB.Close()
}
