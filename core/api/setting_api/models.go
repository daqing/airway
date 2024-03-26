package setting_api

import "time"

type Setting struct {
	Id int64

	Key string
	Val string

	CreatedAt time.Time
	UpdatedAt time.Time
}
