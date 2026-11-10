package bru

import (
	"strings"

	"fearless/internal/core/model"
)

// Auth converts the `auth { mode: … }` marker and its `auth:<mode>` block into
// the auth union.
//
// Bruno stores the mode in one block and the mode's fields in another, so both
// must be read. A file with `auth: bearer` but no `auth:bearer` block is legal and
// yields a bearer config with an empty token.
func (d *Document) Auth() (*model.Auth, error) {
	modeBlock, ok := d.FindBlock("auth")
	if !ok {
		return nil, nil
	}

	auth := &model.Auth{Mode: model.AuthInherit}
	for _, p := range modeBlock.Pairs {
		if p.Key == "mode" {
			auth.Mode = model.AuthMode(strings.TrimSpace(p.Value))
		}
	}
	if auth.Mode == "" {
		auth.Mode = model.AuthInherit
	}

	// The mode's own block carries the fields.
	modeName := string(auth.Mode)
	if blk, ok := d.FindBlock("auth:" + modeName); ok {
		applyAuthFields(auth, blk)
	}

	return auth, nil
}

// applyAuthFields copies the keys of an `auth:<mode>` block onto the union.
//
// The mapping is a straight name-for-name translation because the model uses
// Bruno's own field names; that keeps the round-trip obvious and avoids a second
// naming scheme to learn.
func applyAuthFields(auth *model.Auth, blk *Block) {
	for _, p := range blk.Pairs {
		v := p.Value
		str := func() *string {
			val := v
			return &val
		}
		switch p.Key {
		case "username":
			auth.Username = str()
		case "password":
			auth.Password = str()
		case "domain":
			auth.Domain = str()
		case "token":
			auth.Token = str()
		case "key":
			auth.Key = str()
		case "value":
			auth.Value = str()
		case "placement":
			auth.Placement = model.APIKeyPlacement(strings.TrimSpace(v))
		case "accessKeyId":
			auth.AccessKeyID = str()
		case "secretAccessKey":
			auth.SecretAccessKey = str()
		case "sessionToken":
			auth.SessionToken = str()
		case "service":
			auth.Service = str()
		case "region":
			auth.Region = str()
		case "profileName":
			auth.ProfileName = str()
		case "consumerKey":
			auth.ConsumerKey = str()
		case "consumerSecret":
			auth.ConsumerSecret = str()
		case "accessToken":
			auth.AccessToken = str()
		case "accessTokenSecret":
			auth.AccessTokenSecret = str()
		case "callbackUrl":
			auth.CallbackURL = str()
		case "verifier":
			auth.Verifier = str()
		case "signatureMethod":
			auth.SignatureMethod = str()
		case "privateKey":
			auth.PrivateKey = str()
		case "privateKeyType":
			auth.PrivateKeyType = str()
		case "timestamp":
			auth.Timestamp = str()
		case "nonce":
			auth.Nonce = str()
		case "version":
			auth.Version = str()
		case "realm":
			auth.Realm = str()
		case "includeBodyHash":
			if b, ok := parseBool(strings.TrimSpace(v)); ok {
				auth.IncludeBodyHash = &b
			}
		case "grantType":
			auth.GrantType = model.OAuth2GrantType(strings.TrimSpace(v))
		case "authorizationUrl":
			auth.AuthorizationURL = str()
		case "accessTokenUrl":
			auth.AccessTokenURL = str()
		case "refreshTokenUrl":
			auth.RefreshTokenURL = str()
		case "clientId":
			auth.ClientID = str()
		case "clientSecret":
			auth.ClientSecret = str()
		case "scope":
			auth.Scope = str()
		case "state":
			auth.State = str()
		case "pkce":
			if b, ok := parseBool(strings.TrimSpace(v)); ok {
				auth.PKCE = &b
			}
		case "credentialsPlacement":
			auth.CredentialsPlacement = str()
		case "credentialsId":
			auth.CredentialsID = str()
		case "tokenPlacement":
			auth.TokenPlacement = str()
		case "tokenHeaderPrefix":
			auth.TokenHeaderPrefix = str()
		case "tokenQueryKey":
			auth.TokenQueryKey = str()
		case "autoRefreshToken":
			if b, ok := parseBool(strings.TrimSpace(v)); ok {
				auth.AutoRefreshToken = &b
			}
		case "autoFetchToken":
			if b, ok := parseBool(strings.TrimSpace(v)); ok {
				auth.AutoFetchToken = &b
			}
		case "tokenSource":
			auth.TokenSource = str()
		case "clientToken":
			auth.EdgeGridClientToken = str()
		case "baseURL":
			auth.EdgeGridBaseURL = str()
		case "headersToSign":
			auth.EdgeGridHeadersToSign = str()
		case "maxBodySize":
			if n, err := parseInt(strings.TrimSpace(v)); err == nil {
				auth.EdgeGridMaxBodySize = &n
			}
		}
	}
}

// parseInt parses an integer, returning an error for anything else.
func parseInt(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, errEmptyInt
	}
	neg := false
	for i, c := range s {
		if i == 0 && (c == '-' || c == '+') {
			neg = c == '-'
			continue
		}
		if c < '0' || c > '9' {
			return 0, errBadInt
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}
