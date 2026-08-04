package domain

import (
	"encoding/json"
	"time"
)

type Payload json.RawMessage
type Key string

func (p Payload) isValidSize() bool {
	size := len(p)
	if size > MaxPayloadSize || size <= 0 {
		return false
	}

	return true
}

type Kind string

// TODO: think about granularity to build the payload.
const (
	RunDeployment  Kind = "deployment.run"
	StopDeployment Kind = "deployment.stop"

	StartApp   Kind = "app.start"
	StopApp    Kind = "app.stop"
	RestartApp Kind = "app.restart"
)

// HeaderFrame represent the Header data sent to the queue.
type HeaderFrame struct {
	Identifier UUID
	Kind       Kind
}

// Message type represent the block sent to the queue.
type Message struct {
	Header   HeaderFrame
	Payload  Payload
	Attempts uint32
}

type JobConfig struct {
	RunAfter    time.Time
	MaxAttempts int32
	UniqueKey   Key
}

type JobOptions interface {
	apply(*JobConfig)
}
