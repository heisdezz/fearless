package theme

import (
	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/theme"
)

// Graphite Console is the application's visual world (D-017). It is a
// dark-first working surface: a graphite ground, panels told apart by one
// hairline seam rather than by shadow, one accent colour earned by the primary
// action and the selected state, and state colours that carry meaning and
// nothing else.
//
// The world is expressed as a complete set of Lobe tokens, because every
// component in the app paints from them: restyling the tokens restyles the
// whole interface, and no screen restyles itself.
//
// Palette:
//
//	ground   #141518   the window and the request column
//	panel    #1A1C20   sidebar, response pane, dialogs — one step up
//	raised   #212429   hover faces, selected rows, headers
//	seam     #24262B   every divider; the only elevation the world allows
//	ink      #E6E8EC   primary text          (13.9:1 on ground)
//	mute     #9BA0AA   secondary text        ( 5.9:1 on ground)
//	dim      #6B707A   tertiary, inoperative ( 3.4:1 on ground)
//	accent   #4D8DFF   primary action, selection, focus
//
// Only three hues are semantic: success, warning and danger. HTTP verbs keep
// Bruno's own table (methodColors below) because users read verbs by colour
// across tools.
//
// Light is a finished companion, not a peer (D-018): the same structure on a
// paper ground, resolved so every control stays legible.
var (
	graphite = [4]ui.Color{
		ui.Hex("#141518"), // ground
		ui.Hex("#1A1C20"), // panel
		ui.Hex("#212429"), // raised
		ui.Hex("#24262B"), // seam
	}
	graphiteInk  = [3]ui.Color{ui.Hex("#E6E8EC"), ui.Hex("#9BA0AA"), ui.Hex("#6B707A")}
	graphiteBlue = ui.Hex("#4D8DFF")

	paper = [4]ui.Color{
		ui.Hex("#FBFBFC"), // ground
		ui.Hex("#F4F5F7"), // panel
		ui.Hex("#ECEDF1"), // raised
		ui.Hex("#DCDFE5"), // seam
	}
	paperInk  = [3]ui.Color{ui.Hex("#1B1D21"), ui.Hex("#565C66"), ui.Hex("#7C838F")}
	paperBlue = ui.Hex("#2563EB")
)

// darkAccent is the accent for a single appearance: the world's blue, with a
// hover that lightens and a press that darkens, so the pressed state is
// readable in a dark room without a second hue.
var (
	darkAccentSet  = [3]ui.Color{graphiteBlue, ui.Hex("#6FA3FF"), ui.Hex("#3C79E6")}
	lightAccentSet = [3]ui.Color{paperBlue, ui.Hex("#1D4ED8"), ui.Hex("#1E40AF")}
	verbDark       = [3]ui.Color{ui.Hex("#F85149"), ui.Hex("#D29922"), ui.Hex("#3FB950")}
	verbLight      = [3]ui.Color{ui.Hex("#C62828"), ui.Hex("#9A6700"), ui.Hex("#1A7F37")}
)

// tokens returns the Graphite world for one appearance. It starts from a copy
// of a Lobe token set, not Lobe's pointer: NewConfig stores the theme value it
// derives, so a caller that later writes into a copy cannot change what a live
// frame paints.
func tokens(dark bool) lobe.Tokens {
	tk := *lobe.Light()
	if dark {
		tk = *lobe.Dark()
	}
	applyWorld(&tk, dark)
	return tk
}

