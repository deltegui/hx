//go:build js && wasm

package hxhttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/deltegui/hx"
	"strings"
	"syscall/js"
	"time"

	"honnef.co/go/js/dom/v2"
)

// The requests use XMLHttpRequest instead of fetch because it is the only one
// that reports how much of the body has been uploaded.

// underlying is the DOM element of a node. Only the nodes of the browser engine
// have one.
func underlying(node hx.INode) (js.Value, bool) {
	if n, ok := node.(interface{ Underlying() dom.Element }); ok && n.Underlying() != nil {
		return n.Underlying().Underlying(), true
	}
	return js.Null(), false
}

type domFile struct {
	value js.Value
}

// Do makes the request and waits for the answer.
//
// It blocks until the answer arrives, the request fails, ctx is done or the
// timeout passes, so it must run in its own goroutine. Waiting for it directly
// in an event handler would freeze the page: the answer can only be handled
// after the handler returns. An Action does this for you.
//
// A status that is not 2xx is not an error here; look at Response.OK or Response.Err.
func Do(ctx context.Context, request Request) (response Response, err error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	// syscall/js panics when the browser throws. Report it as an error.
	defer func() {
		if recovered := recover(); recovered != nil {
			response, err = Response{}, fmt.Errorf("hxhttp: request failed: %v", recovered)
		}
	}()

	method := strings.ToUpper(request.Method)
	if method == "" {
		method = "GET"
	}

	xhr := js.Global().Get("XMLHttpRequest").New()
	xhr.Call("open", method, request.URL, true)
	xhr.Set("responseType", "arraybuffer")
	if request.Timeout > 0 {
		xhr.Set("timeout", int(request.Timeout/time.Millisecond))
	}
	if request.Credentials {
		xhr.Set("withCredentials", true)
	}

	payload, contentType, err := request.Body.toJS()
	if err != nil {
		return Response{}, err
	}
	hasContentType := false
	for name, value := range request.Header {
		if strings.EqualFold(name, "Content-Type") {
			hasContentType = true
		}
		xhr.Call("setRequestHeader", name, value)
	}
	if contentType != "" && !hasContentType {
		xhr.Call("setRequestHeader", "Content-Type", contentType)
	}

	type outcome struct {
		response Response
		err      error
	}
	done := make(chan outcome, 1)
	finish := func(o outcome) {
		select {
		case done <- o:
		default:
		}
	}

	var callbacks []js.Func
	listen := func(target js.Value, event string, handler func(args []js.Value)) {
		callback := js.FuncOf(func(_ js.Value, args []js.Value) any {
			handler(args)
			return nil
		})
		callbacks = append(callbacks, callback)
		target.Call("addEventListener", event, callback)
	}
	defer func() {
		for _, callback := range callbacks {
			callback.Release()
		}
	}()

	listen(xhr, "load", func([]js.Value) {
		finish(outcome{response: readResponse(xhr)})
	})
	listen(xhr, "error", func([]js.Value) {
		finish(outcome{err: errors.New("hxhttp: network error")})
	})
	listen(xhr, "timeout", func([]js.Value) {
		finish(outcome{err: fmt.Errorf("hxhttp: the request took more than %v: %w", request.Timeout, context.DeadlineExceeded)})
	})
	listen(xhr, "abort", func([]js.Value) {
		finish(outcome{err: context.Canceled})
	})
	if request.OnProgress != nil {
		report := func(upload bool) func([]js.Value) {
			return func(args []js.Value) {
				event := args[0]
				progress := Progress{Upload: upload, Loaded: int64(event.Get("loaded").Float())}
				if event.Get("lengthComputable").Truthy() {
					progress.Total = int64(event.Get("total").Float())
				}
				request.OnProgress(progress)
			}
		}
		listen(xhr, "progress", report(false))
		listen(xhr.Get("upload"), "progress", report(true))
	}

	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			xhr.Call("abort")
		case <-finished:
		}
	}()

	if payload.IsUndefined() {
		xhr.Call("send")
	} else {
		xhr.Call("send", payload)
	}

	result := <-done
	if result.err != nil && ctx.Err() != nil {
		return Response{}, ctx.Err()
	}
	return result.response, result.err
}

