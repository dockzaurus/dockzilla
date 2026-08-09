package domain

// Status represents the status of a deployment.
type Status string

const (
	// StatusQueued indicates a deployment is queued for processing.
	StatusQueued Status = "deployment.queued"
	// StatusRunning indicates a deployment is currently running.
	StatusRunning Status = "deployment.running"
	// StatusDeployed indicates a deployment has completed successfully.
	StatusDeployed Status = "deployed"
	// StatusFailed indicates a deployment has failed.
	StatusFailed Status = "failed"
)

// Deployment represents a deployment record.
type Deployment struct {
	Identifier  UUID
	AppID       UUID
	ImageRef    string
	Status      Status
	TriggeredBy string
}

// CreateDeploymentInput represents the input for creating a deployment.
type CreateDeploymentInput struct {
	AppID    string `json:"app_id"`
	ImageRef string `json:"image_ref"`
	Status   string `json:"status"`
}
