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
	"golang.org/x/net/html"
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
	z := html.NewTokenizer(&output)
	var triggerIDs []string
	var label string
	for {
		tokenType := z.Next()
		if tokenType == html.ErrorToken {
			if z.Err() != io.EOF {
				t.Fatal(z.Err())
			}
			break
		}
		if tokenType != html.StartTagToken {
			continue
		}
		token := z.Token()
		for _, attr := range token.Attr {
			if token.Data == "button" && attr.Key == "id" {
				triggerIDs = append(triggerIDs, attr.Val)
			}
			if attr.Key == "aria-labelledby" {
				label = attr.Val
			}
		}
	}
	if len(triggerIDs) != 1 || triggerIDs[0] != "menu-trigger" || label != triggerIDs[0] {
		t.Fatalf("trigger ids %v do not uniquely label popup %q", triggerIDs, label)
	}
}
