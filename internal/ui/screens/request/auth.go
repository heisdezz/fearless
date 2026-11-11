package request

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

var authModeOptions = []field.SelectOption{
	{Value: string(model.AuthInherit), Label: "Inherit Auth"},
	{Value: string(model.AuthNone), Label: "No Auth"},
	{Value: string(model.AuthBasic), Label: "Basic Auth"},
	{Value: string(model.AuthBearer), Label: "Bearer Token"},
	{Value: string(model.AuthAPIKey), Label: "API Key"},
	{Value: string(model.AuthDigest), Label: "Digest Auth"},
	{Value: string(model.AuthOAuth2), Label: "OAuth 2.0"},
}

// renderAuthTab renders the authentication mode selector and credentials form.
func renderAuthTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	tk := lobetheme.TokensOf(c)

	if req.Auth == nil {
		req.Auth = &model.Auth{Mode: model.AuthNone}
	}

	selectedMode := string(req.Auth.Mode)
	if selectedMode == "" {
		selectedMode = string(model.AuthNone)
	}

	return ui.Column(c).FillWidth().Gap(16).Children(func() {
		// 1. Auth mode selector dropdown
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "Auth Type:").FontSize(11).Bold().TextColor(tk.TextSecondary)

			field.Select(c, &selectedMode, field.SelectProps{
				Options: authModeOptions,
				Size:    lobetheme.Small,
			})
			req.Auth.Mode = model.AuthMode(selectedMode)
		})

		// 2. Auth mode specific form
		switch req.Auth.Mode {
		case model.AuthInherit:
			ui.Box(c).Padding(16, 0).Children(func() {
				ui.Text(c, "Inheriting authentication from parent collection or folder.").
					FontSize(12).
					TextColor(tk.TextSecondary)
			})

		case model.AuthNone:
			ui.Box(c).Padding(16, 0).Children(func() {
				ui.Text(c, "This request does not use any authentication.").
					FontSize(12).
					TextColor(tk.TextSecondary)
			})

		case model.AuthBasic, model.AuthDigest:
			userStr := ""
			if req.Auth.Username != nil {
				userStr = *req.Auth.Username
			}
			passStr := ""
			if req.Auth.Password != nil {
				passStr = *req.Auth.Password
			}

			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Username").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &userStr, field.InputProps{
						Placeholder: "Username",
						Size:        lobetheme.Small,
					})
					req.Auth.Username = &userStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Password").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &passStr, field.InputProps{
						Placeholder: "Password",
						Size:        lobetheme.Small,
					})
					req.Auth.Password = &passStr
				})
			})

		case model.AuthBearer:
			tokenStr := ""
			if req.Auth.Token != nil {
				tokenStr = *req.Auth.Token
			}

			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Box(c).Width(80).Children(func() {
					ui.Text(c, "Token").FontSize(12).TextColor(tk.Text)
				})
				field.Input(c, &tokenStr, field.InputProps{
					Placeholder: "Bearer token",
					Size:        lobetheme.Small,
				})
				req.Auth.Token = &tokenStr
			})

		case model.AuthAPIKey:
			keyStr := ""
			if req.Auth.Key != nil {
				keyStr = *req.Auth.Key
			}
			valStr := ""
			if req.Auth.Value != nil {
				valStr = *req.Auth.Value
			}
			placementStr := string(req.Auth.Placement)
			if placementStr == "" {
				placementStr = string(model.APIKeyHeader)
			}

			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Key").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &keyStr, field.InputProps{
						Placeholder: "X-API-Key",
						Size:        lobetheme.Small,
					})
					req.Auth.Key = &keyStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Value").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &valStr, field.InputProps{
						Placeholder: "Value",
						Size:        lobetheme.Small,
					})
					req.Auth.Value = &valStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Add To").FontSize(12).TextColor(tk.Text)
					})
					field.Select(c, &placementStr, field.SelectProps{
						Options: []field.SelectOption{
							{Value: string(model.APIKeyHeader), Label: "Header"},
							{Value: string(model.APIKeyQueryParams), Label: "Query Params"},
						},
						Size: lobetheme.Small,
					})
					req.Auth.Placement = model.APIKeyPlacement(placementStr)
				})
			})

		default:
			ui.Box(c).Padding(16, 0).Children(func() {
				ui.Text(c, "Configure authentication options above.").
					FontSize(12).
					TextColor(tk.TextSecondary)
			})
		}
	})
}
