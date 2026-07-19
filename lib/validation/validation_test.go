package validation

import "testing"

func TestDoRequiredString(t *testing.T) {
	if err := Do("name", "", "required"); err == nil || err.Error() != "name.required" {
		t.Fatalf("expected name.required error, got %v", err)
	}

	if err := Do("name", "airway", "required"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDoEmail(t *testing.T) {
	if err := Do("email", "not-an-email", "email"); err == nil || err.Error() != "email.format_error" {
		t.Fatalf("expected email.format_error error, got %v", err)
	}

	if err := Do("email", "a@b.com", "required,email"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDoIntRequired(t *testing.T) {
	if err := DoInt("age", 0, "required"); err == nil || err.Error() != "age.required" {
		t.Fatalf("expected age.required error, got %v", err)
	}

	if err := DoInt("age", 18, "required"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
