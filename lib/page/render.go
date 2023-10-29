package page

import (
	"bytes"
	"fmt"
	"os"
)

var emptyBytes = []byte{}

func Render(template string, data map[string]any) ([]byte, error) {
	var pwd = os.Getenv("AIRWAY_PWD")

	path := fmt.Sprintf("%s/app/templates/%s.amber", pwd, template)

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
