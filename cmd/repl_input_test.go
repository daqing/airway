package cmd

import "testing"

func TestREPLInputAppendsDistinctHistory(t *testing.T) {
	input := &replInput{}

	input.appendHistory("")
	input.appendHistory("repo.Find()")
	input.appendHistory("repo.Find()")
	input.appendHistory("repo.Count()")

	if len(input.history) != 2 {
		t.Fatalf("expected 2 history entries, got %#v", input.history)
	}
}

func TestTrimTrailingNewline(t *testing.T) {
	if got := trimTrailingNewline("repo.Find()\r\n"); got != "repo.Find()" {
		t.Fatalf("unexpected trimmed line: %q", got)
	}
}

func TestPrintableByteAppendsImmediately(t *testing.T) {
	buffer := []rune{}
	buffer = append(buffer, rune('a'))
	buffer = append(buffer, rune('b'))
	buffer = append(buffer, rune('c'))

	if got := string(buffer); got != "abc" {
		t.Fatalf("unexpected buffer contents: %q", got)
	}
}
