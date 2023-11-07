package repo

import "testing"

func TestToCamel(t *testing.T) {
	tests := []struct {
		Value    string
		Expected string
	}{
		{"hello_world", "HelloWorld"},
		{"name_uuid", "NameUUID"},
		{"blog_url", "BlogURL"},
	}

	for _, test := range tests {
		actual := ToCamel(test.Value)

		if actual != test.Expected {
			t.Errorf("Expected %v, got %v", test.Expected, actual)
		}
	}
}
