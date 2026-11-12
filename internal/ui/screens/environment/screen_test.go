package environment

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

func strp(s string) *string { return &s }

func collection() *model.Collection {
	return &model.Collection{
		UID:               "c1",
		Name:              "Demo",
		ActiveEnvironment: "dev",
		Environments: []*model.Environment{
			{UID: "e1", Name: "dev", Variables: []model.Variable{
				{Name: strp("baseUrl"), Value: "https://api.example.com", Enabled: true},
				{Name: strp("token"), Value: "secret-value-123", Enabled: true, Secret: true},
			}},
			{UID: "e2", Name: "prod"},
		},
	}
}

func presenter() *presenters.EnvironmentPresenter {
	c := collection()
	p := presenters.NewEnvironmentPresenter(nil, nil, nil, nil)
	p.SetCollection(c)
	p.SetEditingEnvironment("dev")
	return p
}

// TestScreenShowsEnvironmentsAndChrome: the labels the editor draws — the
// environment list, the column headings and the trailing add row — are real
// text nodes. Variable names and values are input contents, which the tester
// does not report as labels, so they are asserted through the presenter below.
func TestScreenShowsEnvironmentsAndChrome(t *testing.T) {
	p := presenter()
	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, p, &State{})
	}, 900, 600)
	tt.Frame()

	for _, want := range []string{
		"Environments", "dev", "prod", "active",
		"NAME", "VALUE", "SECRET", "Use this environment",
	} {
		if !tt.HasText(want) {
			t.Errorf("expected %q on screen; texts: %v", want, tt.Texts())
		}
	}
}

// TestEditingEnvironmentDrivesTheTable: the table shows the environment being
// edited, so switching environments changes what is offered for editing.
func TestEditingEnvironmentDrivesTheTable(t *testing.T) {
	p := presenter()
	if got := p.EffectiveEnvironment(); got == nil || got.Name != "dev" {
		t.Fatalf("editing environment = %v, want dev", got)
	}

	p.SetEditingEnvironment("prod")
	got := p.EffectiveEnvironment()
	if got == nil || got.Name != "prod" {
		t.Fatalf("editing environment = %v, want prod", got)
	}
	if n := len(got.Variables); n != 0 {
		t.Errorf("prod has %d variables, want 0", n)
	}
}

// TestSecretValueIsMasked: a secret's value must never be rendered in clear.
func TestSecretValueIsMasked(t *testing.T) {
	p := presenter()
	env := p.EffectiveEnvironment()
	if env == nil {
		t.Fatal("no environment")
	}

	for _, v := range env.Variables {
		shown := p.FormattedVariableValue(v)
		if v.Secret {
			if shown == "secret-value-123" {
				t.Error("secret value leaked in clear")
			}
			for _, r := range shown {
				if r != '*' {
					t.Errorf("masked secret has non-mask rune %q", r)
				}
			}
		}
	}
}

// TestEmptyState: with no collection bound the screen explains itself instead
// of rendering an empty box.
func TestEmptyState(t *testing.T) {
	p := presenters.NewEnvironmentPresenter(nil, nil, nil, nil)
	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, p, &State{})
	}, 900, 600)
	tt.Frame()

	if !tt.HasText("No collection selected") {
		t.Errorf("expected an empty state; texts: %v", tt.Texts())
	}
}

// TestNextEnvironmentName: new environments get a free default name.
func TestNextEnvironmentName(t *testing.T) {
	envs := []*model.Environment{{Name: "Untitled 1"}, {Name: "Untitled 2"}}
	if got := nextEnvironmentName(envs); got != "Untitled 3" {
		t.Errorf("nextEnvironmentName = %q, want Untitled 3", got)
	}
	if got := nextEnvironmentName(nil); got != "Untitled 1" {
		t.Errorf("nextEnvironmentName(nil) = %q, want Untitled 1", got)
	}
}
