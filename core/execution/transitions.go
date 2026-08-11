package execution

import "fmt"

// validTransitions defines the allowed state transitions.
var validTransitions = map[Status]map[Status]bool{
	StatusPending: {
		StatusRunning:   true,
		StatusFailed:    true,
	},
	StatusRunning: {
		StatusCompleted: true,
		StatusFailed:    true,
		StatusPaused:    true,
	},
	StatusPaused: {
		StatusRunning:   true, // resume
		StatusFailed:    true, // stop
	},
	StatusCompleted: {},
	StatusFailed:    {},
}

// ValidateTransition checks whether moving from → to is permitted.
func ValidateTransition(from, to Status) error {
	if from == to {
		return fmt.Errorf("execution: no-op transition %s -> %s", from, to)
	}
	allowed, ok := validTransitions[from]
	if !ok || !allowed[to] {
		return fmt.Errorf("execution: illegal transition %s -> %s", from, to)
	}
	return nil
}

// ActionFromMethod maps an API action to a target status.
type Action string

const (
	ActionRun    Action = "run"
	ActionPause  Action = "pause"
	ActionResume Action = "resume"
	ActionStop   Action = "stop"
)

// TargetStatusForAction returns the terminal status an action drives toward,
// or an error if the action cannot be applied to the current status.
func TargetStatusForAction(current Status, action Action) (Status, error) {
	switch action {
	case ActionRun:
		return StatusRunning, nil
	case ActionPause:
		if err := ValidateTransition(current, StatusPaused); err != nil {
			return "", err
		}
		return StatusPaused, nil
	case ActionResume:
		if err := ValidateTransition(current, StatusRunning); err != nil {
			return "", err
		}
		return StatusRunning, nil
	case ActionStop:
		return StatusFailed, nil
	default:
		return "", fmt.Errorf("execution: unknown action %q", action)
	}
}