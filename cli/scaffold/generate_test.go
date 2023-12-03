package scaffold

import "testing"

func TestGenerate(t *testing.T) {
	sf := Scaffold{FieldPairs: []FieldType{
		{"name", "string", false},
		{"age", "int", false},
	}}

	expected := `[]string{"id", "name", "age"}`

	if actual := sf.Fields(); actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}
