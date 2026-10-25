package port

import "context"

// Transactor runs fn inside a single database transaction. Repository calls made with the
// ctx passed to fn take part in that transaction; returning an error from fn rolls it back.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
