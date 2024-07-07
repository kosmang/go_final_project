package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func getTasks(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodGet {
	// 	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	return
	// }

	// search := r.URL.Query().Get("search")
	// var tasks []map[string]string
	// var rows *sql.Rows
	// var err error

	// if search != "" {
	// 	search = "%" + strings.ToLower(search) + "%"
	// 	rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE LOWER(title) LIKE ? OR LOWER(comment) LIKE ? ORDER BY date LIMIT 50", search, search)
	// } else {
	// 	rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT 50")
	// }

	// if err != nil {
	// 	http.Error(w, "Failed to query tasks: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// defer rows.Close()

	// for rows.Next() {
	// 	var id int
	// 	var date, title, comment, repeat string
	// 	err := rows.Scan(&id, &date, &title, &comment, &repeat)
	// 	if err != nil {
	// 		http.Error(w, "Failed to scan task: "+err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}

	// 	task := map[string]string{
	// 		"id":      fmt.Sprint(id),
	// 		"date":    date,
	// 		"title":   title,
	// 		"comment": comment,
	// 		"repeat":  repeat,
	// 	}
	// 	tasks = append(tasks, task)
	// }

	// if tasks == nil {
	// 	tasks = []map[string]string{}
	// }

	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks}); err != nil {
	// 	http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	// }
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
