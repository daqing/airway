package tmpl

import (
	"bytes"
	"fmt"

	"github.com/daqing/amber"
)

var compiler = amber.New()
var emptyBytes = []byte{}

func Render(prefix string, template string, data map[string]any) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.amber", prefix, template)

	err := compiler.ParseFile(path)
	if err != nil {
		return emptyBytes, err
	}

	tpl, err := compiler.Compile()
	if err != nil {
		return emptyBytes, err
	}

	var out bytes.Buffer
	err = tpl.Execute(&out, data)

	return out.Bytes(), err
}
