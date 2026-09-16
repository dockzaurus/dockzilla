package domain

// Status is where a deployment has got to. The values are the labels of the
// deployment_status enum in 00001_init-db.up.sql, so a Status can be written
// to the column with a plain string conversion and no translation table.
//
// Changing one of these strings means altering a Postgres enum, which has no
// DROP VALUE: it is a new type, a column move and a drop. Add values freely,
// rename none.
type Status string

const (
	// StatusQueued is the state every deployment is created in: accepted and
	// recorded, with the job enqueued but not yet picked up.
	StatusQueued Status = "queued"
	// StatusPulling means the worker is fetching the image from its registry,
	// which is the slow step and the one that usually fails.
	StatusPulling Status = "pulling"
	// StatusStarting means the image is local and the container is being
	// created and started, up to the point it passes its health check.
	StatusStarting Status = "starting"
	// StatusRunning is the terminal success state: the container is healthy and
	// the proxy points at it.
	StatusRunning Status = "running"
	// StatusFailed is the terminal failure state. ErrorCode and ErrorMessage on
	// the row say why.
	StatusFailed Status = "failed"
	// StatusSuperseded is the terminal state of a deployment that a newer one
	// overtook before it finished, so nobody is waiting on its outcome.
	StatusSuperseded Status = "superseded"
)

// IsTerminal reports whether the deployment has stopped moving. It must agree
// with models.Deployments.IsTerminal, which answers the same question about
// the row this Status was read from.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusRunning, StatusFailed, StatusSuperseded:
		return true
	case StatusQueued, StatusPulling, StatusStarting:
		return false
	default:
		return false
	}
}

// Deployment represents a deployment record.
type Deployment struct {
	Identifier        UUID   `json:"deployment_id"`
	AppID             UUID   `json:"app_id"`
	ImageRef          string `json:"image_ref"`
	Status            Status
	TriggeredBy       Channel `json:"triggered_by"`
	TriggeredByUserID *UUID
}

// CreateDeploymentInput represents the input for creating a deployment.
type CreateDeploymentInput struct {
	AppID    string `json:"app_id"`
	ImageRef string `json:"image_ref"`
	Status   string `json:"status"`
}
