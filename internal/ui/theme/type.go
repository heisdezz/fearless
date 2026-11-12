package theme

import (
	"strings"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/theme"
)

// The Graphite Console type world, from the direction contract: the UI face at
// 12/13 px, small-caps panel headers at 10 px with 0.08em tracking, and every
// number, size, timing, status code, header name, URL and code value in
// monospace with tabular figures.
//
// The sizes live here rather than in the screens because D-017 is explicit
// that no screen restyles itself: a header is one call to PanelHeader and a
// figure is one call to DataText, so the world cannot drift one screen at a
// time.
const (
	// HeaderSize is a panel header's size in DIPs. Ten against a 13 px body
	// is the brief's step down; below it the caps stop being readable.
	HeaderSize float32 = 10

	// HeaderTracking is the letter spacing of a panel header in DIPs. MyGo
	// measures tracking in DIPs rather than em, so 0.8 DIP at HeaderSize is
	// the contract's 0.08em.
	HeaderTracking float32 = 0.8

	// DataFont is the family every figure is set in. It is the generic
	// family name, so the platform picks its own monospace face rather than
	// the app naming a font that may not ship.
	DataFont = "monospace"

	// DataFeature is the OpenType feature that makes digits one width, so a
	// column of timings and sizes aligns on the decimal.
	DataFeature = "tnum"

	// DataSize is the size data text renders at: the body size of the data
	// column, one step under the 13 px UI face.
	DataSize float32 = 12

	// TitleSize is the collection name in the title strip — the largest text
	// in the chrome, still within the UI face's 12/13 range.
	TitleSize float32 = 13
)

// PanelHeader renders a section header in the world's small caps: the label
// uppercased, at HeaderSize with HeaderTracking, in the secondary ink so the
// header labels the values under it rather than competing with them.
//
// Titles are uppercased here instead of being written in caps by every caller,
// so a header's source text stays normal prose — "Query Parameters", not
// "QUERY PARAMETERS" — and a later change of case style is one edit.
func PanelHeader(c *ui.Context, title string) ui.Element {
	tk := lobe.TokensOf(c)
	return Header(c, title, tk.TextSecondary)
}

// Header renders a section header in the small-caps face, in the given ink.
//
// It is the same face as PanelHeader with the caller's colour, for the places
// where a header's ink carries meaning — a selected section tab is in the
// primary ink, an unselected one in the secondary. The size and the tracking
// still come from the shared world, so a tab and a panel header cannot drift
// apart.
func Header(c *ui.Context, title string, color ui.Color) ui.Element {
	return ui.Text(c, strings.ToUpper(title)).
		FontSize(HeaderSize).
		LetterSpacing(HeaderTracking).
		TextColor(color)
}

// DataText renders one run of data — a number, status code, header name, URL,
// size or timing — in the data face: the monospace family with tabular
// figures. Colour and size are left to the caller, because the same face
// carries a success code and a muted byte count.
//
// It is the shared door to monospace in this app. A screen that sets
// ui.Text(...).Font("monospace") by hand is restyling the type world itself.
func DataText(c *ui.Context, s string) ui.Element {
	return ui.Text(c, s).Font(DataFont).FontFeatures(DataFeature)
}

// DataGroup styles a subtree as data: every figure inside it inherits the
// monospace family and tabular figures, which is how an input (the URL bar)
// and a row of separate labels get the data face without each child asking
// for it.
func DataGroup(c *ui.Context, build func()) ui.Element {
	return ui.Box(c).Font(DataFont).FontFeatures(DataFeature).Children(build)
}
