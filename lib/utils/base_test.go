package utils

import (
	"testing"
)

func TestToArray(t *testing.T) {
	arr := ToArray("1,2,3", ",")

	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}

	if arr[0] != "1" {
		t.Errorf("Expected 1, got %s", arr[0])
	}

	arr2 := ToArray("1", ",")
	if len(arr2) != 1 {
		t.Errorf("Expected 1 element, got %d", len(arr2))
	}

	if arr2[0] != "1" {
		t.Errorf("Expected 1, got %s", arr2[0])
	}
}
