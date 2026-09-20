package chart

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderModel(t *testing.T, config Config, root, children templ.Component) map[string]any {
	t.Helper()
	var out bytes.Buffer
	chart := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return root.Render(templ.WithChildren(ctx, children), w)
	})
	if err := Container(ContainerProps{Config: config}).Render(templ.WithChildren(context.Background(), chart), &out); err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(out.String(), `<script type="application/json" data-tui-chart-model>`)
	if !ok {
		t.Fatal("model script missing")
	}
	payload, _, ok := strings.Cut(rest, "</script>")
	if !ok {
		t.Fatal("model script not closed")
	}
	var model map[string]any
	if err := json.Unmarshal([]byte(payload), &model); err != nil {
		t.Fatal(err)
	}
	return model
}

func TestLineModelValues(t *testing.T) {
	m := renderModel(t, Config{{Key: "value"}}, LineChart(LineChartProps{Data: []Datum{{"value": 10}, {"value": 20}, {"value": 30}}}), Line(LineProps{DataKey: "value"}))
	s := m["series"].([]any)[0].(map[string]any)
	if len(s["values"].([]any)) != 3 {
		t.Fatalf("values: %v", s["values"])
	}
	if _, ok := s["gaps"]; ok {
		t.Fatalf("full series has gaps: %v", s)
	}
}
