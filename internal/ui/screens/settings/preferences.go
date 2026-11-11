package settings

import (
	"context"
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

var prefTabsList = []struct {
	tab   presenters.PreferencesTab
	label string
}{
	{presenters.PrefTabGeneral, "General"},
	{presenters.PrefTabAppearance, "Appearance"},
	{presenters.PrefTabRequest, "Request"},
	{presenters.PrefTabProxy, "Proxy"},
	{presenters.PrefTabTLS, "TLS"},
	{presenters.PrefTabEditor, "Editor"},
	{presenters.PrefTabPrivacy, "Privacy"},
	{presenters.PrefTabOAuth, "OAuth"},
}

// PreferencesScreen renders the global application preferences modal/pane.
func PreferencesScreen(c *ui.Context, p *presenters.PreferencesPresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	if p == nil {
		return ui.Box(c)
	}

	cfg := p.Settings()
	activeTab := p.ActiveTab()

	return ui.Column(c).Fill().Padding(16, 20).Gap(16).Children(func() {
		// Header title
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Preferences").
				FontSize(16).
				Bold().
				TextColor(tk.Text)
		})

		// Main body: Categories navigation on left, settings controls on right
		ui.Row(c).Fill().Gap(20).Children(func() {
			// Left Category Nav — a plain left-aligned list, like Bruno's
			// settings rail. nav.Tabs centers every label inside its pill,
			// which misaligns the rail, and wrapping a sized container here
			// floats the list mid-window because Box and sized columns
			// center their children.
			ui.Column(c).Width(140).Shrink(0).Gap(2).Children(func() {
				for _, t := range prefTabsList {
					t := t
					isActive := t.tab == activeTab

					row := ui.Row(c).
						FillWidth().
						AlignItems(ui.Center).
						Height(26).
						Padding(0, 10).
						Radius(tk.BorderRadius)
					switch {
					case isActive:
						row.Background(tk.BgElevated).TextColor(tk.Text)
					case row.Hovered():
						row.Background(tk.BgTextHover)
					}
					row.Cursor(ui.CursorPointer).OnClick(func() {
						p.SelectTab(t.tab)
					})

					row.Children(func() {
						color := tk.TextSecondary
						if isActive || row.Hovered() {
							color = tk.Text
						}
						ui.Text(c, t.label).SingleLine().FontSize(12).TextColor(color)
					})
				}
			})

			// Right Content Pane. The trailing Grow spacer anchors the
			// content to the top: columns distribute leftover height by
			// centering, so a pane without one floats mid-window.
			ui.Column(c).Grow(1).Children(func() {
				switch activeTab {
				case presenters.PrefTabGeneral:
					renderGeneralPreferences(c, p, cfg.General)
				case presenters.PrefTabAppearance:
					renderAppearancePreferences(c, p, cfg.Appearance)
				case presenters.PrefTabRequest:
					renderRequestPreferences(c, p, cfg.Request)
				case presenters.PrefTabProxy:
					renderProxyPreferences(c, p, cfg.Proxy)
				case presenters.PrefTabTLS:
					renderTLSPreferences(c, p, cfg.TLS)
				case presenters.PrefTabEditor:
					renderEditorPreferences(c, p, cfg.Editor)
				case presenters.PrefTabPrivacy:
					renderPrivacyPreferences(c, p, cfg.Privacy)
				case presenters.PrefTabOAuth:
					renderOAuthPreferences(c, p)
				default:
					renderGeneralPreferences(c, p, cfg.General)
				}

				ui.Box(c).Grow(1)
			})
		})

		// Bottom Action Bar: Save and Revert buttons
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Grow(1)

			saveLabel := "Save"
			if p.IsDirty() {
				saveLabel = "Save *"
			}

			widgets.Button(c, saveLabel, widgets.ButtonProps{
				Variant: action.Primary,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				p.Save(context.Background())
			})

			widgets.Button(c, "Revert", widgets.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				p.Revert()
			})
		})
	})
}

