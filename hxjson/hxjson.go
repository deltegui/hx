// Package hxjson sends and reads JSON with hx.Do.
//
// It is a separate package on purpose. encoding/json adds around 250 KB to a
// WASM file, and Go keeps it as soon as a package imports it, even if the code
// never calls it. The programs that do not use JSON do not pay for it.
//
// The functions block until the answer arrives, so call them from a goroutine,
// for example inside an hx.Action.
package hxjson

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deltegui/hx"
)

// Body returns a request body that sends v encoded as JSON.
func Body(v any) (*hx.Body, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("hxjson: the value cannot be sent as JSON: %w", err)
	}
	return hx.TextBody("application/json", string(data)), nil
}

// Decode reads the JSON body of an answer into v.
func Decode(response hx.Response, v any) error {
	if err := json.Unmarshal(response.Bytes(), v); err != nil {
		return fmt.Errorf("hxjson: the answer is not the JSON that was expected: %w", err)
	}
	return nil
}

// Send sends body (any value, or nil for none) as JSON with that method and URL,
// and decodes the JSON answer into T. An answer whose status is not 2xx is
// returned as an *hx.HTTPError, with the body of the answer inside.
//
// To send something else, for example headers, build the request with hx.Do and
// read the answer with Decode.
func Send[T any](ctx context.Context, method, url string, body any) (T, error) {
	var zero T
	request := hx.Request{Method: method, URL: url, Header: map[string]string{"Accept": "application/json"}}
	if body != nil {
		encoded, err := Body(body)
		if err != nil {
			return zero, err
		}
		request.Body = encoded
	}
	response, err := hx.Do(ctx, request)
	if err != nil {
		return zero, err
	}
	if err := response.Err(); err != nil {
		return zero, err
	}
	var result T
	if len(response.Bytes()) == 0 {
		return result, nil
	}
	if err := Decode(response, &result); err != nil {
		return zero, err
	}
	return result, nil
}

// Get is Send with GET and no body.
func Get[T any](ctx context.Context, url string) (T, error) {
	return Send[T](ctx, "GET", url, nil)
}

// Post is Send with POST.
func Post[T any](ctx context.Context, url string, body any) (T, error) {
	return Send[T](ctx, "POST", url, body)
}
