package domain

// This file is the authoring source of every job payload contract. The JSON
// Schema documents under internal/core/jobs/schemas/catalog/schema are
// generated from these structs by cmd/schemagen and verified in CI, so the
// type a handler unmarshals into and the schema a producer is validated
// against can never drift apart.
//
// The version is part of the type name on purpose. A published version is
// frozen: adding, removing or retyping a field means writing DeployArgsV2 next
// to DeployArgsV1, not editing V1. Editing V1 makes the generated document
// differ from the committed one, which fails the CI check, and differ from the
// row already in the database, which fails the boot-time registration.
//
// Constraints live in `jsonschema` tags; the field's doc comment becomes the
// schema's description, so keep it a sentence about what the value means.

// DeployArgsV1 is the v1 payload of the deployment.run job.
type DeployArgsV1 struct {
	// DeploymentID is the ledger row this run reports its progress against.
	DeploymentID string `json:"deployment_id" jsonschema:"required,format=uuid"`

	// AppID is the application being deployed.
	AppID string `json:"app_id" jsonschema:"required,format=uuid"`

	// ImageRef is the image reference as requested, before the digest is
	// resolved at pull time.
	ImageRef string `json:"image_ref" jsonschema:"required,minLength=1"`

	// TriggeredBy records what caused the deployment.
	TriggeredBy string `json:"triggered_by" jsonschema:"required,enum=api,enum=cli,enum=webhook,enum=rollback"` //nolint:lll // an enum tag cannot be wrapped.

	// TriggeredByUserID is the user who asked for the deployment. It is empty
	// for a webhook, which nobody triggers by hand.
	TriggeredByUserID string `json:"triggered_by_user_id,omitempty" jsonschema:"format=uuid"`
}

// StartAppArgsV1 is the v1 payload of the app.start job.
type StartAppArgsV1 struct {
	// AppID is the application whose container should be started.
	AppID string `json:"app_id" jsonschema:"required,format=uuid"`
}

// StopAppArgsV1 is the v1 payload of the app.stop job.
type StopAppArgsV1 struct {
	// AppID is the application whose container should be stopped.
	AppID string `json:"app_id" jsonschema:"required,format=uuid"`

	// TimeoutSeconds is how long the container is given to exit on its own
	// before it is killed. Zero means the engine's own default.
	TimeoutSeconds int `json:"timeout_seconds,omitempty" jsonschema:"minimum=0,maximum=3600"`
}

// RestartAppArgsV1 is the v1 payload of the app.restart job.
type RestartAppArgsV1 struct {
	// AppID is the application whose container should be restarted.
	AppID string `json:"app_id" jsonschema:"required,format=uuid"`

	// TimeoutSeconds is how long the container is given to exit on its own
	// before it is killed. Zero means the engine's own default.
	TimeoutSeconds int `json:"timeout_seconds,omitempty" jsonschema:"minimum=0,maximum=3600"`
}
