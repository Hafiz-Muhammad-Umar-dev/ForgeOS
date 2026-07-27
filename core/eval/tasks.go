package eval

import (
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agent"
)

// DefaultGoldenTasks returns the 5 golden tasks for agent behavior validation.
// Each task exercises a different capability of the agent system.
func DefaultGoldenTasks() []GoldenTask {
	return []GoldenTask{
		codeGenTask(),
		codeReviewTask(),
		testWritingTask(),
		explainTask(),
		securityAuditTask(),
	}
}

func codeGenTask() GoldenTask {
	return GoldenTask{
		Name:        "code-gen",
		Description: "Generate a Fibonacci function in Go",
		Task: agent.Task{
			ID:          "golden-code-gen",
			Description: "Write a Fibonacci function in Go. Include the function signature and a brief comment.",
			MaxIterations: 1,
		},
		Assertions: []Assertion{
			AssertNotEmpty(),
			AssertContains("func Fib"),
		},
		Timeout: 30 * time.Second,
	}
}

func codeReviewTask() GoldenTask {
	return GoldenTask{
		Name:        "code-review",
		Description: "Review the generated Fibonacci code",
		Task: agent.Task{
			ID:          "golden-review",
			Description: "Review the following Go Fibonacci function for correctness, performance, and style:\n\nfunc Fib(n int) int {\n    if n <= 0 { return 0 }\n    if n == 1 { return 1 }\n    return Fib(n-1) + Fib(n-2)\n}\n\nProvide your review comments.",
			MaxIterations: 1,
		},
		Assertions: []Assertion{
			AssertNotEmpty(),
			AssertContains("review"),
		},
		Timeout: 30 * time.Second,
	}
}

func testWritingTask() GoldenTask {
	return GoldenTask{
		Name:        "test-writing",
		Description: "Write tests for the Fibonacci function",
		Task: agent.Task{
			ID:          "golden-test",
			Description: "Write Go tests for a Fibonacci function. The tests should cover edge cases including n=0, n=1, and n=10. Use the standard testing package.",
			MaxIterations: 1,
		},
		Assertions: []Assertion{
			AssertNotEmpty(),
			AssertContains("TestFib"),
		},
		Timeout: 30 * time.Second,
	}
}

func explainTask() GoldenTask {
	return GoldenTask{
		Name:        "explain",
		Description: "Explain how goroutines work",
		Task: agent.Task{
			ID:          "golden-explain",
			Description: "Explain how goroutines work in Go. Cover concurrency, channels, and the scheduler model.",
			MaxIterations: 1,
		},
		Assertions: []Assertion{
			AssertNotEmpty(),
			AssertMatch("(?i)goroutine|concurren"),
		},
		Timeout: 30 * time.Second,
	}
}

func securityAuditTask() GoldenTask {
	return GoldenTask{
		Name:        "security-audit",
		Description: "Find security issues in a SQL query",
		Task: agent.Task{
			ID:          "golden-security",
			Description: "Find security issues in this SQL query:\n\nSELECT * FROM users WHERE username = '" + "' OR '1'='1" + "';\n\nList each vulnerability and suggest a fix.",
			MaxIterations: 1,
		},
		Assertions: []Assertion{
			AssertNotEmpty(),
			AssertMatch("(?i)injection|sql|vulnerab"),
		},
		Timeout: 30 * time.Second,
	}
}
