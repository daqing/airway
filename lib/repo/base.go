package repo

import (
	"fmt"
	"strings"
)

func buildCondQuery(conds []KeyValueField) (condQuery string, values []any) {
	if len(conds) == 0 {
		return "1=1", nil
	}

	var dollar int

	var condString = []string{}

	for _, cond := range conds {
		dollar += 1

		part := fmt.Sprintf("%s = $%d", cond.KeyField(), dollar)

		condString = append(condString, part)
		values = append(values, cond.ValueField())
	}

	condQuery = strings.Join(condString, " AND ")

	return
}
