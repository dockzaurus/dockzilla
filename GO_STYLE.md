# Go Style & Rules — Dockzilla backend

This is the house style for everything under `packages/backend`. It is written for someone who
knows how to program but is new to Go, so it explains the *why*, not just the rule.

Our baseline is the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).
When this document and Uber's disagree, this one wins — but that should be rare, and if you spot a
contradiction, raise it instead of guessing.

Every example below is real code from this repo, before and after review. Nothing here is
hypothetical.

---

## 0. Before you write a line

Read, in order:

1. [Effective Go](https://go.dev/doc/effective_go) — the language's own idiom guide.
2. [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) — our baseline.
3. [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) — short, and it's what
   reviewers will quote at you.

You don't need to memorise them. You need to have seen them once so the rules below feel familiar.

---

## 1. Tooling — non-negotiable

One command runs everything CI runs:

```bash
task backend:check     # build + vet + lint + test -race
```

The individual pieces, if you want them one at a time:

```bash
cd packages/backend
gofmt -l .             # must print nothing
go build ./...         # must succeed
go vet ./...           # must be silent
golangci-lint run      # must report 0 issues
go test -race ./...    # must pass
```

Toolchain versions are pinned in `mise.toml`, so everyone lints with the same rules. Run
`mise install` once and you have the right `golangci-lint`.

- **`gofmt` is not a preference.** Go has exactly one formatting style and a tool that applies it.
  There is no debate about braces, tabs, or alignment in Go — this is a feature. Configure your
  editor to run `gofmt` on save and never think about it again.
- **`go vet`** catches a real class of bugs: printf arguments that don't match the format string,
  locks copied by value, unreachable code. Silence it, don't ignore it.
- **`golangci-lint`** runs ~50 linters configured in `packages/backend/.golangci.yml`. That file is
  the machine-readable version of this document: nearly every rule below is enforced by a named
  linter, and the config says which one. Read its comments — they map linters to Uber rules.

A large share of findings fix themselves:

```bash
task backend:lint:fix
```

If you're about to disable a check, that's a conversation, not a commit. When a `//nolint` is
genuinely warranted it must name the linter and give a reason — `nolintlint` enforces that:

```go
//nolint:gosec // G404: this ID is not a security token, math/rand is fine.
```

---

## 2. Comments

**Rule: exported identifiers get a doc comment. Unexported ones don't, unless the code is doing
something genuinely non-obvious.**

"Exported" in Go means *starts with a capital letter*. That is the entire visibility system — there
is no `public`/`private` keyword. `Server` is visible to other packages; `srv` is not.

A doc comment starts with the name of the thing it documents and is a complete sentence:

```go
// NewServer builds a Server from opts. It returns an error when a required
// option is missing or when the underlying giniservice server cannot be
// created, so a caller never receives a partially initialised Server.
func NewServer(opts ...Option) (*Server, error) {
```

Why "starts with the name": `go doc` and pkg.go.dev render these comments as the package's API
documentation, and the text reads as `NewServer builds a Server from opts.` A comment that starts
`// This function builds...` reads badly and marks you as new.

Do **not** write doc comments on unexported functions just to fill space:

```go
// BAD — id is unexported; this comment is noise.
// id returns the service identifier as a string, read under the lock.
func (a *Application) id() string {

// GOOD
func (a *Application) id() string {
```

Unexported code should be readable *without* a comment. If it genuinely isn't — a lock ordering
rule, a workaround for an upstream bug, a non-obvious invariant — then comment the *why*, never the
*what*:

```go
// GOOD — explains an invariant you cannot infer from the code.
// mu guards serviceID and cancel. Both are written from the goroutine
// that boots the service and read from the one that stops it.
mu        sync.RWMutex
serviceID domain.UUID
cancel    context.CancelFunc
```

Packages get a doc comment too, on one file per package:

```go
// Package core holds the application service: the top-level orchestrator that
// owns every transport handler (HTTP today, workers later) and drives their
// startup and graceful shutdown as a single unit.
package core
```

Write comments in English, in full sentences, and proofread them. `// Run stary the main loop` was
in this codebase; it's a small thing, but sloppy comments make readers distrust the code.

---

## 3. Errors

### Never panic in normal operation

`panic` kills the process and unwinds the stack. It's for programmer bugs that make continuing
meaningless — not for "the config file was missing."

```go
// BAD — what we had.
func main() {
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	...
}

// GOOD — main stays tiny, run() returns errors, deferred cleanup still runs.
func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	...
}
```

The `main` / `run` split is a standard Go pattern. `os.Exit` skips `defer`s, so you want exactly one
`os.Exit`, at the very top, after everything else has unwound.

### Never swallow an error

This was the worst bug in the codebase:

```go
// BAD — the error is thrown away and nil is returned as a *Server.
func NewServer(opts ...Options) *Server {
	srv, err := ginihttp.NewHTTPServer(...)
	if err != nil {
		return nil
	}
	...
}
```

Two things go wrong. First, the reason for the failure is gone forever. Second — and this is a
classic Go trap — that `nil` gets stored in a `domain.Service` interface. **An interface holding a
nil pointer is not equal to nil.** So `if handler != nil` passes, and the program crashes later, far
from the actual cause.

```go
// GOOD — a constructor that can fail returns an error.
func NewServer(opts ...Option) (*Server, error) {
	srv, err := ginihttp.NewHTTPServer(...)
	if err != nil {
		return nil, fmt.Errorf("http server: %w", err)
	}
	...
}
```

### Wrap with `%w`, add context, don't repeat it

`%w` keeps the original error inspectable by `errors.Is` / `errors.As`. Each layer adds *what it was
trying to do*, lowercase, no trailing punctuation:

```go
return fmt.Errorf("stop %s: %w", handler.Name(), err)
```

Reading the final message top to bottom should tell the story:
`create http server: http server: config is nil`.

Don't include the words "error" or "failed" — the fact that it's an error is already known from
context. `fmt.Errorf("failed to load config: %w", err)` becomes
`failed to load config: failed to open file: ...` once wrapped twice.

### Collect errors when you must keep going

During shutdown you want to stop *every* handler even if the first one fails:

```go
var errs []error

for _, handler := range a.handlers {
	if err := handler.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("stop %s: %w", handler.Name(), err))
	}
}

a.wg.Wait()

return errors.Join(errs...)
```

`errors.Join` returns `nil` for an empty slice, so you don't need an `if len(errs) > 0` guard.

Watch the ordering here — the original code returned early on error and left `a.wg.Wait()`
unreachable, so a failing handler silently skipped waiting for the goroutines.

---

## 4. Constructors and functional options

Our services are built with **functional options**. Uber's guide is specific about the shape: the
`Option` type is an **interface holding an unexported method**, not a bare `func` type.

```go
// Option configures a Server during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type Option interface {
	apply(s *Server)
}

// optionFunc adapts a plain function to the Option interface.
type optionFunc func(*Server)

func (f optionFunc) apply(s *Server) { f(s) }

// WithLogger sets the structured logger used by the server. It is required:
// NewServer fails when no logger is provided.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Server) {
		s.logger = logger
	})
}

func NewServer(opts ...Option) (*Server, error) {
	s := &Server{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.logger == nil {
		return nil, errors.New("http server: logger is required")
	}
	...
}
```

The three extra lines (`optionFunc` and its `apply`) buy you what a closure can't: options are
**values you can compare in a test**, and they can implement other interfaces such as
`fmt.Stringer`. `type Option func(*Server)` gives you neither — two closures are never equal, so you
cannot assert that a function returned the option you expected.

Rules we follow:

- The type is named **`Option`**, singular — it is *one* option. It was `Options` and
  `ApplicationConfig` in two different packages; both are now `Option`. Consistency across packages
  matters more than any individual name.
- **Validate required options in the constructor.** Options are optional by construction, so the
  only place you can enforce "logger is mandatory" is `New...`.
- Say in each option's doc comment whether it is required.
- Use `&Server{}`, never `new(Server)` — the guide's "Initializing Struct References" rule. `&T{}`
  reads the same whether or not you set fields at construction; `new(T)` doesn't scale to that.

This is the one place we deliberately return an interface from an exported function, which is why
`.golangci.yml` exempts `.*Option$` from `ireturn`.

---

## 5. Concurrency

Go makes concurrency easy to write and just as easy to get wrong. The compiler will not save you
here; `go build` happily compiles a data race.

### Guard shared state, and take the *right* lock

```go
// BAD — RLock is a READ lock, and this line writes. Multiple goroutines can
// hold an RLock simultaneously, so this is a data race.
func (a *Application) SetServiceID(serviceID serviceloader.UUID) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.serviceID = domain.UUID(serviceID)
}

// GOOD
func (a *Application) SetServiceID(serviceID serviceloader.UUID) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.serviceID = domain.UUID(serviceID)
}
```

`RWMutex` gives you two locks: `RLock` (many readers at once) and `Lock` (one writer, exclusive).
Writing under `RLock` compiles, runs, and corrupts memory under load. **Every write takes `Lock`.
Every read of that same field takes `RLock`.** A field guarded on write but read unguarded is still
a race.

### Declare what a mutex protects

Put the mutex immediately above the fields it guards, and say so:

```go
// mu guards serviceID and cancel. Both are written from the goroutine
// that boots the service and read from the one that stops it.
mu        sync.RWMutex
serviceID domain.UUID
cancel    context.CancelFunc
```

A `sync.Mutex` floating at the top of a struct with ten fields tells the next reader nothing.

### Verify with the race detector

```bash
go test -race ./...
go run -race ./cmd
```

The race detector finds races that actually happened at runtime. It is the only reliable way to
check this class of bug — use it whenever you touch concurrent code.

### Every goroutine needs a known lifetime

Before you type `go`, answer: *when does this stop, and who waits for it?*

```go
for _, handler := range a.handlers {
	a.wg.Add(1)

	go func(h domain.Service) {
		defer a.wg.Done()
		...
	}(handler)
}
```

`wg.Add` goes **before** the `go`, never inside the goroutine — otherwise `Wait` can run before
`Add` and return immediately. `defer wg.Done()` is the first line so it fires on every return path.

Cancellation flows through `context.Context`: it is always the first parameter, always named `ctx`,
and it is never stored in a struct.

---

## 6. Transactions

Some pairs of writes have to agree unconditionally. `POST /deployments` writes a `deployments` row
*and* enqueues the job that acts on it; if those two can disagree, a deploy is accepted by the API
and then silently never happens — a row stuck at `queued` forever, with no error and no retry,
because the row survived and the intent didn't.

The only way two facts agree unconditionally is one commit.

### Open the transaction in the use case, with `RunInTx`

`postgres.Transactor` is the only thing that opens a transaction. It puts the `bun.Tx` into the
context it hands your function, so every call made inside joins it:

```go
func (uc *UseCase) Create(ctx context.Context, req CreateRequest) (domain.UUID, error) {
	dep := newDeployment(uc.generator(), req)

	if err := uc.transactor.RunInTx(ctx, func(ctx context.Context, _ bun.Tx) error {
		if err := uc.repo.Insert(ctx, dep); err != nil {
			return err
		}

		return uc.jobs.Enqueue(ctx, domain.RunDeployment, payload)
	}); err != nil {
		return domain.UUID{}, fmt.Errorf("create deployment: %w", err)
	}

	return dep.Identifier, nil
}
```

Shadow the outer `ctx` with the one `RunInTx` passes in — that inner one is what carries the
transaction. Naming it something else (`txCtx`) leaves two contexts in scope and invites the next
reader to pass the wrong one, which fails silently by opening its own connection outside the
transaction.

### The transaction travels in the context, not in signatures

Repositories never take a `bun.Tx` or `bun.IDB` parameter. They route every query through
`postgres.IDB(ctx, fallback)`, which returns the ambient transaction when there is one:

```go
func (r *Deployments) Insert(ctx context.Context, dep *models.Deployments) error {
	if _, err := postgres.IDB(ctx, r.db).NewInsert().Model(dep).Exec(ctx); err != nil {
		return fmt.Errorf("insert deployment: %w", err)
	}

	return nil
}
```

The obvious alternative is to thread the transaction explicitly —
`Enqueue(ctx, tx, kind, payload)` taking a concrete `bun.Tx` — so that calling it outside a
transaction doesn't compile. `bun.IDB` is also satisfied by `*bun.DB`, so only the concrete type
actually forbids it. We deliberately don't do this, and the reason is worth understanding,
because we are giving up a compile-time guarantee.

Our transactions span **use cases**, not just repositories: `deployments.Create` calls
`jobs.Enqueue`, an account use case calls a balance use case. Threading the transaction
explicitly puts `bun.Tx` in the *public signature* of every use case that can take part in one,
so `internal/core` — the layer whose whole point is not knowing how storage works — ends up
importing bun everywhere. That cost is paid by every use case forever, to catch a mistake a test
catches once.

### A port that requires a transaction must fail without one

The `fallback` argument is where this gets safe or unsafe. Passing the pool (`r.db`) means "join
the transaction if there is one, otherwise run standalone" — correct for an ordinary read. Passing
`nil` means "there must be a transaction," and turns its absence into a hard failure:

```go
// GOOD — a dual write is the exact thing this port exists to prevent, so a
// silent pool fallback is not an option here.
db := postgres.IDB(ctx, nil)
if db == nil {
	return errs.ErrNoTransaction
}
```

Put the requirement in the interface too, so nobody implements it the other way:

```go
// Insert enqueues msg inside the caller's unit of work. Implementations MUST
// fail when no transaction is ambient — a silent pool fallback would
// reintroduce the dual write this design exists to prevent.
Insert(ctx context.Context, msg domain.Message, opts ...domain.JobOption) error
```

### Every producing use case needs a transaction test

This is the part you have to actually do, because it is the guarantee we chose *instead of* the
compiler's.

With an explicit `bun.Tx` parameter, a missing transaction is a build failure at every call site,
including the ones nobody tested. With an ambient transaction it is a **runtime** failure: the code
compiles, passes review, ships, and returns `ErrNoTransaction` in production the first time that
path runs.

So when you write a use case that enqueues a job, write two tests alongside it:

- calling it outside a transaction returns `ErrNoTransaction`;
- a transaction that rolls back leaves nothing in the queue — against a real Postgres, not a mock.
  A mock will happily "roll back" a write it never made, so it proves nothing about the property
  you care about.

A missing `RunInTx` is not a bug any linter in `.golangci.yml` will find for you.

---

## 7. Receivers, interfaces, and a trap worth memorising

### Pointer vs value receivers

Use a **pointer receiver** when the method mutates the receiver or the type contains a mutex.
Use a **value receiver** for small immutable types.

Here's the trap that bit this codebase:

```go
// BAD — pointer receiver.
func (u *UUID) String() string { ... }
```

Method sets: `*UUID` has both value and pointer methods; `UUID` has only value methods. So with a
pointer receiver, `UUID` **does not** satisfy `fmt.Stringer` — only `*UUID` does. The result:

```go
a.logger.InfoContext(ctx, "...", "identifier", a.serviceID)   // logs a raw [16]byte array
a.logger.InfoContext(ctx, "...", "identifier", a.serviceID.String()) // logs hex
```

Both compile. One produces `[0,206,71,203,...]` in your logs. The fix is a value receiver, so the
type formats correctly everywhere:

```go
// GOOD — UUID (not just *UUID) now satisfies fmt.Stringer.
func (u UUID) String() string {
	return hex.EncodeToString(u[:])
}
```

### Don't mix receiver types on one type

Once `String()` was a value receiver, `UUID` had one value method and one pointer method
(`FromString`, which mutated the receiver). The `recvcheck` linter rejects that mix, and it's right
to: which methods you can call then depends on whether you're holding a `UUID` or a `*UUID`, which is
exactly the confusion that caused the logging bug above.

The idiomatic fix is a **constructor function** instead of a mutating method — the same shape as
`uuid.Parse`, `time.Parse`, `strconv.Atoi`:

```go
// BAD — mutating method forces a pointer receiver.
func (u *UUID) FromString(s string) error

// GOOD — returns a new value, so every method can stay on a value receiver.
func ParseUUID(s string) (UUID, error)
```

Rule of thumb: a small immutable value type (like `UUID`) should have *only* value receivers, and be
constructed by a function. A type with a mutex or one that mutates itself should have *only* pointer
receivers.

### Accept interfaces, return structs

Take the narrowest interface you need as a parameter; return concrete types. `domain.Service` is our
one real interface — small (three methods) and defined by the *consumer* (`core`), not the producer.
That's the Go way round: don't define an interface until you have a second implementation or a test
that needs one.

The one sanctioned exception is the `Option` interface from §4 — the functional options pattern only
works if `With*` returns the interface.

### Verify interface compliance at compile time

Go's interface satisfaction is implicit: nothing in `Server` says "I implement `domain.Service`". So
if you rename a method, the code still compiles and only breaks where the value is *used* as the
interface — possibly in another package, possibly at runtime.

Pin it down with a blank assignment, right above the type:

```go
// Verify at compile time that a Server is a handler the application can run.
var _ domain.Service = (*Server)(nil)
```

If `*Server` ever stops satisfying `domain.Service`, **this line** fails to compile, with the error
pointing at the type that broke rather than at some distant call site. The right-hand side is the
zero value of the asserted type: `nil` for pointers, slices and maps; `T{}` for a struct.

Add one for every type that exists to satisfy an interface. We have two:
`*core.Application` (asserted against `serviceloader.Service`) and `*http.Server`.

---

## 8. Naming and package layout

- **Package names**: short, lowercase, single word, no underscores, no plurals. The package name is
  part of every call site — `core.NewApplication` reads well, `coreutils.NewApplicationHelper`
  doesn't.
- **Don't stutter**: inside package `core`, the type is `Application`, not `CoreApplication`.
  Callers write `core.Application`.
- **Avoid `utils`, `helpers`, `common`, `misc`.** They're where code goes to hide. A package should
  be named for what it *provides*. We still have `internal/utils` — it holds ID generators and
  should become `internal/id`. Don't add to it.
- **Don't shadow the standard library.** We have `internal/infra/transport/http`, which forces every
  file that needs the real `net/http` into an alias. Known debt; don't repeat it in new packages.
- **`internal/` is enforced by the compiler.** Code under `internal/` can only be imported by its
  parent tree. Nesting it twice (`internal/infra/internal/infra/...`, which exists here and is dead
  code) makes it unreachable from anywhere useful. One `internal/` per module is enough.
- **Abbreviations keep their case**: `URL`, `HTTP`, `ID`, `UUID` — so `serviceID`, not `serviceId`;
  `HTTPServer`, not `HttpServer`.

### Prefix unexported package-level vars and consts with `_`

This one surprises everybody, and it is a real Uber rule:

```go
// BAD — `files` looks like a local variable at every use site.
//go:embed *.toml
var files embed.FS

// GOOD — the underscore says "this is package scope" wherever you read it.
//go:embed *.toml
var _configFiles embed.FS

const _defaultServiceName = "dockzilla-application"
```

Top-level names are visible in every file of the package. Without the prefix, reading
`files` halfway down a 200-line file gives you no clue whether it's a local, a parameter, or shared
package state — and shadowing one by accident is silent. The `_` makes it obvious.

This applies to **unexported** top-level `var`s and `const`s only. Exported ones (`ErrNotFound`) keep
their normal names, and locals are never prefixed.

### Keep lines under 99 characters

A soft limit, not a hard one — Uber's number, enforced by `lll`. Long log calls are the usual
offender; break them one key-value pair per line:

```go
a.logger.WarnContext(ctx, "failed to start service",
	"name", h.Name(),
	"error", err,
)
```

---

## 9. Imports

Uber's rule is exactly two groups, separated by one blank line: **standard library first, everything
else second.** Our own `dockzilla/...` packages are part of "everything else" — they do *not* get a
third group of their own.

```go
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"dockzilla/pkg/domain"
	serviceloader "github.com/zixyos/goloader/service"
)
```

`gofmt` sorts within a group but will not create the groups for you — `gci` does, and
`task backend:lint:fix` will rewrite the block for you. Don't hand-maintain it.

> **Why not `goimports`?** Uber's Linting section names it, but current versions of `goimports`
> split module-local imports (`dockzilla/...`) into a *third* group, which contradicts the guide's
> own two-group rule. The two cannot both be satisfied, so we follow the rule and use `gofmt` + `gci`.
> This is noted in `.golangci.yml` so nobody "fixes" it back.

Only alias an import to resolve a genuine collision (`ginihttp` inside our own `http` package).
Never alias for brevity.

---

## 10. Configuration

Config is loaded from `cmd/config.<APP_ENV>.toml` (embedded via `go:embed`) with environment
variables layered on top, and decoded into structs by koanf tags.

**Nothing checks that a TOML key matches its `koanf` tag.** A typo doesn't error — it silently
leaves the zero value. Two live examples from this codebase:

```toml
# BAD — the struct field is `koanf:"enabled"`, so this key was ignored and
# telemetry was silently off for the life of the project.
enable = true

# BAD — the field is a time.Duration, so a bare int is *nanoseconds*.
# This meant a 10ns read timeout: every request would time out.
read_timeout = 10

# GOOD
enabled = true
read_timeout = "10s"
```

So: when you add a config key, **run the service and confirm the value actually arrived**. Durations
are always quoted strings (`"10s"`, `"1m30s"`).

---

## 11. Tests

There are currently **zero tests** in the backend. That's the single biggest gap in the project, and
new code is the place to start closing it.

- Table-driven tests are the Go norm; use subtests (`t.Run`) so failures name themselves.
- Name the file `foo_test.go` next to `foo.go`.
- Use `t.Parallel()` where the test allows it, and always run `go test -race ./...`.
- Don't reach for a mocking framework by default. Our interfaces are small — a hand-written fake
  implementing `domain.Service` is three methods and clearer than generated code.

---

## Pre-PR checklist

```bash
task backend:check           # build + vet + lint + test -race, all must pass
task backend:dev             # and confirm it actually starts
```

The linter covers most of this document mechanically. What it *can't* check is judgement, so reread
your own diff and ask:

- [ ] Did I avoid adding doc comments to **unexported** functions? *(the linter enforces that
      exported ones are documented, but it won't stop you over-commenting)*
- [ ] Can any constructor return a non-nil interface holding a nil pointer?
- [ ] Is every shared field written under `Lock` **and** read under `RLock`? *(no linter finds this
      — only `go test -race` and your own reading do)*
- [ ] Do the writes that must agree happen inside one `RunInTx`, with a test that proves a rollback
      leaves nothing behind?
- [ ] Does every goroutine I started have an owner that waits for it?
- [ ] Any new `panic` outside of genuine programmer error?
- [ ] Did I verify new config keys actually load, rather than assuming?
- [ ] Do my error messages read as a sentence once wrapped, without repeating "failed to"?

The first three are the ones that have actually bitten this codebase. The linter caught none of
them — it can tell you a doc comment is missing, not that a lock is the wrong kind.
