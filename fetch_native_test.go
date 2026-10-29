//go:build !(js && wasm)

package hx

import (
	"context"
	"errors"
	"testing"
)

func TestRequestsFailClearlyOutsideTheBrowser(t *testing.T) {
	if _, err := Do(context.Background(), Request{URL: "/x"}); !errors.Is(err, ErrNotInBrowser) {
		t.Fatalf("err=%v", err)
	}
	if FilesOf(Input()) != nil {
		t.Fatal("no files on a server")
	}
	if _, err := (File{}).Bytes(context.Background()); !errors.Is(err, ErrNotInBrowser) {
		t.Fatalf("err=%v", err)
	}
}
