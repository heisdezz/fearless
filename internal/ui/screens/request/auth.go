package request

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

var authModeOptions = []field.SelectOption{
	{Value: string(model.AuthInherit), Label: "Inherit Auth"},
	{Value: string(model.AuthNone), Label: "No Auth"},
	{Value: string(model.AuthAWSV4), Label: "AWS Sig v4"},
	{Value: string(model.AuthBasic), Label: "Basic Auth"},
	{Value: string(model.AuthBearer), Label: "Bearer Token"},
	{Value: string(model.AuthDigest), Label: "Digest Auth"},
	{Value: string(model.AuthOAuth1), Label: "OAuth 1.0"},
	{Value: string(model.AuthOAuth2), Label: "OAuth 2.0"},
	{Value: string(model.AuthWSSE), Label: "WSSE Auth"},
	{Value: string(model.AuthNTLM), Label: "NTLM"},
	{Value: string(model.AuthAPIKey), Label: "API Key"},
	{Value: string(model.AuthEdgeGrid), Label: "Akamai EdgeGrid"},
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

			widgets.FieldSelect(c, &selectedMode, widgets.FieldSelectProps{
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

		case model.AuthBasic, model.AuthDigest, model.AuthNTLM, model.AuthWSSE:
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
					widgets.FieldInput(c, &userStr, widgets.FieldInputProps{
						Placeholder: "Username",
						Size:        lobetheme.Small,
					})
					req.Auth.Username = &userStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Password").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldInput(c, &passStr, widgets.FieldInputProps{
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
				widgets.FieldInput(c, &tokenStr, widgets.FieldInputProps{
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
					widgets.FieldInput(c, &keyStr, widgets.FieldInputProps{
						Placeholder: "X-API-Key",
						Size:        lobetheme.Small,
					})
					req.Auth.Key = &keyStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Value").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldInput(c, &valStr, widgets.FieldInputProps{
						Placeholder: "Value",
						Size:        lobetheme.Small,
					})
					req.Auth.Value = &valStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Add To").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldSelect(c, &placementStr, widgets.FieldSelectProps{
						Options: []field.SelectOption{
							{Value: string(model.APIKeyHeader), Label: "Header"},
							{Value: string(model.APIKeyQueryParams), Label: "Query Params"},
						},
						Size: lobetheme.Small,
					})
					req.Auth.Placement = model.APIKeyPlacement(placementStr)
				})
			})

		case model.AuthAWSV4:
			akStr := ""
			if req.Auth.AccessKeyID != nil {
				akStr = *req.Auth.AccessKeyID
			}
			skStr := ""
			if req.Auth.SecretAccessKey != nil {
				skStr = *req.Auth.SecretAccessKey
			}
			regStr := ""
			if req.Auth.Region != nil {
				regStr = *req.Auth.Region
			}
			srvStr := ""
			if req.Auth.Service != nil {
				srvStr = *req.Auth.Service
			}

			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(110).Children(func() {
						ui.Text(c, "Access Key ID").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldInput(c, &akStr, widgets.FieldInputProps{
						Placeholder: "AKIA...",
						Size:        lobetheme.Small,
					})
					req.Auth.AccessKeyID = &akStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(110).Children(func() {
						ui.Text(c, "Secret Access Key").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldInput(c, &skStr, widgets.FieldInputProps{
						Placeholder: "Secret Key",
						Size:        lobetheme.Small,
					})
					req.Auth.SecretAccessKey = &skStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(110).Children(func() {
						ui.Text(c, "Region").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldInput(c, &regStr, widgets.FieldInputProps{
						Placeholder: "us-east-1",
						Size:        lobetheme.Small,
					})
					req.Auth.Region = &regStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(110).Children(func() {
						ui.Text(c, "Service").FontSize(12).TextColor(tk.Text)
					})
					widgets.FieldInput(c, &srvStr, widgets.FieldInputProps{
						Placeholder: "s3, execute-api...",
						Size:        lobetheme.Small,
					})
					req.Auth.Service = &srvStr
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
