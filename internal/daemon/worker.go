package daemon

import "context"

// Worker defines the implicit interface for background tasks.
// Any type that implements these methods can be managed by the daemon.
type Worker interface {
	Name() string
	Start(ctx context.Context) error
}
