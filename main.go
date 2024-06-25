package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	// Получение значения переменной окружения TODO_PORT
	port := os.Getenv("TODO_PORT")
	if port == "" {
		// Если переменная окружения не установлена, использовать порт по умолчанию
		port = "7540"
	}

	// Директория с веб-файлами
	webDir := "./web"

	// Установим обработчик для возврата файлов из webDir
	fileServer := http.FileServer(http.Dir(webDir))
	r.Handle("/*", fileServer)

	// Запуск веб-сервера
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
