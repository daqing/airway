package ext_date_api

import "time"

func GetCurrentDate() string {
	now := time.Now()

	return now.Format("2006-01-02T15:04:05")
}
