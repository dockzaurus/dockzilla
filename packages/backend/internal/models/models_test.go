package models_test

import (
	"testing"
	"time"

	"dockzilla/internal/models"
	"github.com/stretchr/testify/require"
)

// now is the instant every predicate below is evaluated against.
func now() time.Time {
	return time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
}

func TestAPITokens_IsActive(t *testing.T) {
	t.Parallel()

	type args struct {
		expiresAt time.Time
		revokedAt time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success - a token with no expiry never expires",
			args: args{},
			want: true,
		},
		{
			name: "success - expiry in the future",
			args: args{expiresAt: now().Add(time.Hour)},
			want: true,
		},
		{
			name: "error - expiry in the past",
			args: args{expiresAt: now().Add(-time.Hour)},
			want: false,
		},
		{
			name: "error - expiry exactly now is not after now",
			args: args{expiresAt: now()},
			want: false,
		},
		{
			name: "error - revoked beats a future expiry",
			args: args{expiresAt: now().Add(time.Hour), revokedAt: now().Add(-time.Minute)},
			want: false,
		},
		{
			name: "error - revoked beats no expiry at all",
			args: args{revokedAt: now().Add(-time.Minute)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := &models.APITokens{
				ExpiresAt: tt.args.expiresAt,
				RevokedAt: tt.args.revokedAt,
			}

			require.Equal(t, tt.want, token.IsActive(now()))
		})
	}
}

func TestSessions_IsActive(t *testing.T) {
	t.Parallel()

	type args struct {
		expiresAt time.Time
		revokedAt time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success - unrevoked and not yet expired",
			args: args{expiresAt: now().Add(time.Hour)},
			want: true,
		},
		{
			name: "error - expired",
			args: args{expiresAt: now().Add(-time.Hour)},
			want: false,
		},
		{
			name: "error - expiry exactly now is not after now",
			args: args{expiresAt: now()},
			want: false,
		},
		{
			name: "error - revoked",
			args: args{expiresAt: now().Add(time.Hour), revokedAt: now().Add(-time.Minute)},
			want: false,
		},
		{
			// Unlike an API token, a session has a NOT NULL expiry, so a zero
			// value is a broken row rather than "never expires".
			name: "error - a zero expiry is not open-ended",
			args: args{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := &models.Sessions{
				ExpiresAt: tt.args.expiresAt,
				RevokedAt: tt.args.revokedAt,
			}

			require.Equal(t, tt.want, session.IsActive(now()))
		})
	}
}

func TestSetupClaims_IsClaimable(t *testing.T) {
	t.Parallel()

	type args struct {
		expiresAt  time.Time
		consumedAt time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success - unconsumed and unexpired",
			args: args{expiresAt: now().Add(time.Hour)},
			want: true,
		},
		{
			name: "error - already consumed",
			args: args{expiresAt: now().Add(time.Hour), consumedAt: now().Add(-time.Minute)},
			want: false,
		},
		{
			name: "error - expired",
			args: args{expiresAt: now().Add(-time.Hour)},
			want: false,
		},
		{
			name: "error - a zero expiry is not open-ended",
			args: args{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			claim := &models.SetupClaims{
				ExpiresAt:  tt.args.expiresAt,
				ConsumedAt: tt.args.consumedAt,
			}

			require.Equal(t, tt.want, claim.IsClaimable(now()))
		})
	}
}

func TestApps_IsArchived(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		archivedAt time.Time
		want       bool
	}{
		{
			name: "success - a live app has no archive stamp",
			want: false,
		},
		{
			name:       "success - archived",
			archivedAt: now(),
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &models.Apps{ArchivedAt: tt.archivedAt}

			require.Equal(t, tt.want, app.IsArchived())
		})
	}
}

func TestDeployments_IsTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "success - queued is still moving", status: models.DeploymentQueued},
		{name: "success - pulling is still moving", status: models.DeploymentPulling},
		{name: "success - starting is still moving", status: models.DeploymentStarting},
		{name: "success - running is terminal", status: models.DeploymentRunning, want: true},
		{name: "success - failed is terminal", status: models.DeploymentFailed, want: true},
		{
			name:   "success - superseded is terminal",
			status: models.DeploymentSuperseded,
			want:   true,
		},
		{name: "success - an unknown status is not terminal", status: "who-knows"},
		{name: "success - the zero value is not terminal", status: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deployment := &models.Deployments{Status: tt.status}

			require.Equal(t, tt.want, deployment.IsTerminal())
		})
	}
}

func TestIdentities_IsLocal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{name: "success - the local provider", provider: models.ProviderLocal, want: true},
		{
			name:     "success - an external provider is not local",
			provider: models.ProviderOIDCPrefix + "keycloak",
		},
		{name: "success - the prefix alone is not local", provider: models.ProviderOIDCPrefix},
		{name: "success - the zero value is not local", provider: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			identity := &models.Identities{Provider: tt.provider}

			require.Equal(t, tt.want, identity.IsLocal())
		})
	}
}