func renderGeneralPreferences(c *ui.Context, p *presenters.PreferencesPresenter, g model.GeneralSettings) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "General Settings").FontSize(14).Bold().TextColor(tk.Text)

		// Confirm on close toggle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &g.ConfirmOnClose, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Confirm before closing with unsaved changes").FontSize(12).TextColor(tk.Text)
		})

		// Restore session toggle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &g.RestoreLastSession, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Restore last session and open tabs on startup").FontSize(12).TextColor(tk.Text)
		})

		// Language input
		langStr := g.Language
		if langStr == "" {
			langStr = "en"
		}
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(120).Children(func() {
				ui.Text(c, "Interface Language").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldInput(c, &langStr, widgets.FieldInputProps{
				Placeholder: "en",
				Size:        lobetheme.Small,
			})
			g.Language = langStr
		})

		p.SetGeneral(g)
	})
}

func renderAppearancePreferences(c *ui.Context, p *presenters.PreferencesPresenter, a model.AppearanceSettings) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "Appearance Settings").FontSize(14).Bold().TextColor(tk.Text)

		// Theme mode select
		themeStr := a.Theme
		if themeStr == "" {
			themeStr = "system"
		}
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(100).Children(func() {
				ui.Text(c, "Theme Mode").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldSelect(c, &themeStr, widgets.FieldSelectProps{
				Options: []field.SelectOption{
					{Value: "system", Label: "System Default"},
					{Value: "light", Label: "Light"},
					{Value: "dark", Label: "Dark"},
				},
				Size: lobetheme.Small,
			})
			a.Theme = themeStr
		})

		// Reduce Motion toggle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &a.ReduceMotion, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Reduce Motion (disable animations)").FontSize(12).TextColor(tk.Text)
		})

		p.SetAppearance(a)
	})
}

func renderRequestPreferences(c *ui.Context, p *presenters.PreferencesPresenter, r model.RequestDefaults) {
	tk := lobetheme.TokensOf(c)

	timeoutStr := fmt.Sprintf("%d", r.TimeoutMs)
	maxRedirStr := fmt.Sprintf("%d", r.MaxRedirects)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "Request Defaults").FontSize(14).Bold().TextColor(tk.Text)

		// Timeout Ms
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(140).Children(func() {
				ui.Text(c, "Request Timeout (ms)").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldInput(c, &timeoutStr, widgets.FieldInputProps{
				Placeholder: "0 (no timeout)",
				Size:        lobetheme.Small,
			})
			if val, err := strconv.Atoi(timeoutStr); err == nil {
				r.TimeoutMs = val
			}
		})

		// Follow Redirects toggle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &r.FollowRedirects, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Follow HTTP Redirects").FontSize(12).TextColor(tk.Text)
		})

		// Max Redirects
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(140).Children(func() {
				ui.Text(c, "Max Redirects").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldInput(c, &maxRedirStr, widgets.FieldInputProps{
				Placeholder: "5",
				Size:        lobetheme.Small,
			})
			if val, err := strconv.Atoi(maxRedirStr); err == nil {
				r.MaxRedirects = val
			}
		})

		// URL Encoding toggle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &r.EncodeURL, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Automatically Percent-Encode URLs").FontSize(12).TextColor(tk.Text)
		})

		p.SetRequest(r)
	})
}

func renderProxyPreferences(c *ui.Context, p *presenters.PreferencesPresenter, proxy model.ProxySettings) {
	tk := lobetheme.TokensOf(c)

	modeStr := string(proxy.Mode)
	if modeStr == "" {
		modeStr = string(model.ProxyOff)
	}

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "Proxy Configuration").FontSize(14).Bold().TextColor(tk.Text)

		// Proxy Mode
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(100).Children(func() {
				ui.Text(c, "Proxy Mode").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldSelect(c, &modeStr, widgets.FieldSelectProps{
				Options: []field.SelectOption{
					{Value: string(model.ProxyOff), Label: "Disabled"},
					{Value: string(model.ProxySystem), Label: "Use System Proxy"},
					{Value: string(model.ProxyManual), Label: "Manual Proxy"},
					{Value: string(model.ProxyPAC), Label: "PAC Script"},
				},
				Size: lobetheme.Small,
			})
			proxy.Mode = model.ProxyMode(modeStr)
		})

		if proxy.Mode == model.ProxyManual {
			ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Box(c).Width(100).Children(func() {
					ui.Text(c, "Hostname").FontSize(12).TextColor(tk.Text)
				})
				widgets.FieldInput(c, &proxy.Hostname, widgets.FieldInputProps{
					Placeholder: "127.0.0.1",
					Size:        lobetheme.Small,
				})
			})

			ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Box(c).Width(100).Children(func() {
					ui.Text(c, "Port").FontSize(12).TextColor(tk.Text)
				})
				widgets.FieldInput(c, &proxy.Port, widgets.FieldInputProps{
					Placeholder: "8080",
					Size:        lobetheme.Small,
				})
			})
		} else if proxy.Mode == model.ProxyPAC {
			ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Box(c).Width(100).Children(func() {
					ui.Text(c, "PAC URL").FontSize(12).TextColor(tk.Text)
				})
				widgets.FieldInput(c, &proxy.PACURL, widgets.FieldInputProps{
					Placeholder: "http://example.com/proxy.pac",
					Size:        lobetheme.Small,
				})
			})
		}

		p.SetProxy(proxy)
	})
}

