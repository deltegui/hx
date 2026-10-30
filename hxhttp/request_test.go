package hxhttp

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestBodyEncoding(t *testing.T) {
	ct, data, _ := TextBody("text/csv", "a,b").encode()
	if ct != "text/csv" || string(data) != "a,b" {
		t.Fatalf("text: %q %q", ct, data)
	}
	// a url.Values can be passed as it is
	ct, data, _ = FormValues(url.Values{"b": {"2"}, "a": {"1", "x y&z=é"}}).encode()
	if ct != "application/x-www-form-urlencoded" || string(data) != "a=1&a=x+y%26z%3D%C3%A9&b=2" {
		t.Fatalf("values: %q %q", ct, data)
	}
	// the same as the standard library writes
	values := url.Values{"name": {"Ada Lovelace"}, "k~v-_.": {"/?#[]@!$'()*+,;="}, "ü": {"ñ"}}
	if got := encodeValues(values); got != values.Encode() {
		t.Fatalf("encodeValues differs from url.Values.Encode:\n got %s\nwant %s", got, values.Encode())
	}
	if _, _, err := FormOf(nil).encode(); err == nil {
		t.Fatal("a form is built by the browser: encode must say so")
	}
}

func TestMultipartBuilder(t *testing.T) {
	m := NewMultipart().Field("name", "Ada").Bytes("avatar", "a.png", "image/png", []byte{1, 2}).File("doc", File{Name: "d.txt", Size: 3})
	if len(m.parts) != 3 || m.parts[0].kind != partField || m.parts[1].kind != partBytes || m.parts[2].kind != partFile {
		t.Fatalf("parts=%+v", m.parts)
	}
	if m.Body().kind != bodyMultipart {
		t.Fatal("wrong body kind")
	}
}

func TestResponse(t *testing.T) {
	ok := Response{Status: 200, body: []byte(`{"n":3}`)}
	if !ok.OK() || ok.Err() != nil || ok.Text() != `{"n":3}` || string(ok.Bytes()) != `{"n":3}` {
		t.Fatalf("ok response: %v", ok.Err())
	}
	bad := Response{Status: 422, StatusText: "Unprocessable", body: []byte("  the email is not valid\n")}
	var httpErr *HTTPError
	if bad.OK() || !errors.As(bad.Err(), &httpErr) || httpErr.Status != 422 || httpErr.Body == "" {
		t.Fatalf("bad response: %v", bad.Err())
	}
	if httpErr.Error() != "422 Unprocessable: the email is not valid" {
		t.Fatalf("message: %q", httpErr.Error())
	}
	if (&HTTPError{Status: 500, StatusText: "Oops"}).Error() != "500 Oops" {
		t.Fatal("an error without body")
	}
	long := &HTTPError{Status: 500, StatusText: "x", Body: strings.Repeat("a", 500)}
	if len(long.Error()) > 230 {
		t.Fatalf("a long body must be cut, got %d", len(long.Error()))
	}
}

func TestProgressFraction(t *testing.T) {
	if (Progress{Loaded: 50, Total: 200}).Fraction() != 0.25 {
		t.Fatal("0.25 expected")
	}
	if (Progress{Loaded: 50}).Fraction() != 0 {
		t.Fatal("unknown total must be 0")
	}
}
