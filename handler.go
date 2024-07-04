package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Response struct {
	NextDate string // `json:"next_date,omitempty"`
	Error    string // `json:"error,omitempty"`
}

func handleNextDate(w http.ResponseWriter, r *http.Request) {
	nowStr := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")
	// w.Write([]byte(nowStr + date + repeat))

	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Invalid now parameter", http.StatusBadRequest)
		return
	}
	// w.Write([]byte(now.Format("20060102")))

	nextDate, err := NextDate(now, date, repeat)
	response := Response{}
	if err != nil {
		response.Error = err.Error()
	} else {
		response.NextDate = nextDate
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(response.NextDate))
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(response.NextDate); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	startDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %v", err)
	}

	switch {
	case repeat == "":
		return "", errors.New("repeat rule is empty")

	case strings.HasPrefix(repeat, "d "):
		days, err := strconv.Atoi(strings.TrimPrefix(repeat, "d "))
		if err != nil || days < 1 || days > 400 {
			return "", errors.New("invalid day interval")
		}
		for next := startDate; ; next = next.AddDate(0, 0, days) {
			if next.After(startDate) && next.After(now) {
				return next.Format("20060102"), nil
			}
		}

	case repeat == "y":
		for next := startDate; ; next = next.AddDate(1, 0, 0) {
			if next.After(startDate) && next.After(now) {
				return next.Format("20060102"), nil
			}
		}

	default:
		return "", errors.New("unsupported repeat rule")
	}
}
