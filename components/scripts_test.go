package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestScripts(t *testing.T) {
	var output bytes.Buffer
	if err := Scripts().Render(templ.WithNonce(context.Background(), "test-nonce"), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`src="` + bundleSrc + `"`, `nonce="test-nonce"`, `<script defer`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output %q missing %q", output.String(), want)
		}
	}
}
