package utils

import (
	"testing"
	"time"
)

func TestTimeAgo(t *testing.T) {

	tests := []struct {
		Base     time.Time
		Target   time.Time
		Expected string
	}{
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 11, 20, 19, 25, 34, 0, time.Local),
			"1 分钟前",
		},
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 11, 20, 19, 28, 2, 0, time.Local),
			"3 分钟前",
		},
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 11, 20, 20, 30, 2, 0, time.Local),
			"1 小时 5 分钟前",
		},
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 11, 20, 21, 32, 2, 0, time.Local),
			"2 小时 7 分钟前",
		},
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 11, 21, 19, 32, 2, 0, time.Local),
			"1 天前",
		},
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 11, 28, 19, 32, 2, 0, time.Local),
			"1 周前",
		},
		{
			time.Date(2023, 11, 20, 19, 25, 0, 0, time.Local),
			time.Date(2023, 12, 28, 19, 32, 2, 0, time.Local),
			"2023-12-28 19:32",
		},
	}

	for _, test := range tests {
		actual := timeAgo(test.Base, test.Target)

		if actual != test.Expected {
			t.Errorf("Expected %v, got %v", test.Expected, actual)
		}
	}
}
