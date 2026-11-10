package auth

import (
	"context"

	"fearless/internal/core/model"
)

// bearerProvider implements Authorization: Bearer <token>.
type bearerProvider struct{}

func (bearerProvider) Mode() model.AuthMode { return model.AuthBearer }

// Prepare sets the Authorization header.
//
// An empty token yields "Bearer " with a trailing space, which is what Bruno
// sends. It is a real (if useless) behaviour and matching it keeps a collection
// that relies on the header being *present* working unchanged.
func (bearerProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	token := interpolate(deps, model.DerefString(cfg.Token))
	return Prepared{
		Headers: []Header{{Name: "Authorization", Value: "Bearer " + token}},
	}, nil
}

// apiKeyProvider injects an API key as a header or a query parameter.
type apiKeyProvider struct{}

func (apiKeyProvider) Mode() model.AuthMode { return model.AuthAPIKey }

// Prepare applies the key according to Placement.
//
// An empty key name is a no-op, matching Bruno: a half-configured API key should
// not send a nameless header.
func (apiKeyProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	key := interpolate(deps, model.DerefString(cfg.Key))
	if key == "" {
		return Prepared{}, nil
	}
	value := interpolate(deps, model.DerefString(cfg.Value))

	switch cfg.Placement {
	case model.APIKeyQueryParams:
		return Prepared{QueryParams: []QueryParam{{Name: key, Value: value}}}, nil
	default:
		// Bruno defaults to the header placement when unset.
		return Prepared{Headers: []Header{{Name: key, Value: value}}}, nil
	}
}

// noneProvider is the explicit "no auth" mode.
//
// It exists so Resolve can stop inheritance at a request that deliberately opts
// out, rather than continuing to an ancestor's credentials.
type noneProvider struct{}

func (noneProvider) Mode() model.AuthMode { return model.AuthNone }

func (noneProvider) Prepare(_ context.Context, _ *model.Auth, _ Deps) (Prepared, error) {
	return Prepared{}, nil
}
