package task

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	dbVar "github.com/kosmang/go_final_project/cmd/database"
	nd "github.com/kosmang/go_final_project/pkg/nextdate"
	_ "modernc.org/sqlite"
)

type task struct {
	ID      string `json:"id,omitempty"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

const QueryLimit = 50

func HandleGetTasks(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	var tasks []map[string]string
	var rows *sql.Rows
	var err error

	err = dbVar.DB.Ping()
	if err != nil {
		fmt.Println("Database connection failed:", err)
	} else {
		fmt.Println("Database connection successful")
	}

	if search != "" {
		if parsedDate, err := time.Parse("02.01.2006", search); err == nil {
			// Если search соответствует формату даты
			formattedDate := parsedDate.Format(nd.DateFormat)
			rows, err = dbVar.DB.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT ?", formattedDate, QueryLimit)
		} else {
			// Поиск по заголовку или комментарию (регистронезависимо)
			searchPattern := "%" + search + "%"
			rows, err = dbVar.DB.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE :search OR comment LIKE :search ORDER BY date LIMIT ?", sql.Named("search", searchPattern), QueryLimit)
		}
	} else {
		rows, err = dbVar.DB.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?", QueryLimit)
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

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed during execution query: "+err.Error(), http.StatusInternalServerError)
	}

	if tasks == nil {
		tasks = []map[string]string{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks}); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func HandleGetTask(w http.ResponseWriter, r *http.Request) {
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
	err := dbVar.DB.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).
		Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return task, err
	}
	return task, nil
}

func HandleTask(w http.ResponseWriter, r *http.Request) {
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
		_, err = w.Write(jsonResponse)
		if err != nil {
			log.Println(err)
		}
		return
	}

	if t.Date != "" {
		tmpDate, err := time.Parse(nd.DateFormat, t.Date)
		if err != nil {
			errorText := "Неправильный формат времени"
			response := map[string]interface{}{"error": errorText}
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write(jsonResponse)
			if err != nil {
				log.Println(err)
			}
			return
		}
		if !tmpDate.After(time.Now()) {
			t.Date = time.Now().Format(nd.DateFormat)
		}
	} else {
		t.Date = time.Now().Format(nd.DateFormat)
	}

	if t.Repeat != "" && !nd.IsValidRepeatRule(t.Repeat) {
		errorText := "Не корректный формат правил повторения"
		response := map[string]interface{}{"error": errorText}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(jsonResponse)
		if err != nil {
			log.Println(err)
		}
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
	_, err = w.Write(jsonResponse)
	if err != nil {
		log.Println(err)
	}
}

func insertTask(t task) (string, error) {
	stmt, err := dbVar.DB.Prepare("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)")
	if err != nil {
		return "temp error", err
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

func HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var t task
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if t.ID == "" {
		http.Error(w, `{"error": "Invalid task ID"}`, http.StatusBadRequest)
		return
	}

	_, err = time.Parse(nd.DateFormat, t.Date)
	if err != nil {
		http.Error(w, `{"error": "Invalid date format"}`, http.StatusBadRequest)
		return
	}

	err = updateTask(t)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte(`{}`))
	if err != nil {
		log.Println(err)
	}
}

func updateTask(t task) error {
	tempID, err := strconv.Atoi(t.ID)
	if err != nil {
		return errors.New(`{"error": "cannot conversation to int64"}`)
	}
	if !nd.IsValidRepeatRule(t.Repeat) {
		return errors.New("")
	}
	if t.Title == "" {
		return errors.New(`{"error": "Не указан заголовок задачи"}`)
	}
	result, err := dbVar.DB.Exec("UPDATE scheduler SET title=?, comment=?, repeat=?, date=? WHERE id=?",
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

func HandleTaskDone(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	var t task
	err := dbVar.DB.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Ошибка запроса к базе данных"}`, http.StatusInternalServerError)
		}
		return
	}

	if t.Repeat == "" {
		_, err = dbVar.DB.Exec("DELETE FROM scheduler WHERE id = ?", id)
	} else {
		nextDate, err := nd.NextDate(time.Now(), t.Date, t.Repeat)
		if err != nil {
			http.Error(w, `{"error": "Ошибка определения следующей даты выполнения задачи"}`, http.StatusInternalServerError)
			return
		}
		_, err = dbVar.DB.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, id)
	}
	if err != nil {
		http.Error(w, `{"error": "Ошибка обновления задачи"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte(`{}`))
	if err != nil {
		log.Println(err)
	}
}

func HandleTaskDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}
	if _, err := strconv.Atoi(id); err != nil {
		http.Error(w, `{"error": "Некорректный идентификатор"}`, http.StatusBadRequest)
		return
	}

	_, err := dbVar.DB.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Ошибка удаления задачи"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte(`{}`))
	if err != nil {
		log.Println(err)
	}
}
