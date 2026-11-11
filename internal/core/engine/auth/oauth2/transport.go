package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// maxTokenResponseBytes caps a token response.
//
// Token responses are small. The cap is what stops a misconfigured endpoint that
// points at a large resource from reading it into memory.
const maxTokenResponseBytes = 1 << 20

// buildFormRequest constructs a form POST for a token endpoint.
func buildFormRequest(ctx context.Context, endpoint string, form url.Values, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth2: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	return req, nil
}

// readLimited reads a response body up to the cap.
func readLimited(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxTokenResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("oauth2: read response: %w", err)
	}
	return data, nil
}

// decodeJSON unmarshals a response body.
func decodeJSON(data []byte, out any) error {
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("oauth2: response is not JSON: %w", err)
	}
	return nil
}