// applyWorld writes the Graphite world into tk. Every Lobe token a component
// paints with is set here; tokens Lobe derives (accent fills, status
// backgrounds) are recomputed from the world's colours so nothing falls back to
// Lobe's own palette mid-frame.
func applyWorld(tk *lobe.Tokens, dark bool) {
	p := graphite
	ink := graphiteInk
	acc := darkAccentSet
	verb := verbDark
	if !dark {
		p = paper
		ink = paperInk
		acc = lightAccentSet
		verb = verbLight
	}

	tk.Dark = dark

	// Surfaces. The ground is the window; panel is one step up and carries
	// every fixed region; raised is a transient face (hover, selection,
	// header). Nothing here uses a shadow.
	tk.BgLayout = p[0]
	tk.BgBase = p[1]
	tk.BgContainer = p[1]
	tk.BgElevated = p[2]
	tk.BgContainerSecondary = p[2]
	tk.BgContainerDisabled = overlay(ink[1], 0.05)
	tk.BgMask = ui.RGBA(0, 0, 0, 0.5)
	if !dark {
		tk.BgMask = ui.RGBA(27, 29, 33, 0.4)
	}
	tk.BgSpotlight = p[2]
	tk.BgBlur = overlay(p[2], 0.9)
	tk.BgTextHover = overlay(ink[0], 0.06)
	tk.BgTextActive = overlay(ink[0], 0.12)
	tk.BgSolid = ink[0]
	tk.BgSolidHover = mixHex(ink[0], p[0], 0.08)
	tk.BgSolidActive = mixHex(ink[0], p[0], 0.16)

	// Seams. One colour separates everything; Border is the seam that also
	// outlines controls, so it is a touch stronger than the flat divider.
	tk.Border = p[3]
	tk.BorderBg = p[0]
	tk.BorderSecondary = mixHex(p[3], p[1], 0.35)
	tk.BorderDisabled = mixHex(p[3], p[1], 0.6)
	tk.Split = p[3]
	tk.Fill = p[2]
	tk.FillSecondary = mixHex(p[2], p[1], 0.35)
	tk.FillTertiary = mixHex(p[2], p[1], 0.6)
	tk.FillQuaternary = mixHex(p[2], p[1], 0.8)
	tk.FillAlter = p[2]
	tk.FillContent = p[2]
	tk.FillContentHover = p[3]

	// Text, in three levels and nothing between them.
	tk.Text = ink[0]
	tk.TextHeading = ink[0]
	tk.TextLabel = ink[1]
	tk.TextDescription = ink[1]
	tk.TextSecondary = ink[1]
	tk.TextTertiary = ink[2]
	tk.TextQuaternary = mixHex(ink[2], p[1], 0.35)
	tk.TextPlaceholder = ink[2]
	tk.TextDisabled = mixHex(ink[1], p[1], 0.45)
	tk.Icon = ink[1]
	tk.IconHover = ink[0]
	tk.Link = acc[0]
	tk.LinkHover = acc[1]
	tk.LinkActive = acc[2]
	tk.Highlight = acc[0]

	// The one accent: earned by the primary action and the selected state.
	// Its fills are the accent mixed into the panel, never a second hue.
	tk.Primary = acc[0]
	tk.PrimaryHover = acc[1]
	tk.PrimaryActive = acc[2]
	tk.PrimaryBorder = acc[1]
	tk.PrimaryBg = mixHex(p[1], acc[0], 0.14)
	tk.PrimaryBgHover = mixHex(p[1], acc[0], 0.24)
	tk.PrimaryFill = overlay(acc[0], 0.22)
	tk.PrimaryFillSecondary = overlay(acc[0], 0.14)
	tk.PrimaryFillTertiary = overlay(acc[0], 0.09)
	tk.PrimaryFillQuaternary = overlay(acc[0], 0.05)
	tk.PrimaryBorderHover = acc[0]
	tk.PrimaryText = acc[0]
	tk.PrimaryTextHover = acc[1]
	tk.PrimaryTextActive = mixHex(p[0], acc[0], 0.85)
	tk.PrimaryForeground = onColor(acc[0])

	// State, semantic and nothing else.
	applyStatus(tk, verb[2], "Success", p, dark)
	applyStatus(tk, verb[1], "Warning", p, dark)
	applyStatus(tk, verb[0], "Error", p, dark)

	// Info is the accent, restated: an informational message is not a second
	// hue in a world with one accent.
	tk.Info = acc[0]
	tk.InfoHover = acc[1]
	tk.InfoActive = acc[2]
	tk.InfoBg = mixHex(p[1], acc[0], 0.14)
	tk.InfoBgHover = mixHex(p[1], acc[0], 0.22)
	tk.InfoBorder = mixHex(p[1], acc[0], 0.36)
	tk.InfoBorderHover = mixHex(p[1], acc[0], 0.5)
	tk.InfoText = acc[0]
	tk.InfoTextHover = acc[1]
	tk.InfoTextActive = acc[0]
	tk.InfoFill = overlay(acc[0], 0.22)
	tk.InfoFillSecondary = overlay(acc[0], 0.14)
	tk.InfoFillTertiary = overlay(acc[0], 0.09)
	tk.InfoFillQuaternary = overlay(acc[0], 0.05)

	// Syntax accents used by highlights and tags: derived from the verb table
	// so one palette governs the whole surface.
	tk.Blue = acc[0]
	tk.BlueActive = acc[2]
	tk.BlueFillTertiary = overlay(acc[0], 0.12)
	tk.BlueFillQuaternary = overlay(acc[0], 0.06)
	tk.Purple = mixHex(acc[0], verb[0], 0.45)
	tk.PurpleFillTertiary = overlay(tk.Purple, 0.12)
	tk.PurpleFillQuaternary = overlay(tk.Purple, 0.06)
	tk.Cyan = mixHex(acc[0], verb[2], 0.5)
	tk.CyanFillTertiary = overlay(tk.Cyan, 0.12)
	tk.CyanFillQuaternary = overlay(tk.Cyan, 0.06)
	tk.Green = verb[2]
	tk.GreenFillTertiary = overlay(verb[2], 0.12)
	tk.GreenFillQuaternary = overlay(verb[2], 0.06)
	tk.Orange = verb[1]
	tk.OrangeFillTertiary = overlay(verb[1], 0.12)
	tk.OrangeFillQuaternary = overlay(verb[1], 0.06)
	tk.Red = verb[0]
	tk.RedFillTertiary = overlay(verb[0], 0.12)
	tk.RedFillQuaternary = overlay(verb[0], 0.06)

	// Scale. A desktop tool keeps a compact rhythm; the base size stays
	// Lobe's 13 px so text does not jump when a theme switches. Radii are
	// small: seams and square panels carry the structure, controls are only
	// softened at the corners.
	tk.FontSize = 13
	tk.FontSizeSM = 12
	tk.FontSizeLG = 15
	tk.FontSizeXL = 18
	tk.BorderRadius = 4
	tk.BorderRadiusSM = 3
	tk.BorderRadiusLG = 6
	tk.BorderRadiusXS = 2
	if dark {
		tk.FontWeightStrong = 600
	}
}

