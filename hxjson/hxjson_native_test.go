//go:build !(js && wasm)

package hxjson

import (
	"context"
	"errors"
	"testing"

	"github.com/deltegui/hx/hxhttp"
)

func TestSendOutsideTheBrowser(t *testing.T) {
	if _, err := Get[int](context.Background(), "/x"); !errors.Is(err, hxhttp.ErrNotInBrowser) {
		t.Fatalf("err=%v", err)
	}
	if _, err := Post[int](context.Background(), "/x", map[string]int{"a": 1}); !errors.Is(err, hxhttp.ErrNotInBrowser) {
		t.Fatalf("err=%v", err)
	}
	if _, err := Send[int](context.Background(), "PUT", "/x", make(chan int)); err == nil {
		t.Fatal("a body that is not JSON must fail before the request")
	}
}
