package library

import "context"

func sendWithContext[T any](ctx context.Context, ch chan<- T, value T) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- value:
		return true
	}
}
