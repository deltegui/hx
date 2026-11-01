package ssr_test

import (
	"strings"
	"testing"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/ssr"
)

func TestSvgConstructors(t *testing.T) {
	got := ssr.Render(func() hx.INode {
		return hx.Svg().Body(
			hx.Defs().Body(hx.LinearGradient().Attribute("id", "g").Body(hx.Stop().Attribute("offset", "0"))),
			hx.G().Body(hx.Circle().Attribute("r", "5"), hx.Rect(), hx.Line(), hx.Polygon(), hx.Polyline(), hx.Ellipse()),
			hx.Text().Body(hx.Tspan().Text("hi")),
			hx.SvgTitle().Text("t"))
	})
	for _, want := range []string{"<defs>", "<lineargradient id=\"g\">", "<stop offset=\"0\">", "<g>", "<circle r=\"5\">", "<rect>", "<tspan>hi</tspan>", "<title>t</title>"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
