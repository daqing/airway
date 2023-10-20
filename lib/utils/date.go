package utils

import "time"

type Date struct {
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Day   int        `json:"day"`
}

func Today() Date {
	t := time.Now()

	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}
}

func (d Date) Yesterday() Date {
	t := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.Local)
	y := t.AddDate(0, 0, -1)

	return Date{y.Year(), y.Month(), y.Day()}
}
