package validation

import (
	"fmt"
	"strings"
)

func DoInt(name string, value int, validators string) error {
	v := &intValidator{
		Name:       name,
		Value:      value,
		Validators: validators,
	}

	return v.validate()
}

type intValidator struct {
	Name       string
	Value      int
	Validators string
}

func (v *intValidator) validate() error {
	var validators = []*checkingIntStruct{}

	for _, x := range strings.Split(v.Validators, ",") {
		switch strings.TrimSpace(x) {
		case "required":
			validators = append(validators, &checkingIntStruct{Name: v.Name, Tip: "required", Fn: notEmptyInt})
		}
	}

	return checkingInt(v.Value, validators...)
}

type checkingIntStruct struct {
	Name string
	Tip  string
	Fn   checkFuncInt
}

type checkFuncInt func(int) bool

func notEmptyInt(val int) bool {
	return val > 0
}

// if fn returns false, then checking failed.
// so every check function should return true
// in order to pass checking
func checkingInt(val int, cs ...*checkingIntStruct) error {
	for _, c := range cs {
		if !c.Fn(val) {
			return fmt.Errorf(c.Name + "." + c.Tip)
		}
	}

	return nil
}
