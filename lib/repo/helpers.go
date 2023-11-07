package repo

import (
	"strings"

	"github.com/iancoleman/strcase"
)

func ToCamel(s string) string {
	var camel = strcase.ToCamel(s)

	r := strings.Replace(camel, "Uuid", "UUID", -1)

	return strings.Replace(r, "Url", "URL", -1)
}
