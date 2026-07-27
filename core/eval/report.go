package eval

import "fmt"

// Summary returns a human-readable summary of evaluation results.
func Summary(results []EvalResult) string {
	total := len(results)
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}

	s := fmt.Sprintf("\n=== Evaluation Results ===\n")
	s += fmt.Sprintf("Tasks: %d total, %d passed, %d failed\n", total, passed, total-passed)

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		s += fmt.Sprintf("  %-20s %s (%s)\n", r.TaskName, status, r.Duration)
		if !r.Passed {
			for _, err := range r.Errors {
				s += fmt.Sprintf("    - %s\n", err)
			}
		}
	}

	return s
}
