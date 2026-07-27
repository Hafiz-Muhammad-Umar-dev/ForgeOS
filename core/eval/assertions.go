// Package eval provides golden task evaluation for agent behavior validation.
// It defines Assertion types, an Evaluator for running tasks, and predefined
// golden tasks that exercise the full intent→agent→workspace pipeline.
//
// Sprint 0 scope:
//   - GoldenTask definitions (5 tasks)
//   - Assertion helpers (Contains, NotEmpty, Match)
//   - Evaluator (runs tasks through Agent.Runtime)
//   - Integration tests with FakeProvider + FakeWorkspace
//
// See Build Order Step 13 (Hardening), E7 (Quality & Safety).
package eval

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
)

// Assertion validates the output of a golden task execution.
type Assertion struct {
	// Description explains what this assertion verifies.
	Description string

	// Check returns an error if the assertion fails.
	Check func(result *agent.Result) error
}

// AssertContains returns an Assertion that verifies the result summary
// contains the given substring.
func AssertContains(substr string) Assertion {
	return Assertion{
		Description: fmt.Sprintf("result contains %q", substr),
		Check: func(result *agent.Result) error {
			if !strings.Contains(result.Summary, substr) {
				return fmt.Errorf("expected result to contain %q, got %q", substr, result.Summary)
			}
			return nil
		},
	}
}

// AssertNotEmpty returns an Assertion that verifies the result summary
// is not empty.
func AssertNotEmpty() Assertion {
	return Assertion{
		Description: "result is not empty",
		Check: func(result *agent.Result) error {
			if result.Summary == "" {
				return fmt.Errorf("expected non-empty result, got empty")
			}
			return nil
		},
	}
}

// AssertMatch returns an Assertion that verifies the result summary
// matches the given regular expression.
func AssertMatch(pattern string) Assertion {
	re := regexp.MustCompile(pattern)
	return Assertion{
		Description: fmt.Sprintf("result matches %q", pattern),
		Check: func(result *agent.Result) error {
			if !re.MatchString(result.Summary) {
				return fmt.Errorf("expected result to match %q, got %q", pattern, result.Summary)
			}
			return nil
		},
	}
}

// AssertStatus returns an Assertion that verifies the result status.
func AssertStatus(status agent.ResultStatus) Assertion {
	return Assertion{
		Description: fmt.Sprintf("result status is %s", status),
		Check: func(result *agent.Result) error {
			if result.Status != status {
				return fmt.Errorf("expected status %s, got %s", status, result.Status)
			}
			return nil
		},
	}
}
