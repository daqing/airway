package media_api

import (
	"testing"
)

func TestReplace(t *testing.T) {
	expected := "bar.pdf"
	actual := replace("foo.pdf", "bar")

	if actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}

	expected = "hello.zip"
	actual = replace("foo.bar.zip", "hello")

	if actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestHashDirPath(t *testing.T) {
	expected := "/tmp/a2/a2b3"

	if actual := hashDirPath("/tmp", "a2b3041922.png"); actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestDirParts(t *testing.T) {
	tests := []struct {
		Path     string
		Expected string
	}{
		{"abc", "abc"},
		{"abcd.png", "/ab/abcd"},
	}

	for _, test := range tests {
		actual := DirParts(test.Path)

		if actual != test.Expected {
			t.Errorf("expected %s, got %s", test.Expected, actual)
		}
	}
}