// applyStatus writes one semantic status group. The fill is the hue mixed into
// the panel (never a translucent wash over an unknown ground), the text is the
// hue lifted for legibility on it, and the border is a seam of the same hue.
func applyStatus(tk *lobe.Tokens, hue ui.Color, name string, p [4]ui.Color, dark bool) {
	text := hue
	if dark {
		text = mixHex(hue, ui.Hex("#FFFFFF"), 0.36)
	} else {
		text = mixHex(hue, ui.Hex("#000000"), 0.28)
	}
	bg := mixHex(p[1], hue, 0.16)
	bgHover := mixHex(p[1], hue, 0.24)
	border := mixHex(p[1], hue, 0.36)
	borderHover := mixHex(p[1], hue, 0.5)
	fill := overlay(hue, 0.22)
	fill2 := overlay(hue, 0.14)
	fill3 := overlay(hue, 0.09)
	fill4 := overlay(hue, 0.05)

	switch name {
	case "Success":
		tk.Success, tk.SuccessHover, tk.SuccessActive = hue, mixHex(hue, ui.Hex("#FFFFFF"), 0.2), mixHex(hue, ui.Hex("#000000"), 0.15)
		tk.SuccessBg, tk.SuccessBgHover = bg, bgHover
		tk.SuccessBorder, tk.SuccessBorderHover = border, borderHover
		tk.SuccessText, tk.SuccessTextHover, tk.SuccessTextActive = text, text, text
		tk.SuccessFill, tk.SuccessFillSecondary = fill, fill2
		tk.SuccessFillTertiary, tk.SuccessFillQuaternary = fill3, fill4
	case "Warning":
		tk.Warning, tk.WarningHover, tk.WarningActive = hue, mixHex(hue, ui.Hex("#FFFFFF"), 0.2), mixHex(hue, ui.Hex("#000000"), 0.15)
		tk.WarningBg, tk.WarningBgHover = bg, bgHover
		tk.WarningBorder, tk.WarningBorderHover = border, borderHover
		tk.WarningText, tk.WarningTextHover, tk.WarningTextActive = text, text, text
		tk.WarningFill, tk.WarningFillSecondary = fill, fill2
		tk.WarningFillTertiary, tk.WarningFillQuaternary = fill3, fill4
	case "Error":
		tk.Error, tk.ErrorHover, tk.ErrorActive = hue, mixHex(hue, ui.Hex("#FFFFFF"), 0.2), mixHex(hue, ui.Hex("#000000"), 0.15)
		tk.ErrorAffix = hue
		tk.ErrorBg, tk.ErrorBgHover, tk.ErrorBgActive = bg, bgHover, mixHex(p[1], hue, 0.3)
		tk.ErrorBgFilledHover = mixHex(p[1], hue, 0.2)
		tk.ErrorBorder, tk.ErrorBorderHover = border, borderHover
		tk.ErrorText, tk.ErrorTextHover, tk.ErrorTextActive = text, text, text
		tk.ErrorFill, tk.ErrorFillSecondary = fill, fill2
		tk.ErrorFillTertiary, tk.ErrorFillQuaternary = fill3, fill4
		tk.ErrorOutline = fill4
	}
}

