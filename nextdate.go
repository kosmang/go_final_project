package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ResponseDate struct {
	NextDate string // `json:"next_date,omitempty"`
	Error    string // `json:"error,omitempty"`
}

func handleNextDate(w http.ResponseWriter, r *http.Request) {
	nowStr := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")

	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Invalid now parameter", http.StatusBadRequest)
		return
	}

	nextDate, err := NextDate(now, date, repeat)
	response := ResponseDate{}
	if err != nil {
		response.Error = err.Error()
	} else {
		response.NextDate = nextDate
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(response.NextDate))
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	now = now.Truncate(24 * time.Hour)

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
		next := startDate
		for {
			next = next.AddDate(0, 0, days)
			if next.After(now) {
				return next.Format("20060102"), nil
			}
		}

	case repeat == "y":
		next := startDate
		for {
			next = next.AddDate(1, 0, 0)
			if next.After(now) {
				return next.Format("20060102"), nil
			}
		}

	default:
		return "", errors.New("unsupported repeat rule")
	}
}
