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
	if !isValidRepeatRule(repeat) {
		return "", errors.New("incorrect repeat rule")
	}
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

	case strings.HasPrefix(repeat, "w "):
		daysOfWeekStr := strings.TrimPrefix(repeat, "w ")
		daysOfWeekArr := strings.Split(daysOfWeekStr, ",")
		var daysOfWeek []int
		for _, dayStr := range daysOfWeekArr {
			day, err := strconv.Atoi(dayStr)
			if err != nil || day < 1 || day > 7 {
				return "", errors.New("invalid day of week")
			}
			daysOfWeek = append(daysOfWeek, day)
		}
		return getNextWeekday(now, startDate, daysOfWeek)

	case strings.HasPrefix(repeat, "m "):
		return getNextMonthday(now, startDate, strings.TrimPrefix(repeat, "m "))

	default:
		return "", errors.New("unsupported repeat rule")
	}
}

func getNextWeekday(now, startDate time.Time, daysOfWeek []int) (string, error) {
	daysOfWeekMap := make(map[int]bool)
	for _, day := range daysOfWeek {
		daysOfWeekMap[day] = true
	}

	next := startDate
	if next.Before(now) {
		next = now
	}

	for {
		dayOfWeek := int(next.Weekday())
		if dayOfWeek == 0 {
			dayOfWeek = 7
		}
		if daysOfWeekMap[dayOfWeek] && next.After(now) {
			return next.Format("20060102"), nil
		}
		next = next.AddDate(0, 0, 1)
	}
}

func getNextMonthday(now, startDate time.Time, repeat string) (string, error) {
	parts := strings.Split(repeat, " ")
	if len(parts) == 0 {
		return "", errors.New("invalid monthday repeat format")
	}

	daysStr := parts[0]
	monthsStr := ""
	if len(parts) > 1 {
		monthsStr = parts[1]
	}

	daysArr := strings.Split(daysStr, ",")
	var days []int
	for _, dayStr := range daysArr {
		day, err := strconv.Atoi(dayStr)
		if err != nil {
			return "", errors.New("invalid day format")
		}
		days = append(days, day)
	}

	var months []int
	if monthsStr != "" {
		monthsArr := strings.Split(monthsStr, ",")
		for _, monthStr := range monthsArr {
			month, err := strconv.Atoi(monthStr)
			if err != nil {
				return "", errors.New("invalid month format")
			}
			months = append(months, month)
		}
	}

	return findNextMonthday(now, startDate, days, months)
}

func findNextMonthday(now, startDate time.Time, days, months []int) (string, error) {
	next := startDate
	if next.Before(now) {
		next = now
	}

	for {
		year, month, _ := next.Date()
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, next.Location()).Day()
		monthMatches := len(months) == 0 || contains(months, int(month))

		if monthMatches {
			closestNextDate := time.Time{}
			for _, day := range days {
				var nextDate time.Time
				if day > 0 && day <= lastDay {
					nextDate = time.Date(year, month, day, 0, 0, 0, 0, next.Location())
				} else if day == -1 {
					nextDate = time.Date(year, month, lastDay, 0, 0, 0, 0, next.Location())
				} else if day == -2 && lastDay > 1 {
					nextDate = time.Date(year, month, lastDay-1, 0, 0, 0, 0, next.Location())
				}

				if nextDate.After(now) {
					if closestNextDate.IsZero() || nextDate.Before(closestNextDate) {
						closestNextDate = nextDate
					}
				}
			}
			if !closestNextDate.IsZero() {
				return closestNextDate.Format("20060102"), nil
			}
		}

		next = next.AddDate(0, 1, 0)
	}
}

func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
