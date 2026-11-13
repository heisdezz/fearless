package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func tokenView(f *TemplateField, url *string, opened *[]string) func(c *ui.Context) {
	return func(c *ui.Context) {
		TemplateFieldView(c, f, url, TemplateFieldProps{
			OnVariableClick: func(name string) { *opened = append(*opened, name) },
		})
	}
}

// TestTemplateField_ClickOnTokenOpensVariable: a click on a `{{variable}}`
// opens that variable, so it can be created or edited, and does not leave the
// field in its edit state.
func TestTemplateField_ClickOnTokenOpensVariable(t *testing.T) {
	f := &TemplateField{}
	url := "{{baseUrl}}/v1"
	var opened []string

	tt := ui.NewTester(tokenView(f, &url, &opened), 400, 60)
	tt.Frame()
	token, ok := tt.Find("{{baseUrl}}")
	if !ok {
		t.Fatalf("token not drawn; texts: %v", tt.Texts())
	}
	tt.ClickAt(token.X+4, token.Y+token.H/2)

	if len(opened) != 1 || opened[0] != "baseUrl" {
		t.Fatalf("click opened %v, want [baseUrl]", opened)
	}
	if f.Editing {
		t.Error("opening a variable must not start an edit")
	}
}

// TestTemplateField_DoubleClickOnTokenOpensVariableOnce: two clicks open the
// variable once, not twice, so the editor is not opened again on the second.
func TestTemplateField_DoubleClickOnTokenOpensVariableOnce(t *testing.T) {
	f := &TemplateField{}
	url := "{{baseUrl}}/v1"
	var opened []string

	tt := ui.NewTester(tokenView(f, &url, &opened), 400, 60)
	tt.Frame()
	token, ok := tt.Find("{{baseUrl}}")
	if !ok {
		t.Fatalf("token not drawn; texts: %v", tt.Texts())
	}
	x, y := token.X+4, token.Y+token.H/2
	tt.ClickAt(x, y)
	tt.Frame()
	tt.ClickAt(x, y)

	if len(opened) < 1 || opened[0] != "baseUrl" {
		t.Fatalf("clicks opened %v, want baseUrl", opened)
	}
	if f.Editing {
		t.Error("opening a variable must not start an edit")
	}
}

// TestTemplateField_ClickOutsideTokensEdits: a click on the text between tokens
// still starts editing the field.
func TestTemplateField_ClickOutsideTokensEdits(t *testing.T) {
	f := &TemplateField{}
	url := "{{baseUrl}}/v1"
	var opened []string

	tt := ui.NewTester(tokenView(f, &url, &opened), 400, 60)
	tt.Frame()
	path, ok := tt.Find("/v1")
	if !ok {
		t.Fatalf("path text not drawn; texts: %v", tt.Texts())
	}
	tt.ClickAt(path.X+path.W/2, path.Y+path.H/2)

	if len(opened) != 0 {
		t.Fatalf("a click on the path opened %v", opened)
	}
	if !f.Editing {
		t.Error("a click on the path should start editing the field")
	}
}
