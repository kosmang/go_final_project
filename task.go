package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type task struct {
	ID      string `json:"id,omitempty"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

func handleGetTasks(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	var tasks []map[string]string
	var rows *sql.Rows
	var err error

	if search != "" {
		if parsedDate, err := time.Parse("02.01.2006", search); err == nil {
			// Если search соответствует формату даты
			formattedDate := parsedDate.Format("20060102")
			rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT 50", formattedDate)
		} else {
			// Поиск по заголовку или комментарию (регистронезависимо)
			searchPattern := "%" + search + "%"
			rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? COLLATE NOCASE OR comment LIKE ? COLLATE NOCASE ORDER BY date LIMIT 50", searchPattern, searchPattern)
		}
	} else {
		rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT 50")
	}

	if err != nil {
		http.Error(w, "Failed to query tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var date, title, comment, repeat string
		err := rows.Scan(&id, &date, &title, &comment, &repeat)
		if err != nil {
			http.Error(w, "Failed to scan task: "+err.Error(), http.StatusInternalServerError)
			return
		}

		task := map[string]string{
			"id":      fmt.Sprint(id),
			"date":    date,
			"title":   title,
			"comment": comment,
			"repeat":  repeat,
		}
		tasks = append(tasks, task)
	}

	if tasks == nil {
		tasks = []map[string]string{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks}); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	task, err := getTaskByID(id)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func getTaskByID(id string) (task, error) {
	var task task
	// Perform a database query to fetch task details based on the ID
	// Example query: SELECT * FROM scheduler WHERE id = ?
	err := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).
		Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return task, err
	}
	return task, nil
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

func insertTask(t task) (string, error) {
	stmt, err := db.Prepare("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)")
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	res, err := stmt.Exec(t.Date, t.Title, t.Comment, t.Repeat)
	if err != nil {
		return "", err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}

	return strconv.Itoa(int(id)), nil
}

func isValidRepeatRule(rule string) bool {
	if rule == "" {
		return true
	}

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

func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var t task
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate task ID
	if t.ID == "" {
		http.Error(w, `{"error": "Invalid task ID"}`, http.StatusBadRequest)
		return
	}

	// Validate date format
	_, err = time.Parse("20060102", t.Date)
	if err != nil {
		http.Error(w, `{"error": "Invalid date format"}`, http.StatusBadRequest)
		return
	}

	// Validate and update task in the database
	err = updateTask(t)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

// Функция для обновления задачи в базе данных
func updateTask(t task) error {
	tempID, err := strconv.Atoi(t.ID)
	if err != nil {
		return errors.New(`{"error": "cannot conversation to int64"}`)
	}
	if !isValidRepeatRule(t.Repeat) {
		return errors.New("")
	}
	if t.Title == "" {
		return errors.New(`{"error": "Не указан заголовок задачи"}`)
	}
	result, err := db.Exec("UPDATE scheduler SET title=?, comment=?, repeat=?, date=? WHERE id=?",
		t.Title, t.Comment, t.Repeat, t.Date, tempID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
