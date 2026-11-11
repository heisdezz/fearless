package theme

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type BrunoTheme struct {
	Name       string                  `json:"name"`
	Mode       string                  `json:"mode"`
	Brand      NRGBA                   `json:"brand"`
	Primary    PrimaryColors           `json:"primary"`
	Background BackgroundColors        `json:"background"`
	Status     map[string]StatusColors `json:"status"`
	Font       FontSizes               `json:"font"`
	Radius     BorderRadius            `json:"radius"`
	Border     BorderColors            `json:"border"`
	Methods    map[string]NRGBA        `json:"methods"`
	Colors     ColorExtras             `json:"colors"`
	Input      InputColors             `json:"input"`
	Request    RequestColors           `json:"request"`
	WS         WSColors                `json:"ws"`
	CodeMirror CodeMirrorColors        `json:"codemirror"`
}

type NRGBA struct {
	R, G, B, A uint8  `json:"-"`
	Hex        string `json:"hex"`
}

func (n NRGBA) Color() color.NRGBA {
	if n.Hex != "" {
		return parseHex(n.Hex)
	}
	return color.NRGBA{R: n.R, G: n.G, B: n.B, A: n.A}
}

func parseHex(s string) color.NRGBA {
	var c color.NRGBA
	if len(s) == 7 && s[0] == '#' {
		fmt.Sscanf(s[1:], "%02x%02x%02x", &c.R, &c.G, &c.B)
		c.A = 255
	} else if len(s) == 9 && s[0] == '#' {
		fmt.Sscanf(s[1:], "%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A)
	}
	return c
}

type PrimaryColors struct {
	Solid  NRGBA `json:"solid"`
	Text   NRGBA `json:"text"`
	Strong NRGBA `json:"strong"`
	Subtle NRGBA `json:"subtle"`
}

type BackgroundColors struct {
	Base     NRGBA `json:"base"`
	Mantle   NRGBA `json:"mantle"`
	Crust    NRGBA `json:"crust"`
	Surface0 NRGBA `json:"surface0"`
	Surface1 NRGBA `json:"surface1"`
	Surface2 NRGBA `json:"surface2"`
}

type StatusColors struct {
	Background NRGBA `json:"background"`
	Text       NRGBA `json:"text"`
	Border     NRGBA `json:"border"`
}

type FontSizes struct {
	XS   int `json:"xs"`
	SM   int `json:"sm"`
	Base int `json:"base"`
	MD   int `json:"md"`
	LG   int `json:"lg"`
	XL   int `json:"xl"`
}

type BorderRadius struct {
	SM   int `json:"sm"`
	Base int `json:"base"`
	MD   int `json:"md"`
	LG   int `json:"lg"`
	XL   int `json:"xl"`
}

type BorderColors struct {
	Border0 NRGBA `json:"border0"`
	Border1 NRGBA `json:"border1"`
	Border2 NRGBA `json:"border2"`
	Border3 NRGBA `json:"border3"`
	Subtle  NRGBA `json:"subtle"`
}

type ColorExtras struct {
	Text struct {
		White    NRGBA `json:"white"`
		Green    NRGBA `json:"green"`
		Danger   NRGBA `json:"danger"`
		Warning  NRGBA `json:"warning"`
		Muted    NRGBA `json:"muted"`
		Purple   NRGBA `json:"purple"`
		Yellow   NRGBA `json:"yellow"`
		Subtext0 NRGBA `json:"subtext0"`
		Subtext1 NRGBA `json:"subtext1"`
		Subtext2 NRGBA `json:"subtext2"`
	} `json:"text"`
	BG struct {
		Danger NRGBA `json:"danger"`
	} `json:"bg"`
	Accent NRGBA `json:"accent"`
}

type InputColors struct {
	BG          NRGBA `json:"bg"`
	Border      NRGBA `json:"border"`
	FocusBorder NRGBA `json:"focusBorder"`
	Placeholder NRGBA `json:"placeholder"`
}

type RequestColors struct {
	Methods map[string]NRGBA `json:"methods"`
}

type WSColors struct {
	ActiveMessage struct {
		Label NRGBA `json:"label"`
	} `json:"activeMessage"`
}

type CodeMirrorColors struct {
	Gutter struct {
		BG NRGBA `json:"bg"`
	} `json:"gutter"`
	Border NRGBA `json:"border"`
}

// isDark reports which appearance the snapshot describes. The match is
// case- and space-insensitive: a theme file is hand-edited JSON, and
// "Dark"/"DARK"/" dark " must not silently fall through to the light table
// (review finding on UI-THEME, 2026-11-11T02:18Z). Anything that is not
// dark — including an empty mode — is a light-theme import.
func (b *BrunoTheme) isDark() bool {
	return b != nil && strings.EqualFold(strings.TrimSpace(b.Mode), "dark")
}

// hslColor is one colour of Bruno's palette: hue in degrees, saturation and
// lightness in percent.
type hslColor struct{ h, s, l float64 }

// Bruno's request.methods tables, transcribed from the shipped themes:
//
//	light: bruno/packages/bruno-app/src/themes/light/light.js:262
//	dark:  bruno/packages/bruno-app/src/themes/dark/dark.js:270
//
// each hue taken from that file's palette (light.js:2, dark.js:3). The dark
// table's delete is `lighten(0.08, hues.RED)`, which adds 0.08 to HSL
// lightness, so hsl(8, 70%, 52%) → hsl(8, 70%, 60%). Note what the source
// actually says: post is PURPLE in light and INDIGO in dark, patch is PURPLE
// in light and ORANGE in dark — the tables are not the same mapping twice.
var (
	brunoLightMethods = map[string]hslColor{
		"get":     {145, 50, 36}, // hues.GREEN
		"post":    {280, 45, 48}, // hues.PURPLE
		"put":     {35, 85, 42},  // hues.ORANGE
		"delete":  {8, 60, 52},   // hues.RED
		"patch":   {280, 45, 48}, // hues.PURPLE (shares post's colour)
		"options": {178, 50, 36}, // hues.TEAL
		"head":    {195, 55, 42}, // hues.CYAN
	}
	brunoDarkMethods = map[string]hslColor{
		"get":     {140, 72, 68}, // hues.GREEN
		"post":    {202, 88, 72}, // hues.INDIGO
		"put":     {24, 88, 72},  // hues.ORANGE
		"delete":  {8, 70, 60},   // lighten(0.08, hues.RED)
		"patch":   {24, 88, 72},  // hues.ORANGE (shares put's colour)
		"options": {170, 70, 60}, // hues.TEAL
		"head":    {190, 82, 72}, // hues.CYAN
	}
	// colors.text.muted of each theme, the colour MethodBadge paints a method
	// with no entry (MethodBadge/StyledWrapper.js:4): light SUBTEXT1 #838383,
	// dark SUBTEXT1 #aaa.
	brunoMuted = [2]ui.Color{ui.Hex("#838383"), ui.Hex("#aaaaaa")}
)

// modeIndex maps an appearance to a table slot: 0 light, 1 dark.
func modeIndex(dark bool) int {
	if dark {
		return 1
	}
	return 0
}

// defaultMethodColors returns Bruno's method table for one mode.
func defaultMethodColors(dark bool) map[string]ui.Color {
	src := brunoLightMethods
	if dark {
		src = brunoDarkMethods
	}
	out := make(map[string]ui.Color, len(src))
	for m, c := range src {
		out[m] = hsl(c.h, c.s, c.l)
	}
	return out
}

// hsl converts hue in degrees and saturation/lightness in percent to a
// colour, rounding each channel half-up the way JavaScript's Math.round
// does. Bruno's palette is HSL and the browser rounds the same way, so the
// resulting hexes are the ones Bruno actually renders.
func hsl(h, s, l float64) ui.Color {
	s /= 100
	l /= 100
	c := (1 - math.Abs(2*l-1)) * s
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return ui.RGBA(roundChannel(r+m), roundChannel(g+m), roundChannel(b+m), 1)
}

func roundChannel(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(math.Floor(v*255 + 0.5))
}

type Adapter struct {
	mu    sync.RWMutex
	bruno *BrunoTheme
	// brunoGen counts SetBrunoTheme calls; cfgGen is the generation cfg was
	// built for, so a theme change rebuilds the token config on the next Apply.
	brunoGen int
	cfgGen   int
	cfg      *theme.Config
	// methods holds one method→colour table per appearance, and muted the
	// fallback for a method with no entry (Bruno's colors.text.muted). Both
	// start as Bruno's built-in tables and a loaded theme overrides the mode
	// it declares.
	methods [2]map[string]ui.Color
	muted   [2]ui.Color
}

func NewAdapter() *Adapter {
	return &Adapter{
		methods: [2]map[string]ui.Color{
			defaultMethodColors(false),
			defaultMethodColors(true),
		},
		muted: brunoMuted,
	}
}

// Apply makes the frame use this adapter's tokens in the frame's own
// appearance — the light or dark set, following the system — and is what the
// rest of the UI must call instead of Lobe's theme.Apply, so a Bruno theme
// import reaches every component through one call point.
func (a *Adapter) Apply(c *ui.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg == nil || a.cfgGen != a.brunoGen {
		a.buildConfig()
		a.cfgGen = a.brunoGen
	}

	a.cfg.Apply(c)
}

// buildConfig rebuilds the light/dark token pair. A loaded Bruno theme
// describes one appearance, so its snapshot is mapped into that mode's tokens
// only; the other mode keeps Lobe's defaults. Mapping the snapshot into both
// would paint a light theme's background into dark mode and break "light/dark
// follows system".
func (a *Adapter) buildConfig() {
	lightTokens := *theme.Light()
	darkTokens := *theme.Dark()

	if a.bruno != nil {
		if a.bruno.isDark() {
			a.mapBrunoToTokens(&darkTokens)
		} else {
			a.mapBrunoToTokens(&lightTokens)
		}
	}

	a.cfg = theme.NewConfig(lightTokens, darkTokens)
}

func (a *Adapter) mapBrunoToTokens(tk *theme.Tokens) {
	if a.bruno == nil {
		return
	}

	if a.bruno.Brand.Hex != "" {
		brand := toUIColor(a.bruno.Brand.Color())
		tk.Primary = brand
		tk.PrimaryHover = brand
		tk.PrimaryActive = brand
		tk.PrimaryBorder = brand
		tk.PrimaryForeground = brand
	}

	if a.bruno.Primary.Solid.Hex != "" {
		tk.Primary = toUIColor(a.bruno.Primary.Solid.Color())
		tk.PrimaryHover = toUIColor(a.bruno.Primary.Strong.Color())
		tk.PrimaryActive = toUIColor(a.bruno.Primary.Strong.Color())
		tk.PrimaryText = toUIColor(a.bruno.Primary.Text.Color())
		tk.PrimaryForeground = toUIColor(a.bruno.Primary.Text.Color())
	}

	if a.bruno.Background.Base.Hex != "" {
		tk.BgBase = toUIColor(a.bruno.Background.Base.Color())
		tk.BgLayout = toUIColor(a.bruno.Background.Mantle.Color())
		tk.BgContainer = toUIColor(a.bruno.Background.Surface0.Color())
		tk.BgElevated = toUIColor(a.bruno.Background.Surface1.Color())
		tk.BgContainerSecondary = toUIColor(a.bruno.Background.Surface2.Color())
	}

	if a.bruno.Status != nil {
		if v, ok := a.bruno.Status["success"]; ok {
			tk.SuccessBg = toUIColor(v.Background.Color())
			tk.SuccessText = toUIColor(v.Text.Color())
			tk.SuccessBorder = toUIColor(v.Border.Color())
		}
		if v, ok := a.bruno.Status["warning"]; ok {
			tk.WarningBg = toUIColor(v.Background.Color())
			tk.WarningText = toUIColor(v.Text.Color())
			tk.WarningBorder = toUIColor(v.Border.Color())
		}
		if v, ok := a.bruno.Status["danger"]; ok {
			tk.ErrorBg = toUIColor(v.Background.Color())
			tk.ErrorText = toUIColor(v.Text.Color())
			tk.ErrorBorder = toUIColor(v.Border.Color())
		}
		if v, ok := a.bruno.Status["info"]; ok {
			tk.InfoBg = toUIColor(v.Background.Color())
			tk.InfoText = toUIColor(v.Text.Color())
			tk.InfoBorder = toUIColor(v.Border.Color())
		}
	}

	if a.bruno.Font.Base != 0 {
		tk.FontSize = float32(a.bruno.Font.Base)
		tk.FontSizeSM = float32(a.bruno.Font.SM)
		tk.FontSizeLG = float32(a.bruno.Font.LG)
		tk.FontSizeXL = float32(a.bruno.Font.XL)
	}

	if a.bruno.Radius.Base != 0 {
		tk.BorderRadius = float32(a.bruno.Radius.Base)
		tk.BorderRadiusSM = float32(a.bruno.Radius.SM)
		tk.BorderRadiusLG = float32(a.bruno.Radius.LG)
	}

	if a.bruno.Border.Border0.Hex != "" {
		tk.Border = toUIColor(a.bruno.Border.Border0.Color())
		tk.BorderSecondary = toUIColor(a.bruno.Border.Border1.Color())
		tk.Split = toUIColor(a.bruno.Border.Subtle.Color())
	}
}

func toUIColor(c color.NRGBA) ui.Color {
	return ui.RGBA(c.R, c.G, c.B, float32(c.A)/255)
}

// SetBrunoTheme installs a Bruno theme snapshot, or clears it with nil.
// The snapshot's method colours and muted fallback override the appearance
// its Mode names (matched case-insensitively: "dark"/"Dark"/"DARK" → dark,
// anything else, including empty, → light); the other appearance keeps
// Bruno's built-in table, so a light theme import can never repaint dark
// mode. Method entries merge over the defaults, so a theme that ships only
// some verbs keeps the rest. `request.methods` — where Bruno's own themes
// put the table — wins over a top-level `methods`.
func (a *Adapter) SetBrunoTheme(b *BrunoTheme) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bruno = b
	a.brunoGen++
	// Rebuild from the built-in tables so clearing a theme restores them.
	a.methods = [2]map[string]ui.Color{
		defaultMethodColors(false),
		defaultMethodColors(true),
	}
	a.muted = brunoMuted

	if b == nil {
		return
	}
	slot := modeIndex(b.isDark())
	for k, v := range b.Methods {
		a.methods[slot][strings.ToLower(k)] = toUIColor(v.Color())
	}
	if len(b.Request.Methods) > 0 {
		for k, v := range b.Request.Methods {
			a.methods[slot][strings.ToLower(k)] = toUIColor(v.Color())
		}
	}
	if b.Colors.Text.Muted.Hex != "" {
		a.muted[slot] = toUIColor(b.Colors.Text.Muted.Color())
	}
}

