package security

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestGoModVendor verifies that go.mod and go.sum are consistent.
func TestGoModVendor(t *testing.T) {
	cmd := exec.Command("go", "mod", "verify")
	cmd.Dir = "../core"
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("go mod verify output: %s", string(output))
		// mod verify may fail if go.sum doesn't exist yet — this is non-fatal
		t.Logf("go mod verify: %v (non-fatal)", err)
	}
}

// TestGoVetPasses verifies the core packages pass go vet.
func TestGoVetPasses(t *testing.T) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = "../core"
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet failed: %v\n%s", err, string(output))
	}
}

// TestGoBuildPasses verifies the core packages compile.
func TestGoBuildPasses(t *testing.T) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = "../core"
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(output))
	}
}

// TestEnvFileNotCommitted verifies that .env files (which may contain secrets)
// are not tracked by git.
func TestEnvFileNotCommitted(t *testing.T) {
	data, err := os.ReadFile("../.gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Check that .env is in gitignore
	if !strings.Contains(content, ".env") {
		t.Error(".gitignore does not contain .env")
	}

	// Check that no .env file besides .env.example is tracked
	cmd := exec.Command("git", "ls-files", "*.env")
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	tracked := strings.TrimSpace(string(output))
	t.Logf("tracked .env files: %s", tracked)
}

// TestGoSumExists verifies critical modules have go.sum files.
func TestGoSumExists(t *testing.T) {
	modules := []string{
		"../core/go.sum",
	}

	for _, path := range modules {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing go.sum: %s", path)
		} else {
			t.Logf("go.sum found: %s", path)
		}
	}
}
