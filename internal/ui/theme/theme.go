package theme

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type BrunoTheme struct {
	Name    string         `json:"name"`
	Mode    string         `json:"mode"`
	Brand   NRGBA          `json:"brand"`
	Primary PrimaryColors  `json:"primary"`
	Background BackgroundColors `json:"background"`
	Status  map[string]StatusColors `json:"status"`
	Font    FontSizes      `json:"font"`
	Radius  BorderRadius   `json:"radius"`
	Border  BorderColors   `json:"border"`
	Methods map[string]NRGBA `json:"methods"`
	Colors  ColorExtras    `json:"colors"`
	Input   InputColors    `json:"input"`
	Request RequestColors  `json:"request"`
	WS      WSColors       `json:"ws"`
	CodeMirror CodeMirrorColors `json:"codemirror"`
}

type NRGBA struct {
	R, G, B, A uint8 `json:"-"`
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
	XS  int `json:"xs"`
	SM  int `json:"sm"`
	Base int `json:"base"`
	MD  int `json:"md"`
	LG  int `json:"lg"`
	XL  int `json:"xl"`
}

type BorderRadius struct {
	SM  int `json:"sm"`
	Base int `json:"base"`
	MD  int `json:"md"`
	LG  int `json:"lg"`
	XL  int `json:"xl"`
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

type Adapter struct {
	mu      sync.RWMutex
	bruno   *BrunoTheme
	cfg     *theme.Config
	methods map[string]ui.Color
}

func NewAdapter() *Adapter {
	return &Adapter{
		methods: defaultMethodColors(),
	}
}

func defaultMethodColors() map[string]ui.Color {
	return map[string]ui.Color{
		"get":    ui.Hex("#34a853"),
		"post":   ui.Hex("#fbbc04"),
		"put":    ui.Hex("#ea4335"),
		"delete": ui.Hex("#ea4335"),
		"patch":  ui.Hex("#ea4335"),
		"options": ui.Hex("#4285f4"),
		"head":   ui.Hex("#9aa0a6"),
	}
}

func (a *Adapter) Apply(c *ui.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg == nil {
		lightTokens := *theme.Light()
		darkTokens := *theme.Dark()
		a.cfg = theme.NewConfig(lightTokens, darkTokens)
	}

	if a.bruno != nil {
		a.applyBrunoTheme()
	}

	a.cfg.Apply(c)
}

func (a *Adapter) applyBrunoTheme() {
	if a.bruno == nil {
		return
	}

	lightTokens := *theme.Light()
	darkTokens := *theme.Dark()

	a.mapBrunoToTokens(&lightTokens, false)
	a.mapBrunoToTokens(&darkTokens, true)

	a.cfg = theme.NewConfig(lightTokens, darkTokens)
}

func (a *Adapter) mapBrunoToTokens(tk *theme.Tokens, dark bool) {
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

func (a *Adapter) SetBrunoTheme(b *BrunoTheme) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bruno = b
	if b != nil && b.Methods != nil {
		for k, v := range b.Methods {
			a.methods[k] = toUIColor(v.Color())
		}
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

func (a *Adapter) MethodColor(method string) ui.Color {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if c, ok := a.methods[method]; ok {
		return c
	}
	return ui.Hex("#6b7280")
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

func MethodColor(method string) ui.Color {
	return Global().MethodColor(method)
}

func LoadBrunoTheme(path string) error {
	return Global().LoadBrunoTheme(path)
}

func SetBrunoTheme(b *BrunoTheme) {
	Global().SetBrunoTheme(b)
}