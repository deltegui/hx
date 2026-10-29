//go:build !(js && wasm)

package hx

import (
	"context"
	"errors"
)

// The requests are made by the browser. On a server these give an error, so
// code shared with the browser builds, and the actions that use it fail clearly
// instead of doing something unexpected.

type domFile struct{}

// Do makes the request. It only works in the browser: on a server it returns ErrNotInBrowser.
func Do(ctx context.Context, request Request) (Response, error) {
	return Response{}, ErrNotInBrowser
}

// FilesOf returns the files chosen in a file input. On a server there are none.
func FilesOf(input *VNode) []File { return nil }

// Bytes reads the content of the file. It only works in the browser.
func (f File) Bytes(ctx context.Context) ([]byte, error) {
	return nil, errors.Join(ErrNotInBrowser)
}
