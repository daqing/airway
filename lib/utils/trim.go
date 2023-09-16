package utils

import "strings"

type TrimType int

const (
	None TrimType = iota
	Left
	Right
	Full
)

const cutset = "\t\n\v\f\r "

func Trim(s string, trimType TrimType) string {
	switch trimType {
	case Left:
		return strings.TrimLeft(s, cutset)
	case Right:
		return strings.TrimRight(s, cutset)
	case Full:
		return strings.Trim(s, cutset)
	}

	return s
}

func TrimFull(s string) string {
	return Trim(s, Full)
}
