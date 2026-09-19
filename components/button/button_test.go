package button_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/axadrn/shadcn-templ/v2/components/button"
	"github.com/axadrn/shadcn-templ/v2/components/dropdownmenu"
)

func TestAttributesIDTakesPrecedence(t *testing.T) {
	for _, href := range []string{"", "/example"} {
		t.Run("href="+href, func(t *testing.T) {
			var output bytes.Buffer
			err := button.Button(button.Props{
				ID: "mine", Href: href, Attributes: templ.Attributes{"id": "theirs"},
			}).Render(context.Background(), &output)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(output.String(), ` id="`); got != 1 || !strings.Contains(output.String(), ` id="theirs"`) {
				t.Fatalf("expected one id from Attributes: %s", output.String())
			}
		})
	}
}

func TestDropdownTriggerIDLabelsPopup(t *testing.T) {
	children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := button.Button(button.Props{ID: "x", Attributes: dropdownmenu.Trigger(ctx)}).Render(ctx, w); err != nil {
			return err
		}
		return dropdownmenu.Content().Render(ctx, w)
	})
	var output bytes.Buffer
	if err := dropdownmenu.DropdownMenu(dropdownmenu.Props{ID: "menu"}).Render(templ.WithChildren(context.Background(), children), &output); err != nil {
		t.Fatal(err)
	}
	out := output.String()
	_, trigger, found := strings.Cut(out, "<button")
	if !found {
		t.Fatalf("missing dropdown trigger: %s", out)
	}
	trigger, _, found = strings.Cut(trigger, ">")
	if !found || strings.Count(trigger, ` id="`) != 1 || !strings.Contains(trigger, ` id="menu-trigger"`) {
		t.Fatalf("expected one derived dropdown trigger id: %s", out)
	}
	if !strings.Contains(out, ` aria-labelledby="menu-trigger"`) {
		t.Fatalf("popup is not labelled by the dropdown trigger: %s", out)
	}
}
