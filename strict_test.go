package hx

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(nil) })
	return &buf
}

func TestReadingOutsideAnEffectSaysHowToFixIt(t *testing.T) {
	if strictReads {
		t.Skip("in strict mode it is a panic, see TestStrictReads")
	}
	buf := captureLog(t)
	s := Signal(1)
	s.Get()
	got := buf.String()
	for _, want := range []string{"Cannot call Get if there is no effect", "strict_test.go", "BindText", "UntrackGet"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the warning has no %q: %s", want, got)
		}
	}
}

func TestReadsThatAreFineDoNotWarn(t *testing.T) {
	buf := captureLog(t)
	s := Signal(1)
	EffectFunc(func() { s.Get() }) // inside an effect
	UntrackGet[int](s)             // on purpose
	Untrack(func() { s.Get() })
	if buf.Len() != 0 {
		t.Fatalf("it warned: %s", buf.String())
	}
}