func readResponse(xhr js.Value) Response {
	response := Response{
		Status:     xhr.Get("status").Int(),
		StatusText: xhr.Get("statusText").String(),
		Header:     map[string]string{},
	}
	for _, line := range strings.Split(xhr.Call("getAllResponseHeaders").String(), "\r\n") {
		if name, value, ok := strings.Cut(line, ":"); ok {
			response.Header[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
		}
	}
	if buffer := xhr.Get("response"); !buffer.IsNull() && !buffer.IsUndefined() {
		bytes := js.Global().Get("Uint8Array").New(buffer)
		response.body = make([]byte, bytes.Get("length").Int())
		js.CopyBytesToGo(response.body, bytes)
	}
	return response
}

// toJS prepares what XMLHttpRequest.send takes, and the content type to send it
// with. Without a body it is undefined and the content type is empty.
func (b *Body) toJS() (payload js.Value, contentType string, err error) {
	if b == nil {
		return js.Undefined(), "", nil
	}
	switch b.kind {
	case bodyText, bodyValues:
		contentType, data, err := b.encode()
		if err != nil {
			return js.Undefined(), "", err
		}
		return js.ValueOf(string(data)), contentType, nil

	case bodyForm:
		element, ok := underlying(b.form)
		if b.form == nil || !ok {
			return js.Undefined(), "", errors.New("hxhttp: FormOf needs a form that is already in the page")
		}
		// The browser sets the content type, with the boundary.
		return js.Global().Get("FormData").New(element), "", nil

	case bodyMultipart:
		formData := js.Global().Get("FormData").New()
		for _, p := range b.multipart.parts {
			switch p.kind {
			case partField:
				formData.Call("append", p.name, p.value)
			case partFile:
				formData.Call("append", p.name, p.file.value, p.file.Name)
			case partBytes:
				bytes := js.Global().Get("Uint8Array").New(len(p.data))
				js.CopyBytesToJS(bytes, p.data)
				options := js.Global().Get("Object").New()
				options.Set("type", p.mime)
				blob := js.Global().Get("Blob").New([]any{bytes}, options)
				formData.Call("append", p.name, blob, p.filename)
			}
		}
		return formData, "", nil
	}
	return js.Undefined(), "", errors.New("hxhttp: unknown body")
}

// FilesOf returns the files chosen in a file input. The input has to be in the
// page already.
func FilesOf(input hx.INode) []File {
	if input == nil {
		return nil
	}
	element, ok := underlying(input)
	if !ok {
		return nil
	}
	list := element.Get("files")
	if list.IsUndefined() || list.IsNull() {
		return nil
	}
	files := make([]File, list.Length())
	for i := range files {
		file := list.Index(i)
		files[i] = File{
			Name:    file.Get("name").String(),
			Size:    int64(file.Get("size").Float()),
			Type:    file.Get("type").String(),
			domFile: domFile{value: file},
		}
	}
	return files
}

// Bytes reads the content of the file. It blocks, so call it from a goroutine.
func (f File) Bytes(ctx context.Context) ([]byte, error) {
	buffer, err := await(ctx, f.value.Call("arrayBuffer"))
	if err != nil {
		return nil, err
	}
	bytes := js.Global().Get("Uint8Array").New(buffer)
	data := make([]byte, bytes.Get("length").Int())
	js.CopyBytesToGo(data, bytes)
	return data, nil
}

// await waits for a JavaScript promise.
func await(ctx context.Context, promise js.Value) (js.Value, error) {
	type result struct {
		value js.Value
		err   error
	}
	done := make(chan result, 1)
	resolved := js.FuncOf(func(_ js.Value, args []js.Value) any {
		done <- result{value: args[0]}
		return nil
	})
	rejected := js.FuncOf(func(_ js.Value, args []js.Value) any {
		done <- result{err: errors.New(args[0].Get("message").String())}
		return nil
	})
	defer resolved.Release()
	defer rejected.Release()
	promise.Call("then", resolved).Call("catch", rejected)

	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		return js.Undefined(), ctx.Err()
	}
}
