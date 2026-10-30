package hx

import "testing"

func TestRunCallbackMarksItsDynamicExtent(t *testing.T) {
	if InCallback() {
		t.Fatal("not in a callback at the start")
	}
	RunCallback(func() {
		if !InCallback() {
			t.Fatal("must be in a callback inside RunCallback")
		}
		RunCallback(func() {})
		if !InCallback() {
			t.Fatal("an inner callback that ended must not end the outer one")
		}
	})
	if InCallback() {
		t.Fatal("must not be in a callback after it ends")
	}
}

func TestRunCallbackIsUndoneByAPanic(t *testing.T) {
	func() {
		defer func() { recover() }()
		RunCallback(func() { panic("boom") })
	}()
	if InCallback() {
		t.Fatal("a panic left the program marked as in a callback")
	}
}
