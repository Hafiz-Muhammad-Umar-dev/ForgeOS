// Package security provides automated security audit tests for the DevOS repository.
package security

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// allowlistedPatterns contains patterns that are known test values or
// false positives and should be excluded from secret scan results.
var allowlistedPatterns = []string{
	"sk-ant-api03", // Test API key in .env.example
	"devos-development-secret", // Default dev secret in cmd/main.go
	"test-key", // Unit test key
	"fake-key", // Test key
	"test-token", // Test token
}

// secretPatterns contains regex-like patterns for detecting committed secrets.
var secretPatterns = []struct {
	name    string
	check   func(line string) bool
}{
	{
		name: "AWS Access Key",
		check: func(line string) bool {
			return strings.Contains(line, "AKIA") && strings.Contains(line, "=")
		},
	},
	{
		name: "Generic API Key",
		check: func(line string) bool {
			return strings.Contains(line, "api_key") || strings.Contains(line, "api_secret")
		},
	},
	{
		name: "Private Key",
		check: func(line string) bool {
			return strings.Contains(line, "BEGIN PRIVATE KEY") || strings.Contains(line, "BEGIN RSA PRIVATE KEY")
		},
	},
	{
		name: "JWT Secret in Code",
		check: func(line string) bool {
			return strings.Contains(line, "jwt_secret") || strings.Contains(line, "JWT_SECRET")
		},
	},
}

// TestCommittedSecrets scans Go source files for accidentally committed secrets.
func TestCommittedSecrets(t *testing.T) {
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".env") && !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for _, pattern := range secretPatterns {
				if pattern.check(line) && !isAllowlisted(line) {
					t.Errorf("potential secret in %s:%d: %s matched %s", path, lineNum, pattern.name, line)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// isAllowlisted checks if a line matches known test values.
func isAllowlisted(line string) bool {
	for _, p := range allowlistedPatterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

// TestGitIgnoreExists verifies the repository has a .gitignore for common secret files.
func TestGitIgnoreExists(t *testing.T) {
	data, err := os.ReadFile("../.gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)

	requiredPatterns := []string{
		".env",
		"*.key",
		"*.pem",
		"secrets",
	}

	for _, pattern := range requiredPatterns {
		if !strings.Contains(content, pattern) {
			t.Errorf(".gitignore missing pattern: %s", pattern)
		}
	}
}
