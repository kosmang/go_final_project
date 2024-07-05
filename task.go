package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type task struct {
	ID      int64  `json:"id,omitempty"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

func handleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var t task
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if t.Title == "" {
		errorText := "Не указан заголовок задачи"
		response := map[string]interface{}{"error": errorText}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	if t.Date != "" {
		tmpDate, err := time.Parse("20060102", t.Date)
		if err != nil {
			errorText := "Неправильный формат времени"
			response := map[string]interface{}{"error": errorText}
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jsonResponse)
			return
		}
		if !tmpDate.After(time.Now()) {
			t.Date = time.Now().Format("20060102")
		}
	} else {
		t.Date = time.Now().Format("20060102")
	}

	if t.Repeat != "" && !isValidRepeatRule(t.Repeat) {
		errorText := "Не корректный формат правил повторения"
		response := map[string]interface{}{"error": errorText}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	t.ID, err = insertTask(t)
	if err != nil {
		http.Error(w, "Failed to insert task into database", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{"id": t.ID}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func insertTask(t task) (int64, error) {
	stmt, err := db.Prepare("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(t.Date, t.Title, t.Comment, t.Repeat)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func isValidRepeatRule(rule string) bool {
	if rule == "y" {
		return true
	}

	if strings.HasPrefix(rule, "d ") {
		days, err := strconv.Atoi(strings.TrimPrefix(rule, "d "))
		if err != nil || days < 1 || days > 400 {
			return false
		}
		return true
	}

	return false
}
