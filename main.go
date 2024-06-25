package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
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
	http.Handle("/", fileServer)

	// Запуск веб-сервера
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