func renderTLSPreferences(c *ui.Context, p *presenters.PreferencesPresenter, tls model.TLSSettings) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "TLS & SSL Certificates").FontSize(14).Bold().TextColor(tk.Text)

		// Verify certificates toggle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &tls.VerifyServerCertificates, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Verify Server Certificates").FontSize(12).TextColor(tk.Text)
		})

		// Custom CA Bundle
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(120).Children(func() {
				ui.Text(c, "CA Certificate File").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldInput(c, &tls.CACertFile, widgets.FieldInputProps{
				Placeholder: "/path/to/custom-ca.pem",
				Size:        lobetheme.Small,
			})
		})

		p.SetTLS(tls)
	})
}

func renderEditorPreferences(c *ui.Context, p *presenters.PreferencesPresenter, e model.EditorSettings) {
	tk := lobetheme.TokensOf(c)

	fontSizeStr := fmt.Sprintf("%d", e.FontSize)
	tabSizeStr := fmt.Sprintf("%d", e.TabSize)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "Code Editor Settings").FontSize(14).Bold().TextColor(tk.Text)

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(100).Children(func() {
				ui.Text(c, "Font Size").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldInput(c, &fontSizeStr, widgets.FieldInputProps{
				Placeholder: "13",
				Size:        lobetheme.Small,
			})
			if val, err := strconv.Atoi(fontSizeStr); err == nil {
				e.FontSize = val
			}
		})

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Width(100).Children(func() {
				ui.Text(c, "Tab Size").FontSize(12).TextColor(tk.Text)
			})
			widgets.FieldInput(c, &tabSizeStr, widgets.FieldInputProps{
				Placeholder: "2",
				Size:        lobetheme.Small,
			})
			if val, err := strconv.Atoi(tabSizeStr); err == nil {
				e.TabSize = val
			}
		})

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &e.WordWrap, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Soft Word Wrap").FontSize(12).TextColor(tk.Text)
		})

		p.SetEditor(e)
	})
}

func renderPrivacyPreferences(c *ui.Context, p *presenters.PreferencesPresenter, priv model.PrivacySettings) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "Privacy & Data Storage").FontSize(14).Bold().TextColor(tk.Text)

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &priv.SaveRequestHistory, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Save Request Execution History").FontSize(12).TextColor(tk.Text)
		})

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &priv.SaveResponseBodies, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Save Response Bodies in History").FontSize(12).TextColor(tk.Text)
		})

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			choice.Switch(c, &priv.TelemetryEnabled, choice.SwitchProps{Size: lobetheme.Small})
			ui.Text(c, "Enable Anonymous Diagnostic Telemetry").FontSize(12).TextColor(tk.Text)
		})

		p.SetPrivacy(priv)
	})
}

func renderOAuthPreferences(c *ui.Context, p *presenters.PreferencesPresenter) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, "OAuth 2.0 Credentials Cache").FontSize(14).Bold().TextColor(tk.Text)

		ui.Text(c, "Clear all cached OAuth2 access and refresh tokens across all collections.").
			FontSize(12).
			TextColor(tk.TextSecondary)

		widgets.Button(c, "Clear OAuth2 Token Cache", widgets.ButtonProps{
			Variant: action.Default,
			Size:    lobetheme.Small,
		}).OnClick(func() {
			p.ClearOAuthCache(context.Background(), "")
		})
	})
}
