//go:build js && wasm

package hxhttp

import (
	"context"
	"errors"
	"testing"

	"github.com/deltegui/hx"
)

// Inside an event handler Do cannot wait for the answer: Go would stop with a
// deadlock. It must say so with an error, before it starts anything.
func TestDoInsideACallbackFailsInsteadOfDeadlocking(t *testing.T) {
	hx.RunCallback(func() {
		if _, err := Do(context.Background(), Request{URL: "/x"}); !errors.Is(err, ErrInCallback) {
			t.Fatalf("Do: err=%v, want ErrInCallback", err)
		}
		if _, err := (File{}).Bytes(context.Background()); !errors.Is(err, ErrInCallback) {
			t.Fatalf("File.Bytes: err=%v, want ErrInCallback", err)
		}
	})
}
