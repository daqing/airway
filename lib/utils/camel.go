package utils

import (
	"strings"

	"github.com/iancoleman/strcase"
)

func ToCamel(s string) string {
	var camel = strcase.ToCamel(s)

	return strings.Replace(camel, "Uuid", "UUID", -1)
}
