// Package middleware holds the gin handlers that run before a route's own
// handler: authentication, and later rate limiting and request identifiers.
package middleware

import (
	"context"
	"log/slog"

	dockzillahttp "dockzilla/internal/infra/transport/http"
	"dockzilla/internal/utils"
	"dockzilla/pkg/domain"
	"github.com/gin-gonic/gin"
)

// devUserID is the user every request is attributed to until real
// authentication lands. It is a valid UUID because the identifier it stands in
// for is one: deployments.triggered_by_user_id is a uuid column with a foreign
// key onto users, so a placeholder like "dckz_user_0112" fails to parse and a
// well-formed but unknown one fails the constraint. The users row for this
// identifier has to exist in the development database.
const devUserID = "00000000-0000-4000-8000-000000000001"

// Claims is what a validated token asserts about its bearer.
type Claims struct{}

// TokenPair is the access and refresh token pair handed to a client.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

// Auth validates and refreshes the tokens presented by clients.
type Auth interface {
	ValidateToken(ctx context.Context, token string) (*Claims, error)
	RefreshToken(ctx context.Context) (*Claims, error)
}

// RequireAuth is the middleware to protect routes. It is a stub: there is no
// auth service yet, so it attaches a fixed development actor and lets the
// request through. Provenance is set here rather than in each handler so that
// when this starts reading a real token, every route picks it up unchanged.
//
// The parse happens once, at construction, because devUserID is a constant: a
// bad one is a build mistake and there is nothing a request can do to change
// the answer. When it fails the routes behind this middleware refuse instead
// of the process dying at whichever request arrives first.
func RequireAuth(_ Auth, logger *slog.Logger) gin.HandlerFunc {
	userID, err := utils.UUIDParser(devUserID)
	if err != nil {
		return dockzillahttp.Misconfigured(logger, err)
	}

	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(
			domain.ContextWithActor(c.Request.Context(), domain.Actor{
				Channel: domain.API,
				UserID:  &userID,
			}),
		)

		c.Next()
	}
}
