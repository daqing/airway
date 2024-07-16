package services

import "time"

func Now() string {
	now := time.Now()

	return now.Format("2006-01-02T15:04:05")
}
