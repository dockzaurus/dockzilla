package domain

import (
	"context"
	"encoding/json"
	"fmt"
)

type Payload = json.RawMessage

func IsPayloadValid(payload []byte) bool {
	pSize := len(payload)

	if pSize < 1 {
		return false
	}

	if pSize > MaxPayloadSize {
		return false
	}

	return true
}
func NewPayload(b []byte) (Payload, error) {
	// allocate every new even if wrong here ......
	p := Payload(b)

	if !IsPayloadValid(p) {
		return nil, fmt.Errorf("invalid payload")
	}

	return p, nil
}

type Key string

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

type Dispatch func(context.Context, Message) error
