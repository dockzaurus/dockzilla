package domain

type Payload [MaxPayloadSize]byte

type Kind string

// TODO: think about granularity to build the payload.
const (
	// SampleExample represant a simple kind example for payload.
	SampleExample Kind = "sample.run"
)

type HeaderFrame struct {
	Identifier UUID
	Kind       Kind
}
type Message struct {
	Header   HeaderFrame
	Payload  Payload
	Attempts uint32
}

type JobOptions func()
