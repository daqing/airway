package utils

import (
	"strconv"
	"strings"
)

const API_PREFIX = "airway_"

func GenerateApiToken() string {
	return API_PREFIX + RandomHex(20)
}

func ToArray(str string, sep string) []string {
	return strings.Split(str, sep)
}

func ToIntArray(arr []string) []int {
	var result = make([]int, 0)
	for _, s := range arr {
		i, err := strconv.Atoi(s)
		if err != nil {
			i = 0
		}

		result = append(result, i)
	}

	return result
}
