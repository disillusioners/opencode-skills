package main

import (
	"testing"
)

func TestFormatSubmittedMessage(t *testing.T) {
	project := "testproject"
	session := "testsession"
	expected := "[SUBMITTED] Run: opencode_skill testproject testsession /wait"
	result := formatSubmittedMessage(project, session)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFormatSubmittedMessageDifferentInputs(t *testing.T) {
	tests := []struct {
		project  string
		session  string
		expected string
	}{
		{"myproject", "mysession", "[SUBMITTED] Run: opencode_skill myproject mysession /wait"},
		{"abc", "123", "[SUBMITTED] Run: opencode_skill abc 123 /wait"},
		{"project-name", "session-name", "[SUBMITTED] Run: opencode_skill project-name session-name /wait"},
	}

	for _, tc := range tests {
		result := formatSubmittedMessage(tc.project, tc.session)
		if result != tc.expected {
			t.Errorf("formatSubmittedMessage(%q, %q) = %q, want %q", tc.project, tc.session, result, tc.expected)
		}
	}
}
