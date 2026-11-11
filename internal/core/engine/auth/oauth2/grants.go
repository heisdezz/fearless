package oauth2

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"fearless/internal/core/model"
)

// fetchClientCredentials obtains a token for a machine-to-machine client.
//
// Source: bruno-requests/src/auth/oauth2-helper.ts fetchTokenClientCredentials
func (c *Client) fetchClientCredentials(ctx context.Context, s Settings) (Token, error) {
	if s.AccessTokenURL == "" {
		return Token{}, fmt.Errorf("%w: access token URL", ErrMissingConfig)
	}
	if s.ClientID == "" {
		return Token{}, fmt.Errorf("%w: client ID", ErrMissingConfig)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	if strings.TrimSpace(s.Scope) != "" {
		form.Set("scope", s.Scope)
	}

	headers := c.basicHeaders(s)
	if s.AdditionalParams != nil {
		c.applyAdditional(form, &headers, s.AdditionalParams.AccessTokenRequest)
	}
	c.applyClientCredentials(form, s)

	endpoint := c.withQueryParams(s.AccessTokenURL, s.AdditionalParams, func(p *model.OAuth2AdditionalParams) model.AKVSet {
		return p.AccessTokenRequest
	})

	token, err := c.postForm(ctx, endpoint, form, headers)
	if err != nil {
		return Token{}, err
	}
	token = c.normalise(token)
	c.cache().Put(s.CredentialsID, token)
	return token, nil
}

// fetchPassword obtains a token from a username and password.
//
// The resource owner password grant is deprecated by the OAuth2 best-current-
// practice document, but providers still offer it and Bruno supports it, so a
// collection that uses it must keep working.
//
// Source: bruno-requests/src/auth/oauth2-helper.ts fetchTokenPassword
func (c *Client) fetchPassword(ctx context.Context, s Settings) (Token, error) {
	if s.AccessTokenURL == "" {
		return Token{}, fmt.Errorf("%w: access token URL", ErrMissingConfig)
	}
	if s.Username == "" {
		return Token{}, fmt.Errorf("%w: username", ErrMissingConfig)
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", s.Username)
	form.Set("password", s.Password)
	if strings.TrimSpace(s.Scope) != "" {
		form.Set("scope", s.Scope)
	}

	headers := c.basicHeaders(s)
	if s.AdditionalParams != nil {
		c.applyAdditional(form, &headers, s.AdditionalParams.AccessTokenRequest)
	}
	c.applyClientCredentials(form, s)

	endpoint := c.withQueryParams(s.AccessTokenURL, s.AdditionalParams, func(p *model.OAuth2AdditionalParams) model.AKVSet {
		return p.AccessTokenRequest
	})

	token, err := c.postForm(ctx, endpoint, form, headers)
	if err != nil {
		return Token{}, err
	}
	token = c.normalise(token)
	c.cache().Put(s.CredentialsID, token)
	return token, nil
}

// fetchAuthorizationCode runs the interactive grant.
//
// The sequence is the one bruno-electron/src/utils/oauth2.js implements:
//
//  1. build the authorization URL: response_type=code, client_id, redirect_uri,
//     scope, optional PKCE challenge, and a state that is always randomised
//  2. hand it to the Authorizer, which opens a browser and returns the code
//  3. exchange the code at the token endpoint, sending code_verifier when PKCE
//     was used
//
// Source: bruno-electron/src/utils/oauth2.js getOAuth2AuthorizationCode
func (c *Client) fetchAuthorizationCode(ctx context.Context, s Settings) (Token, error) {
	if s.AuthorizationURL == "" {
		return Token{}, fmt.Errorf("%w: authorization URL", ErrMissingConfig)
	}
	if s.AccessTokenURL == "" {
		return Token{}, fmt.Errorf("%w: access token URL", ErrMissingConfig)
	}
	if c.Authorize == nil {
		return Token{}, fmt.Errorf("%w: the authorization-code grant needs a browser authorizer", ErrMissingConfig)
	}

	pkce := PKCEPair{}
	if s.PKCE {
		pkce = NewPKCEPair()
	}

	// The state is always randomised, even when the user supplied one: a fixed
	// state is no defence against an injected callback, and the user's text is
	// preserved as a prefix so their own handler still recognises it.
	state := NewState(s.State)

	authorizeURL, err := c.authorizationURL(s, state, pkce)
	if err != nil {
		return Token{}, err
	}

	code, err := c.Authorize.Authorize(ctx, authorizeURL, state)
	if err != nil {
		return Token{}, fmt.Errorf("oauth2: authorization: %w", err)
	}
	if code == "" {
		return Token{}, fmt.Errorf("%w: authorization returned no code", ErrMissingConfig)
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	if s.CallbackURL != "" {
		form.Set("redirect_uri", s.CallbackURL)
	}
	if pkce.Verifier != "" {
		form.Set("code_verifier", pkce.Verifier)
	}

	headers := c.basicHeaders(s)
	if s.AdditionalParams != nil {
		c.applyAdditional(form, &headers, s.AdditionalParams.AccessTokenRequest)
	}
	c.applyClientCredentials(form, s)

	endpoint := c.withQueryParams(s.AccessTokenURL, s.AdditionalParams, func(p *model.OAuth2AdditionalParams) model.AKVSet {
		return p.AccessTokenRequest
	})

	token, err := c.postForm(ctx, endpoint, form, headers)
	if err != nil {
		return Token{}, err
	}
	token = c.normalise(token)
	c.cache().Put(s.CredentialsID, token)
	return token, nil
}

// authorizationURL builds the URL the user's browser opens.
func (c *Client) authorizationURL(s Settings, state string, pkce PKCEPair) (string, error) {
	parsed, err := url.Parse(s.AuthorizationURL)
	if err != nil {
		return "", fmt.Errorf("oauth2: invalid authorization URL %q: %w", s.AuthorizationURL, err)
	}

	// Build on the existing query rather than replacing it: some providers ship a
	// URL that already carries a required parameter.
	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("client_id", s.ClientID)

	if s.CallbackURL != "" {
		query.Set("redirect_uri", s.CallbackURL)
	}
	if s.Scope != "" {
		query.Set("scope", s.Scope)
	}
	if pkce.Challenge != "" {
		query.Set("code_challenge", pkce.Challenge)
		query.Set("code_challenge_method", pkce.Method)
	}
	query.Set("state", state)

	if s.AdditionalParams != nil {
		for _, param := range s.AdditionalParams.AuthorizationRequest.QueryParams {
			if !param.Enabled {
				continue
			}
			if name := model.DerefString(param.Name); name != "" {
				query.Set(name, model.DerefString(param.Value))
			}
		}
		// The authorization step has no request body, so a "body" parameter in
		// this group has nowhere to go. It is surfaced through the Authorizer's
		// headers instead, which is where Bruno's `additionalHeaders` puts it.
	}

	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// AuthorizeURL exposes the URL builder for the CLI, which prints it instead of
// opening a browser.
func (c *Client) AuthorizeURL(s Settings, state string) (string, error) {
	pkce := PKCEPair{}
	if s.PKCE {
		pkce = NewPKCEPair()
	}
	return c.authorizationURL(s, state, pkce)
}

// Device grant defaults.
const (
	// DefaultDevicePollInterval is the minimum the RFC allows when the server
	// omits `interval`.
	DefaultDevicePollInterval = 5 * time.Second
	// DefaultDeviceTimeout bounds the whole flow.
	DefaultDeviceTimeout = 5 * time.Minute
)

// FetchDevice runs the device authorization grant.
//
// It is not one of Bruno's four grant names (Bruno has no device flow), but the
// package brief lists it, and it is the only interactive flow that works without
// a browser or a loopback port — which makes it the right choice for a CLI
// logging in over SSH.
//
// Source: RFC 8628 §3.4 (polling) and §3.5 (error handling)
func (c *Client) FetchDevice(ctx context.Context, s Settings) (Token, error) {
	s = s.applyDefaults()

	deviceURL := s.DeviceAuthorizationURL
	if deviceURL == "" {
		deviceURL = s.AccessTokenURL
	}
	if deviceURL == "" {
		return Token{}, fmt.Errorf("%w: device authorization URL", ErrMissingConfig)
	}

	form := url.Values{}
	form.Set("client_id", s.ClientID)
	if s.Scope != "" {
		form.Set("scope", s.Scope)
	}

	headers := c.basicHeaders(s)

	verification, err := c.requestDeviceCode(ctx, deviceURL, form, headers)
	if err != nil {
		return Token{}, err
	}

	if c.PromptDevice != nil {
		if !c.PromptDevice(verification.VerificationURIComplete, verification.UserCode) {
			return Token{}, fmt.Errorf("oauth2: device authorization cancelled")
		}
	}

	interval := verification.Interval
	if interval <= 0 {
		interval = DefaultDevicePollInterval
	}

	deadline := time.Now().Add(DefaultDeviceTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return Token{}, err
		}
		if time.Now().After(deadline) {
			return Token{}, fmt.Errorf("oauth2: device authorization timed out")
		}

		token, retryAfter, err := c.pollDevice(ctx, s, verification.DeviceCode)
		if err != nil {
			return Token{}, err
		}
		// retryAfter means the server is still waiting for the user, which is not
		// success: the poll returned no error and no token. Checking only the error
		// here would return an empty token on the first pending response.
		if !retryAfter {
			token = c.normalise(token)
			c.cache().Put(s.CredentialsID, token)
			return token, nil
		}

		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// deviceVerification is the device authorization response.
type deviceVerification struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int64
	Interval                time.Duration
}

// requestDeviceCode asks the server for a user code.
func (c *Client) requestDeviceCode(ctx context.Context, endpoint string, form url.Values, headers map[string]string) (deviceVerification, error) {
	body, err := c.postRaw(ctx, endpoint, form, headers)
	if err != nil {
		return deviceVerification{}, err
	}

	var raw map[string]any
	if err := decodeJSON(body, &raw); err != nil {
		return deviceVerification{}, err
	}

	verification := deviceVerification{}
	verification.DeviceCode, _ = raw["device_code"].(string)
	verification.UserCode, _ = raw["user_code"].(string)
	verification.VerificationURI, _ = raw["verification_uri"].(string)
	verification.VerificationURIComplete, _ = raw["verification_uri_complete"].(string)

	if value, ok := raw["expires_in"].(float64); ok {
		verification.ExpiresIn = int64(value)
	}
	if value, ok := raw["interval"].(float64); ok && value > 0 {
		verification.Interval = time.Duration(value) * time.Second
	}

	// Both codes are mandatory. Without this check a misconfigured URL that
	// returns a token response instead of a device response would be parsed as an
	// empty verification and then poll against a device code of "", which fails
	// later with a far less useful message.
	if verification.DeviceCode == "" || verification.UserCode == "" {
		return deviceVerification{}, fmt.Errorf("oauth2: device authorization response is incomplete (device_code=%q user_code=%q)",
			verification.DeviceCode, verification.UserCode)
	}

	if verification.VerificationURIComplete == "" {
		// A server that omits the complete URI gets the plain one, so the prompt
		// still has something to show.
		verification.VerificationURIComplete = verification.VerificationURI
	}
	return verification, nil
}

// pollDevice makes one token poll.
//
// `retryAfter` reports the two errors the RFC says to keep polling through:
// authorization_pending and slow_down. Everything else is terminal.
func (c *Client) pollDevice(ctx context.Context, s Settings, deviceCode string) (Token, bool, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("device_code", deviceCode)
	c.applyClientCredentials(form, s)

	headers := c.basicHeaders(s)

	req, err := buildFormRequest(ctx, s.AccessTokenURL, form, headers)
	if err != nil {
		return Token{}, false, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Token{}, false, fmt.Errorf("oauth2: device poll: %w", err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body)
	if err != nil {
		return Token{}, false, err
	}

	token, decodeErr := decodeToken(body)
	if decodeErr == nil && token.AccessToken != "" {
		return token, false, nil
	}

	// The error code decides whether to keep polling.
	if token.Raw != nil {
		code, _ := token.Raw["error"].(string)
		switch code {
		case "authorization_pending", "slow_down":
			return Token{}, true, nil
		}
	}
	if decodeErr != nil {
		return Token{}, false, decodeErr
	}
	return Token{}, false, fmt.Errorf("oauth2: device authorization failed: %s", tokenError(token.Raw))
}

// postRaw performs a form POST and returns the body.
func (c *Client) postRaw(ctx context.Context, endpoint string, form url.Values, headers map[string]string) ([]byte, error) {
	req, err := buildFormRequest(ctx, endpoint, form, headers)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth2: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth2: device authorization returned %d: %s", resp.StatusCode, truncate(string(body), 256))
	}
	return body, nil
}
