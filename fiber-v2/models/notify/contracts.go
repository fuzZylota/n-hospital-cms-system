// Package notify defines transport-independent, process-local notification
// delivery. Authorization and serialization belong to the calling application.
package notify

import "context"

// RoomID identifies a delivery group, not an authorization boundary.
type RoomID string

// ConnectionID identifies one transport session, never a user. Callers generate
// it; the hub neither parses it nor derives a UserID from it.
type ConnectionID string

// UserID identifies the authenticated user; empty means unavailable/anonymous.
type UserID string

// BranchID identifies a hospital branch (sid), not a medical department (brid).
type BranchID string

// Role carries an application role without granting permission by itself.
type Role string

// Protocol carries the consumer's notification category, such as "kullanici".
// It does not imply authentication or dictate a wire transport.
type Protocol string

// Metadata contains only the recipient attributes evidenced by current callers.
// It is a value snapshot. Role/BranchID may be empty; they are not a substitute
// for current server-side user/branch authorization. No mutable fields are held.
type Metadata struct {
	UserID   UserID
	BranchID BranchID
	Role     Role
	Protocol Protocol
}

// Client separates transport identity from recipient metadata. Each successful
// registration owns one room; ConnectionID is unique across the whole hub.
type Client struct {
	ID       ConnectionID
	Metadata Metadata
}

// Predicate selects recipients from a value snapshot, outside all hub locks.
// It is called again immediately before queued delivery, so it must tolerate
// repeated and concurrent calls and must read current authorization state.
// Nil selects all. It must terminate promptly and be safe for concurrent calls
// from different broadcasts. A panic aborts that broadcast before any enqueue,
// returning ErrPredicatePanic without exposing the panic value. Side effects in
// a predicate are not rolled back. Pure, precomputed authorization is preferred.
type Predicate func(Client) bool

// Send delivers an opaque payload to one transport. The hub calls it serially
// for that registration. The payload is a private copy: the callback may retain
// or mutate it. Send MUST return promptly on ctx cancellation; the adapter owns
// socket close/deadlines, including interrupting blocked I/O. The hub does not
// invoke a competing Close callback or spawn a goroutine per send.
//
// Errors/panics remove the client and become fixed safe errors; their original
// text/value is never retained. Callbacks must not wait for their own Done or
// a successful Shutdown, which necessarily waits for the callback to return.
type Send func(ctx context.Context, payload []byte) error

// Registration is a generation-specific lifecycle handle, so an old cleanup
// cannot unregister a later connection that reuses the same ID.
type Registration interface {
	// Unregister is idempotent and nonblocking. It removes routing immediately,
	// cancels Send and discards pending messages. An already dispatched send may
	// finish. Wait for Done before releasing transport resources or reusing ID.
	Unregister()
	// Done closes after the writer has finished all sends and queue cleanup.
	Done() <-chan struct{}
	// Err returns the first terminal reason; nil while active or after explicit
	// Unregister. After Done its value is final. Raw transport errors never escape.
	Err() error
}

// BroadcastResult reports queue admission, not successful network delivery.
type BroadcastResult struct {
	Enqueued     int // Clients that accepted a private payload copy.
	Disconnected int // Clients removed by this call because their queue was full.
}

// Hub routes best-effort notifications without persistence, retry or replay.
// Every method is safe for concurrent calls. Invalid arguments are rejected
// before checking hub state; otherwise a closed hub takes precedence over
// registration cancellation or duplicate identity.
type Hub interface {
	// Register starts one writer. ctx is the client lifetime, not a request that
	// ends immediately after registration. Nil ctx/send or empty room/ID is invalid.
	// Duplicate IDs are rejected (even in another room) until the old Done closes;
	// registration never replaces a live or retiring writer. Metadata is copied.
	Register(ctx context.Context, room RoomID, client Client, send Send) (Registration, error)
	// Broadcast takes a recipient snapshot, evaluates predicates, then admits the
	// entire broadcast under a short lock. Sequential calls preserve client FIFO;
	// concurrent calls are ordered by admission, not invocation/predicate start.
	// Clients added after the snapshot do not receive it; removed clients are skipped.
	// A recipient that fails the second predicate check is skipped before Send.
	// Missing rooms are successful no-ops; empty room IDs are invalid.
	// A full queue disconnects only that client and never waits for transport I/O.
	// The input is copied before predicates and per recipient; caller must not
	// mutate it during this call, but may reuse it immediately after return.
	Broadcast(room RoomID, payload []byte, predicate Predicate) (BroadcastResult, error)
	// Shutdown permanently rejects valid Register/Broadcast calls with ErrClosed, discards
	// queued messages, cancels writers and waits for their completion. Repeated
	// calls are safe. ctx bounds only the wait: expiry does not reopen the hub.
	// A nil context is invalid and does not initiate shutdown. An already canceled
	// non-nil context still initiates shutdown. Completion wins if already observed.
	Shutdown(ctx context.Context) error
}

// ErrorCode is an immutable, comparable error with fixed non-sensitive text.
// errors.Is works by equality; no caller data or underlying cause is stored.
type ErrorCode uint8

const (
	ErrInvalidConfig   ErrorCode = iota + 1 // Queue capacity must be positive.
	ErrInvalidArgument                      // Nil context/send or empty room/connection ID.
	ErrDuplicateClient                      // ID is still owned by a writer.
	ErrClosed                               // Hub shutdown was initiated.
	ErrSlowClient                           // Bounded queue overflowed.
	ErrSendFailed                           // Transport returned an error.
	ErrSendPanic                            // Transport panicked.
	ErrPredicatePanic                       // Recipient predicate panicked.
)

// Error returns a fixed description without formatting caller-provided data.
func (e ErrorCode) Error() string {
	switch e {
	case ErrInvalidConfig:
		return "notification: invalid queue capacity"
	case ErrInvalidArgument:
		return "notification: invalid argument"
	case ErrDuplicateClient:
		return "notification: connection already registered"
	case ErrClosed:
		return "notification: hub closed"
	case ErrSlowClient:
		return "notification: slow client disconnected"
	case ErrSendFailed:
		return "notification: send failed"
	case ErrSendPanic:
		return "notification: send panicked"
	case ErrPredicatePanic:
		return "notification: predicate panicked"
	default:
		return "notification: unknown error"
	}
}
