package hx

import "testing"

func TestSafeAttribute(t *testing.T) {
	bad := []string{"javascript:alert(1)", " JavaScript:alert(1)", "java\tscript:alert(1)", "\x01javascript:x", "data:text/html,<b>", "vbscript:x", "java\nscript:x"}
	for _, v := range bad {
		if got := SafeAttribute("href", v); got != UnsafeURL {
			t.Errorf("href %q = %q, want unsafe", v, got)
		}
	}
	good := []string{"/a", "a/b:c", "?q=a:b", "#x:y", "https://x.y", "HTTP://x", "mailto:a@b", "tel:1", "page.html", ""}
	for _, v := range good {
		if got := SafeAttribute("src", v); got != v {
			t.Errorf("src %q = %q, want unchanged", v, got)
		}
	}
	if SafeAttribute("title", "javascript:x") != "javascript:x" {
		t.Error("only URL attributes are checked")
	}
	AllowURLScheme("whatsapp")
	if SafeAttribute("href", "whatsapp://send") != "whatsapp://send" {
		t.Error("an allowed scheme passes")
	}
}
