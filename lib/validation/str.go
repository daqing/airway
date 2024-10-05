package validation

import (
	"fmt"
	"strings"
)

func Do(name string, value string, validators string) error {
	v := &stringValidator{
		Name:       name,
		Value:      value,
		Validators: validators,
	}

	return v.validate()
}

// internal implementation
type stringValidator struct {
	Name       string
	Value      string
	Validators string
}

func (v *stringValidator) validate() error {
	var validators = []*checkingStruct{}

	for _, x := range strings.Split(v.Validators, ",") {
		switch strings.TrimSpace(x) {
		case "required":
			validators = append(validators, &checkingStruct{Name: v.Name, Tip: "required", Fn: notEmpty})
		case "email":
			validators = append(validators, &checkingStruct{Name: v.Name, Tip: "format_error", Fn: isEmail})
		}
	}

	return checking(v.Value, validators...)
}

type checkingStruct struct {
	Name string
	Tip  string
	Fn   checkFuncString
}

func isEmail(val string) bool {
	return strings.Contains(val, "@") && strings.Contains(val, ".")
}

func notEmpty(val string) bool {
	return len(val) > 0
}

type checkFuncString func(string) bool

// if fn returns false, then checking failed.
// so every check function should return true
// in order to pass checking
func checking(val string, cs ...*checkingStruct) error {
	for _, c := range cs {
		if !c.Fn(val) {
			return fmt.Errorf(c.Name + "." + c.Tip)
		}
	}

	return nil
}
