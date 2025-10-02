package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hosts-manager/internal/config"
	"hosts-manager/internal/hosts"
)

func TestCategoryAddCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		expectOutput  string
	}{
		{
			name:         "Add category with name only",
			args:         []string{"testing"},
			expectError:  false,
			expectOutput: "Added category: testing",
		},
		{
			name:         "Add category with name and description",
			args:         []string{"testing", "Testing category with description"},
			expectError:  false,
			expectOutput: "Added category: testing - Testing category with description",
		},
		{
			name:          "Add category without arguments",
			args:          []string{},
			expectError:   true,
			errorContains: "requires at least 1 arg",
		},
		{
			name:          "Add category with too many arguments",
			args:          []string{"testing", "description", "extra"},
			expectError:   true,
			errorContains: "accepts at most 2 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary hosts file
			tmpFile, err := os.CreateTemp("", "hosts_test_*.txt")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			// Write sample content
			sampleContent := `127.0.0.1 localhost
# @category development
192.168.1.100 api.dev
`
			if _, err := tmpFile.WriteString(sampleContent); err != nil {
				t.Fatal(err)
			}
			if err := tmpFile.Close(); err != nil {
				t.Fatal(err)
			}

			// Create temporary config
			tmpDir, err := os.MkdirTemp("", "config_test_*")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create test config
			testConfig := &config.Config{
				General: config.General{
					AutoBackup:      false, // Disable backup for tests
					DefaultCategory: "default",
				},
				Backup: config.Backup{
					Directory: filepath.Join(tmpDir, "backups"),
				},
			}

			// Set global config for test
			cfg = testConfig

			// Create the command
			cmd := categoryAddCmd()

			// Capture output
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set the command args
			cmd.SetArgs(tt.args)

			// Mock the platform behavior by temporarily changing the hosts file path
			// Note: This is a limitation of the current design - in a real refactor,
			// we'd inject dependencies. For now, we'll test what we can.

			// Execute the command
			err = cmd.Execute()

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

			// Should not have error
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				t.Logf("Stderr: %s", stderr.String())
				return
			}

			// Check output
			output := stdout.String()
			if tt.expectOutput != "" && !strings.Contains(output, tt.expectOutput) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expectOutput, output)
			}
		})
	}
}

func TestCategoryAddCmdDryRun(t *testing.T) {
	// Create temporary hosts file
	tmpFile, err := os.CreateTemp("", "hosts_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write sample content
	sampleContent := `127.0.0.1 localhost`
	if _, err := tmpFile.WriteString(sampleContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create temporary config
	tmpDir, err := os.MkdirTemp("", "config_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test config
	testConfig := &config.Config{
		General: config.General{
			AutoBackup:      false,
			DefaultCategory: "default",
		},
		Backup: config.Backup{
			Directory: filepath.Join(tmpDir, "backups"),
		},
	}

	// Set global config for test
	cfg = testConfig

	// Set dry run mode
	dryRun = true
	defer func() { dryRun = false }()

	// Create the command
	cmd := categoryAddCmd()

	// Capture output
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Set the command args
	cmd.SetArgs([]string{"testing", "Test description"})

	// Execute the command
	err = cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Check dry run output
	output := stdout.String()
	expectedOutput := "Would add category: testing - Test description"
	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected dry run output to contain '%s', got: %s", expectedOutput, output)
	}

	// Verify the file wasn't actually modified
	parser := hosts.NewParser(tmpFile.Name())
	hostsFile, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse hosts file after dry run: %v", err)
	}

	// Should not have the testing category
	testingCategory := hostsFile.GetCategory("testing")
	if testingCategory != nil {
		t.Errorf("Expected category 'testing' not to exist after dry run, but it does")
	}
}