func (a *Adapter) LoadBrunoTheme(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var bt BrunoTheme
	if err := json.Unmarshal(data, &bt); err != nil {
		return err
	}
	a.SetBrunoTheme(&bt)
	return nil
}

// MethodColor returns the colour for an HTTP method in the given appearance,
// the exact value of Bruno's request.methods table for that mode. The lookup
// is case-insensitive the way MethodBadge normalizes it, and a method with no
// entry (TRACE, CONNECT, a custom verb) takes the theme's colors.text.muted,
// exactly as Bruno's MethodBadge falls back.
func (a *Adapter) MethodColor(method string, dark bool) ui.Color {
	a.mu.RLock()
	defer a.mu.RUnlock()
	slot := modeIndex(dark)
	if c, ok := a.methods[slot][strings.ToLower(method)]; ok {
		return c
	}
	return a.muted[slot]
}

func (a *Adapter) Tokens(c *ui.Context) *theme.Tokens {
	return theme.TokensOf(c)
}

var globalAdapter *Adapter
var once sync.Once

func Global() *Adapter {
	once.Do(func() {
		globalAdapter = NewAdapter()
	})
	return globalAdapter
}

func Apply(c *ui.Context) {
	Global().Apply(c)
}

func TokensOf(c *ui.Context) *theme.Tokens {
	return Global().Tokens(c)
}

func MethodColor(method string, dark bool) ui.Color {
	return Global().MethodColor(method, dark)
}

// MethodColorFor is MethodColor for the frame being built: the appearance is
// the context's own, so it follows the system like Apply does. Widgets read
// the table through this.
func MethodColorFor(c *ui.Context, method string) ui.Color {
	return Global().MethodColor(method, c.Theme().Dark)
}

func LoadBrunoTheme(path string) error {
	return Global().LoadBrunoTheme(path)
}

func SetBrunoTheme(b *BrunoTheme) {
	Global().SetBrunoTheme(b)
}
