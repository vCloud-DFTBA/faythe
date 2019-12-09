package model

import "fmt"

// State is the state that a scaler is in.
type State int

const (
	StateNone State = iota
	StateStopping
	StateStopped
	StateFailed
	StateActive
)

func (s State) String() string {
	switch s {
	case StateNone:
		return "none"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateFailed:
		return "failed"
	case StateActive:
		return "acitve"
	default:
		panic(fmt.Sprintf("unknown scaler state: %d", s))
	}
}
