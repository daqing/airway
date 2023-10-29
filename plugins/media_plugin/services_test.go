package media_plugin

import (
	"os"
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
	if err := os.Setenv("AIRWAY_STORAGE_DIR", "/tmp"); err != nil {
		t.Error(err)
	}

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
		actual := dirParts(test.Path)

		if actual != test.Expected {
			t.Errorf("expected %s, got %s", test.Expected, actual)
		}
	}
}

func TestAssetHostPath(t *testing.T) {
	expected := "https://abcd.com/ab/abcd/abcdefg.png"
	actual := assetHostPath("https://abcd.com", "abcdefg.png")

	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}
