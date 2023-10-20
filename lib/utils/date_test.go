package utils

import (
	"testing"
	"time"
)

func TestDateType(t *testing.T) {
	var table = []struct {
		When  Date
		Year  int
		Month time.Month
		Day   int
	}{
		{Date{2023, 10, 1}, 2023, 9, 30},
		{Date{2023, 1, 1}, 2022, 12, 31},
		{Date{2023, 3, 1}, 2023, 2, 28},
	}

	for _, tt := range table {
		var yes = tt.When.Yesterday()

		if yes.Year != tt.Year {
			t.Errorf("expected year=%v, got %v", tt.Year, yes.Year)
		}

		if yes.Month != tt.Month {
			t.Errorf("expected month=%v, got %v", tt.Month, yes.Month)
		}

		if yes.Day != tt.Day {
			t.Errorf("expected day=%v, got %v", tt.Day, yes.Day)
		}
	}
}
