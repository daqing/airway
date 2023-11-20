package utils

import (
	"fmt"
	"time"
)

func TimeAgo(target time.Time) string {
	return timeAgo(time.Now(), target)
}

func timeAgo(base, target time.Time) string {
	since := base.Sub(target).Abs()
	sec := since.Seconds()

	if sec <= 60 {
		return "1 分钟前"
	}

	min := int(since.Minutes())

	if sec <= 3600 {
		return fmt.Sprintf("%d 分钟前", min)
	}

	hours := int(since.Hours())
	min = min % 60

	if hours < 24 {
		return fmt.Sprintf("%d 小时 %d 分钟前", hours, min)
	}

	days := int(hours / 24)

	if days < 7 {
		return fmt.Sprintf("%d 天前", days)
	}

	weeks := int(days / 7)
	if weeks < 4 {
		return fmt.Sprintf("%d 周前", weeks)
	}

	return target.Format("2006-01-02 15:04")
}
