package model

// Settings is the persisted application configuration.
//
// SchemaVersion gates migrations: a settings file from an older build is
// upgraded in memory and written back once, never discarded.
type Settings struct {
	SchemaVersion int                `json:"schemaVersion"`
	General       GeneralSettings    `json:"general"`
	Request       RequestDefaults    `json:"request"`
	Proxy         ProxySettings      `json:"proxy"`
	TLS           TLSSettings        `json:"tls"`
	Editor        EditorSettings     `json:"editor"`
	Appearance    AppearanceSettings `json:"appearance"`
	Privacy       PrivacySettings    `json:"privacy"`
}

// CurrentSettingsSchemaVersion is the version this build writes.
const CurrentSettingsSchemaVersion = 1

// GeneralSettings are miscellaneous application preferences.
type GeneralSettings struct {
	// ConfirmOnClose asks before closing with unsaved changes.
	ConfirmOnClose bool `json:"confirmOnClose"`
	// AutosaveIntervalSec is the debounce for autosave; 0 disables autosave.
	AutosaveIntervalSec int `json:"autosaveIntervalSec"`
	// RestoreLastSession reopens the previous tabs at startup.
	RestoreLastSession bool `json:"restoreLastSession"`
	// Language is the UI locale.
	Language string `json:"language,omitempty"`
}

// RequestDefaults are applied when a request does not override them.
type RequestDefaults struct {
	// TimeoutMs is the default request timeout in milliseconds.
	TimeoutMs int `json:"timeoutMs"`
	// FollowRedirects sets the default redirect behaviour.
	FollowRedirects bool `json:"followRedirects"`
	// MaxRedirects bounds the redirect chain.
	MaxRedirects int `json:"maxRedirects"`
	// EncodeURL percent-encodes the URL by default.
	EncodeURL bool `json:"encodeUrl"`
	// ForwardAuthorizationHeader keeps auth on cross-host redirects.
	ForwardAuthorizationHeader bool `json:"forwardAuthorizationHeader"`
	// MaxResponseBytes caps the in-memory body size before spilling to disk.
	MaxResponseBytes int64 `json:"maxResponseBytes"`
	// InlineBodyBytes is the largest body kept in memory rather than spooled.
	InlineBodyBytes int64 `json:"inlineBodyBytes"`
}

// ProxyMode selects how a proxy is chosen.
type ProxyMode string

const (
	ProxyOff    ProxyMode = "off"
	ProxyManual ProxyMode = "manual"
	ProxySystem ProxyMode = "system"
	ProxyPAC    ProxyMode = "pac"
)

// ProxySettings configures outbound proxying.
type ProxySettings struct {
	Mode     ProxyMode `json:"mode"`
	Protocol string    `json:"protocol,omitempty"` // http | https | socks5
	Hostname string    `json:"hostname,omitempty"`
	Port     string    `json:"port,omitempty"`
	Username string    `json:"username,omitempty"`
	Password *string   `json:"password,omitempty"`
	PACURL   string    `json:"pacUrl,omitempty"`
	// Bypass lists hosts that skip the proxy.
	Bypass []string `json:"bypass,omitempty"`
}

// TLSSettings configures certificate handling.
type TLSSettings struct {
	// VerifyServerCertificates disables verification when false.
	VerifyServerCertificates bool `json:"verifyServerCertificates"`
	// CACertFile overrides the system CA bundle when set.
	CACertFile string `json:"caCertFile,omitempty"`
	// ClientCertificates are global mutual-TLS certificates.
	ClientCertificates *ClientCertificates `json:"clientCertificates,omitempty"`
}

// EditorSettings configures text editors.
type EditorSettings struct {
	FontSize   int    `json:"fontSize"`
	FontFamily string `json:"fontFamily,omitempty"`
	TabSize    int    `json:"tabSize"`
	WordWrap   bool   `json:"wordWrap"`
	// FormatOnSend pretty-prints JSON bodies before sending.
	FormatOnSend bool `json:"formatOnSend"`
}

// AppearanceSettings configures the look.
type AppearanceSettings struct {
	// Theme is light | dark | system.
	Theme string `json:"theme"`
	// AccentColor overrides the primary colour when set.
	AccentColor string `json:"accentColor,omitempty"`
	// Zoom is the UI scale percentage.
	Zoom int `json:"zoom"`
	// ReduceMotion disables animations.
	ReduceMotion bool `json:"reduceMotion"`
	// IconPack selects the icon set.
	IconPack string `json:"iconPack,omitempty"`
}

// PrivacySettings controls what is stored.
type PrivacySettings struct {
	// SaveRequestHistory records executions.
	SaveRequestHistory bool `json:"saveRequestHistory"`
	// HistoryRetainDays prunes history older than this; 0 keeps forever.
	HistoryRetainDays int `json:"historyRetainDays"`
	// SaveResponseBodies retains bodies in history.
	SaveResponseBodies bool `json:"saveResponseBodies"`
	// TelemetryEnabled opts into anonymous telemetry.
	TelemetryEnabled bool `json:"telemetryEnabled"`
}

// DefaultSettings returns the settings a fresh install starts with.
func DefaultSettings() Settings {
	return Settings{
		SchemaVersion: CurrentSettingsSchemaVersion,
		General: GeneralSettings{
			ConfirmOnClose:      true,
			AutosaveIntervalSec: 0,
			RestoreLastSession:  true,
			Language:            "en",
		},
		Request: RequestDefaults{
			TimeoutMs:                  0, // 0 means no timeout, as in Bruno
			FollowRedirects:            DefaultFollowRedirects,
			MaxRedirects:               DefaultMaxRedirects,
			EncodeURL:                  DefaultEncodeURL,
			ForwardAuthorizationHeader: DefaultForwardAuthorizationHeader,
			MaxResponseBytes:           512 << 20, // 512 MiB hard cap
			InlineBodyBytes:            32 << 20,  // 32 MiB kept in memory
		},
		Proxy: ProxySettings{Mode: ProxyOff},
		TLS:   TLSSettings{VerifyServerCertificates: true},
		Editor: EditorSettings{
			FontSize: 13,
			TabSize:  2,
		},
		Appearance: AppearanceSettings{Theme: "system", Zoom: 100},
		Privacy: PrivacySettings{
			SaveRequestHistory: true,
			HistoryRetainDays:  30,
			SaveResponseBodies: true,
		},
	}
}
