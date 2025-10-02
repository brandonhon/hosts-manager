package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCategoryAddCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Add category without arguments",
			args:          []string{},
			expectError:   true,
			errorContains: "accepts between 1 and 2 arg(s), received 0",
		},
		{
			name:          "Add category with too many arguments",
			args:          []string{"testing", "description", "extra"},
			expectError:   true,
			errorContains: "accepts between 1 and 2 arg(s), received 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the command
			cmd := categoryAddCmd()

			// Capture output
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			// Set the command args
			cmd.SetArgs(tt.args)

			// Execute the command - we only test argument validation here
			// since the actual execution requires elevated privileges
			err := cmd.Execute()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}
		})
	}
}

func TestCategoryAddCmdStructure(t *testing.T) {
	// Test that the command is properly structured
	cmd := categoryAddCmd()

	if cmd.Use != "add <name> [description]" {
		t.Errorf("Expected Use to be 'add <name> [description]', got: %s", cmd.Use)
	}

	if cmd.Short != "Add a new category" {
		t.Errorf("Expected Short to be 'Add a new category', got: %s", cmd.Short)
	}

	// Test that command has proper argument validation
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Test with no arguments
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error with no arguments")
	} else if !strings.Contains(err.Error(), "accepts between 1 and 2 arg(s), received 0") {
		t.Errorf("Expected specific argument error, got: %v", err)
	}

	// Test with too many arguments
	cmd.SetArgs([]string{"cat1", "desc1", "extra"})
	err = cmd.Execute()
	if err == nil {
		t.Error("Expected error with too many arguments")
	} else if !strings.Contains(err.Error(), "accepts between 1 and 2 arg(s), received 3") {
		t.Errorf("Expected specific argument error, got: %v", err)
	}
}
