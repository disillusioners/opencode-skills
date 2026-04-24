package main

import (
	"os"
	"strings"
	"testing"

	"opencode_skill/internal/config"
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

func TestResolveMessage_NormalMessage(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "single word",
			parts:    []string{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple words",
			parts:    []string{"hello", "world"},
			expected: "hello world",
		},
		{
			name:     "with spaces preserved",
			parts:    []string{"hello", "world", "test"},
			expected: "hello world test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolveMessage(tc.parts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("resolveMessage(%v) = %q, want %q", tc.parts, result, tc.expected)
			}
		})
	}
}

func TestResolveMessage_FileSyntax(t *testing.T) {
	// Create temp file for testing
	tmpFile, err := os.CreateTemp("", "test_message_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test content
	testContent := "This is a test message.\nWith multiple lines."
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	result, err := resolveMessage([]string{"@" + tmpFile.Name()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be trimmed (no trailing newline)
	expected := "This is a test message.\nWith multiple lines."
	if result != expected {
		t.Errorf("resolveMessage(@file) = %q, want %q", result, expected)
	}
}

func TestResolveMessage_FileNotFound(t *testing.T) {
	// Must use @ prefix for file syntax
	_, err := resolveMessage([]string{"@/nonexistent/file/path.txt"})
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestResolveMessage_EmptyParts(t *testing.T) {
	_, err := resolveMessage([]string{})
	if err == nil {
		t.Error("expected error for empty parts, got nil")
	}
	if err.Error() != "no message provided" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveMessage_FileTrimsWhitespace(t *testing.T) {
	// Create temp file with trailing whitespace
	tmpFile, err := os.CreateTemp("", "test_trim_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write content with trailing newline and spaces
	testContent := "Message with trailing whitespace\n   \n"
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	result, err := resolveMessage([]string{"@" + tmpFile.Name()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should trim trailing whitespace
	expected := "Message with trailing whitespace"
	if result != expected {
		t.Errorf("resolveMessage(@file) = %q, want %q", result, expected)
	}
}

func TestCouncilHint(t *testing.T) {
	// Verify the hint is defined and contains key phrases
	hint := config.CouncilHint
	if hint == "" {
		t.Error("CouncilHint should not be empty")
	}
	// Should start with newline separators
	if len(hint) < 10 {
		t.Error("CouncilHint is too short")
	}
	// Should mention council subagent
	if !strings.Contains(hint, "council subagent") {
		t.Error("CouncilHint should mention 'council subagent'")
	}
}
