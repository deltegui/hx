package hxjson

import (
	"testing"

	"github.com/deltegui/hx/hxhttp"
)

func TestBody(t *testing.T) {
	body, err := Body(map[string]int{"a": 1})
	if err != nil || body == nil {
		t.Fatalf("body=%v err=%v", body, err)
	}
	if _, err := Body(make(chan int)); err == nil {
		t.Fatal("a value that cannot be JSON must be an error")
	}
}

func TestDecode(t *testing.T) {
	var v struct{ N int }
	if err := Decode(hxhttp.Response{}, &v); err == nil {
		t.Fatal("an empty body is not JSON")
	}
}
