package widgets

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// CodeEditorProps configures the CodeEditor widget.
type CodeEditorProps struct {
	Language    string // e.g. "json", "xml", "javascript", "go", "text"
	ReadOnly    bool
	Variables   []string // Variables available for {{var}} completion
	Placeholder string
	OnChanged   func(newContent string)
}

// CodeEditor wraps MyGo's native code editor with Fearless {{var}} autocomplete.
func CodeEditor(c *ui.Context, state **editor.State, initialContent string, props CodeEditorProps) ui.Element {
	if state == nil {
		return ui.Box(c)
	}

	if *state == nil {
		buf := editor.NewBuffer(initialContent)
		langName := props.Language
		if langName == "" {
			langName = "json"
		}
		if lang := editor.LanguageNamed(langName); lang != nil {
			buf.SetLanguage(lang)
		}

		st := &editor.State{
			Buffer:      buf,
			ReadOnly:    props.ReadOnly,
			Placeholder: props.Placeholder,
		}

		// Configure completion handler for {{var}}
		if len(props.Variables) > 0 {
			vars := props.Variables
			st.OnCompletion = func(req editor.CompletionRequest) {
				var items []editor.Completion
				for _, v := range vars {
					items = append(items, editor.Completion{
						Label:  "{{" + v + "}}",
						Detail: "variable",
						Insert: v + "}}",
					})
				}
				st.ShowCompletions(st.Cursor(), items)
			}
		}

		*state = st
	}

	return editor.View(c, *state).Fill()
}
