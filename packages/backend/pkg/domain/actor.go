package domain

import (
	"context"

	errs "dockzilla/pkg/domain/errors"
)

// Channel is what carried a request into the platform. It is the label set of
// the trigger_channel enum in 00001_init-db.up.sql, so a Channel converts
// straight to the column with no translation table.
type Channel string

const (
	// API is a call to the HTTP API, whether by a human with a session or a
	// machine with a token.
	API Channel = "api"
	// CLI is a call from the dockzilla command-line client.
	CLI Channel = "cli"
	// WebHook is a push from an external system, for example a registry
	// announcing a new image. Nobody is behind it.
	WebHook Channel = "webhook"
	// RollBack is the platform reverting to a previous deployment. It is the
	// one channel the platform triggers on its own behalf.
	RollBack Channel = "rollback"
)

// Actor is who asked for something and how it reached us.
//
// UserID is a pointer rather than a zero-valued UUID because "no user" is a
// real, common state — a webhook and an automated rollback both have a channel
// and nobody behind them — and a zero UUID would be written to the database as
// a valid-looking identifier that matches no row.
type Actor struct {
	Channel Channel
	UserID  *UUID
}

// IsHuman reports whether a named user is behind the action, as opposed to a
// machine acting on its own.
func (a Actor) IsHuman() bool { return a.UserID != nil }

// actorKey is the context key for an Actor. It is an unexported struct type so
// no other package can collide with it: a string key like "user_id" is a value
// any dependency could write, and the reader could not tell whose it was.
type actorKey struct{}

// ContextWithActor returns a copy of ctx carrying actor. Transport middleware
// calls it once, after authentication, so use cases downstream never have to
// take provenance as a parameter that every caller could forge.
func ContextWithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// ActorFromContext returns the Actor ctx carries.
//
// It returns ErrNoActor rather than a usable zero value when nothing set one.
// A zero Actor would have an empty Channel, which the trigger_channel enum
// rejects, so the request would fail anyway — but on a constraint violation
// from the database instead of a sentence naming the missing middleware.
func ActorFromContext(ctx context.Context) (Actor, error) {
	actor, ok := ctx.Value(actorKey{}).(Actor)
	if !ok {
		return Actor{}, errs.ErrNoActor
	}

	return actor, nil
}