// worldTheme finishes the ui.Theme a Config builds from the world's tokens:
// tk.Theme() already derives Background, Surface, Border and Text, so this only
// adds the parts the world is stricter about than Lobe, chiefly the accent and
// its pressed state. It is called from buildConfig on each mode's config theme,
// so a component that repaints in a new appearance still resolves the world's
// tokens through Lobe's TokensOf instead of falling back to Lobe's defaults
// (the white-input defect D-017 records).
func worldTheme(dark bool) *ui.Theme {
	tk := tokens(dark)
	th := tk.Theme()
	p := graphite
	ink := graphiteInk
	acc := darkAccentSet
	if !dark {
		p = paper
		ink = paperInk
		acc = lightAccentSet
	}
	th.Background = p[0]
	th.Surface = p[2]
	th.SurfaceHover = mixHex(p[2], ink[0], 0.06)
	th.SurfacePressed = mixHex(p[2], ink[0], 0.12)
	th.Border = p[3]
	th.Text = ink[0]
	th.TextMuted = ink[1]
	th.Accent = acc[0]
	th.AccentHover = acc[1]
	th.AccentPressed = acc[2]
	th.AccentText = onColor(acc[0])
	th.InactiveAccent = mixHex(acc[0], p[1], 0.55)
	th.Danger = tk.Error
	th.Warning = tk.Warning
	th.Success = tk.Success
	th.Selection = overlay(acc[0], 0.30)
	th.Focus = acc[0]
	th.FocusRingWidth = 1
	th.FocusRingOffset = 1
	th.Inverse = ink[0]
	th.InverseText = p[0]
	th.Scrollbar = overlay(ink[1], 0.35)
	th.ScrollbarWidth = 8
	th.Radius = 4
	th.Spacing = 4
	th.FontSize = 13
	return th
}

// overlay is c at the given alpha, for tints meant to sit on any surface.
func overlay(c ui.Color, a float32) ui.Color {
	c.A = uint8(a*255 + 0.5)
	return c
}

// mixHex blends t toward o; the alpha comes from the opaque source so a mix of
// two solid colours stays solid.
func mixHex(c, o ui.Color, t float32) ui.Color {
	out := lobe.Mix(c, o, t)
	out.A = 255
	return out
}

// onColor is the ink that reads on fill: near-white on a dark, saturated fill and
// near-black on a light one, chosen by relative luminance, never by habit.
func onColor(fill ui.Color) ui.Color {
	l := 0.2126*float32(fill.R)/255 + 0.7152*float32(fill.G)/255 + 0.0722*float32(fill.B)/255
	if l > 0.55 {
		return ui.Hex("#141518")
	}
	return ui.Hex("#FFFFFF")
}
