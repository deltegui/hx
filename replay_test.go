package hx

import (
	"strings"
	"testing"
)

func TestReplayScriptIsOneInlineScript(t *testing.T) {
	s := ReplayScript()
	if !strings.HasPrefix(s, "<script>") || !strings.HasSuffix(s, "</script>") {
		t.Fatalf("not a script element: %q", s)
	}
	inner := s[len("<script>") : len(s)-len("</script>")]
	if strings.Contains(inner, "<") {
		t.Fatalf("the script could close itself: %q", inner)
	}
	if len(s) > 800 {
		t.Fatalf("the script grew to %d bytes", len(s))
	}
	for _, want := range []string{"__hxq", "__hxqStop", "__hxLive", `"click"`, `"input"`, `"change"`, `"keydown"`} {
		if !strings.Contains(s, want) {
			t.Errorf("the script lacks %s", want)
		}
	}
}
