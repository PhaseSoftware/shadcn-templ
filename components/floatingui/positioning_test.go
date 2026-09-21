package floatingui

import (
	"os"
	"strings"
	"testing"
)

func requireSource(t *testing.T, path string, wants ...string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range wants {
		if !strings.Contains(string(source), want) {
			t.Fatalf("%s is missing %q", path, want)
		}
	}
}

func requireSourceCount(t *testing.T, path, want string, count int) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(source), want); got != count {
		t.Fatalf("%s contains %q %d times, want %d", path, want, got, count)
	}
}

func TestPortaledPositioningStrategiesMatchTheirCoordinateSystems(t *testing.T) {
	absoluteComponents := map[string]string{
		"combobox":     `class="pointer-events-none isolate absolute inset-auto`,
		"dropdownmenu": `class="pointer-events-none isolate absolute inset-auto`,
		"popover":      `class="pointer-events-none isolate absolute inset-auto`,
		"hovercard":    `"absolute inset-auto left-0 top-0`,
		"tooltip":      `"pointer-events-none absolute inset-auto left-0 top-0`,
	}
	for component, positionerClass := range absoluteComponents {
		requireSource(t, "../"+component+"/"+component+".js", `strategy: "absolute"`)
		requireSource(t, "../"+component+"/"+component+".templ", positionerClass)
	}
	// The dropdown's root popup is portaled to <body> and positions absolute.
	// Its sub content stays nested inside that popup, so it positions fixed
	// to escape the popup's overflow clip, the context menu's pattern.
	requireSourceCount(t, "../dropdownmenu/dropdownmenu.js", `strategy: "absolute"`, 1)
	requireSourceCount(t, "../dropdownmenu/dropdownmenu.js", `strategy: "fixed"`, 1)
	requireSource(t, "../dropdownmenu/dropdownmenu.templ", `"hidden fixed inset-auto left-0 top-0`)

	requireSourceCount(t, "../contextmenu/contextmenu.js", `strategy: "fixed"`, 2)
	requireSource(t, "../contextmenu/contextmenu.templ", `class="pointer-events-none isolate fixed inset-auto`)

	requireSource(t, "../select/select.js",
		`content.style.position = strategy`,
		`positionPopper(content, trigger, alignMode ? "fixed" : "absolute")`,
		`positionPopper(content, trigger, "absolute")`,
	)
	requireSource(t, "../select/select.templ", `class="pointer-events-none isolate fixed inset-auto`)

	originComponents := []string{
		"combobox", "dropdownmenu", "popover", "hovercard", "tooltip", "contextmenu", "select",
	}
	for _, component := range originComponents {
		requireSource(t, "../"+component+"/"+component+".js",
			"anchorRect.left + anchorRect.width / 2 - positionerRect.left",
			"anchorRect.top + anchorRect.height / 2 - positionerRect.top",
		)
	}
}
