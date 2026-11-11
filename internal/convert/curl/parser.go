package curl

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"fearless/internal/core/model"
)

func newUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Result holds the output of importing a cURL command.
type Result struct {
	Item     *model.Item
	Request  *model.Request
	Warnings []string
}

// Parse converts a cURL command string into a Result containing a model.Request
// and model.Item, plus any warnings for dropped or adapted flags.
func Parse(cmd string) (*Result, error) {
	tokens, err := Tokenize(cmd)
	if err != nil {
		return nil, fmt.Errorf("tokenizing curl command: %w", err)
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty curl command")
	}

	var (
		rawURL          string
		method          string
		headers         []model.KeyValue
		droppedFlags    []string
		rawBodyChunks   []string
		urlEncodeItems  []string
		multipartItems  []string
		fileBodyPath    string
		contentType     string
		basicUser       string
		basicPass       string
		bearerToken     string
		authMode        model.AuthMode
		isGet           bool
		followRedirects *bool
		insecure        bool
	)

	n := len(tokens)
	for i := 0; i < n; i++ {
		arg := tokens[i]

		// Handle long options with '=': --key=val
		var optKey, optVal string
		hasInlineVal := false
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			idx := strings.Index(arg, "=")
			optKey = arg[:idx]
			optVal = arg[idx+1:]
			hasInlineVal = true
		} else if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			// Handle attached short option: -XPOST, -HHeader:Value, -uUser:Pass, -dData, etc.
			switch arg[1] {
			case 'X', 'H', 'u', 'd', 'A', 'e', 'b', 'F', 'm', 'x', 'o', 'c':
				optKey = arg[:2]
				optVal = arg[2:]
				hasInlineVal = true
			default:
				optKey = arg
			}
		} else {
			optKey = arg
		}

		nextVal := func() (string, bool) {
			if hasInlineVal {
				return optVal, true
			}
			if i+1 < n {
				i++
				return tokens[i], true
			}
			return "", false
		}

		switch optKey {
		case "-X", "--request":
			if val, ok := nextVal(); ok {
				method = strings.ToUpper(strings.TrimSpace(val))
			}
		case "-I", "--head":
			method = "HEAD"
		case "--url":
			if val, ok := nextVal(); ok {
				rawURL = strings.TrimSpace(val)
			}
		case "-H", "--header":
			if val, ok := nextVal(); ok {
				val = strings.TrimSpace(val)
				if strings.Contains(val, ":") {
					parts := strings.SplitN(val, ":", 2)
					k := strings.TrimSpace(parts[0])
					v := strings.TrimSpace(parts[1])

					if strings.EqualFold(k, "Content-Type") {
						contentType = v
					} else if strings.EqualFold(k, "Authorization") {
						lower := strings.ToLower(v)
						if strings.HasPrefix(lower, "bearer ") {
							bearerToken = strings.TrimSpace(v[7:])
						} else if strings.HasPrefix(lower, "basic ") {
							enc := strings.TrimSpace(v[6:])
							if dec, decErr := base64.StdEncoding.DecodeString(enc); decErr == nil {
								s := string(dec)
								if strings.Contains(s, ":") {
									bp := strings.SplitN(s, ":", 2)
									basicUser = bp[0]
									basicPass = bp[1]
								}
							}
						}
					}

					headers = append(headers, model.KeyValue{
						UID:     newUID(),
						Name:    &k,
						Value:   &v,
						Enabled: true,
					})
				} else if strings.HasSuffix(val, ";") {
					k := strings.TrimSuffix(val, ";")
					v := ""
					headers = append(headers, model.KeyValue{
						UID:     newUID(),
						Name:    &k,
						Value:   &v,
						Enabled: true,
					})
				}
			}
		case "-u", "--user":
			if val, ok := nextVal(); ok {
				val = strings.TrimSpace(val)
				if strings.Contains(val, ":") {
					parts := strings.SplitN(val, ":", 2)
					basicUser = parts[0]
					basicPass = parts[1]
				} else {
					basicUser = val
					basicPass = ""
				}
			}
		case "--basic":
			authMode = model.AuthBasic
		case "--digest":
			authMode = model.AuthDigest
		case "--ntlm":
			authMode = model.AuthNTLM
		case "--bearer", "--oauth2-bearer":
			if val, ok := nextVal(); ok {
				bearerToken = strings.TrimSpace(val)
				authMode = model.AuthBearer
			}
		case "--aws-sigv4":
			if _, ok := nextVal(); ok {
				authMode = model.AuthAWSV4
			}
		case "-d", "--data", "--data-raw", "--data-ascii":
			if val, ok := nextVal(); ok {
				rawBodyChunks = append(rawBodyChunks, val)
			}
		case "--data-binary":
			if val, ok := nextVal(); ok {
				if strings.HasPrefix(val, "@") {
					fileBodyPath = strings.TrimPrefix(val, "@")
				} else {
					rawBodyChunks = append(rawBodyChunks, val)
				}
			}
		case "--data-urlencode":
			if val, ok := nextVal(); ok {
				urlEncodeItems = append(urlEncodeItems, val)
			}
		case "-F", "--form", "--form-string":
			if val, ok := nextVal(); ok {
				multipartItems = append(multipartItems, val)
			}
		case "-b", "--cookie":
			if val, ok := nextVal(); ok {
				val = strings.TrimSpace(val)
				if strings.Contains(val, "=") {
					k := "Cookie"
					headers = append(headers, model.KeyValue{
						UID:     newUID(),
						Name:    &k,
						Value:   &val,
						Enabled: true,
					})
				} else {
					droppedFlags = append(droppedFlags, fmt.Sprintf("Cookie jar file '%s' not loaded into request header", val))
				}
			}
		case "-A", "--user-agent":
			if val, ok := nextVal(); ok {
				k := "User-Agent"
				headers = append(headers, model.KeyValue{
					UID:     newUID(),
					Name:    &k,
					Value:   &val,
					Enabled: true,
				})
			}
		case "-e", "--referer":
			if val, ok := nextVal(); ok {
				k := "Referer"
				headers = append(headers, model.KeyValue{
					UID:     newUID(),
					Name:    &k,
					Value:   &val,
					Enabled: true,
				})
			}
		case "-k", "--insecure":
			insecure = true
			droppedFlags = append(droppedFlags, "Flag '--insecure' / '-k' noted (TLS certificate verification disabled)")
		case "-L", "--location":
			t := true
			followRedirects = &t
		case "-G", "--get":
			isGet = true
		case "-s", "--silent", "-S", "--show-error", "-v", "--verbose", "-i", "--include":
			// Informational output flags; safe to ignore
		case "--compressed":
			droppedFlags = append(droppedFlags, "Flag '--compressed' noted (compression handled automatically)")
		case "--proxy", "-x":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Proxy flag '--proxy %s' dropped (configure proxies in collection or preferences)", val))
			}
		case "-m", "--max-time":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Timeout flag '--max-time %s' dropped (timeout configured in settings)", val))
			}
		case "--connect-timeout":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Connect timeout '--connect-timeout %s' dropped", val))
			}
		case "--cacert":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("CA certificate '--cacert %s' dropped (configure in collection security)", val))
			}
		case "--cert", "-E":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Client certificate '--cert %s' dropped (configure in collection security)", val))
			}
		case "--key":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Client key '--key %s' dropped", val))
			}
		case "-o", "--output":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Output flag '--output %s' dropped", val))
			}
		case "-c", "--cookie-jar":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Cookie jar flag '--cookie-jar %s' dropped", val))
			}
		case "--retry":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Retry flag '--retry %s' dropped", val))
			}
		case "--resolve":
			if val, ok := nextVal(); ok {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Resolve flag '--resolve %s' dropped", val))
			}
		default:
			// If not starting with '-', treat as positional URL if not set yet
			if !strings.HasPrefix(arg, "-") && rawURL == "" {
				rawURL = strings.TrimSpace(arg)
			} else if strings.HasPrefix(arg, "-") {
				droppedFlags = append(droppedFlags, fmt.Sprintf("Unrecognized flag '%s' dropped", arg))
			}
		}
	}

	if rawURL == "" {
		return nil, fmt.Errorf("no URL found in cURL command")
	}

	// Determine HTTP Method
	if method == "" {
		if isGet {
			method = "GET"
		} else if len(rawBodyChunks) > 0 || len(urlEncodeItems) > 0 || len(multipartItems) > 0 || fileBodyPath != "" {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	// Parse URL and extract Query Parameters
	parsedURL, queryParams := extractURLAndParams(rawURL)

	// If -G / --get was specified, body chunks convert into query parameters
	if isGet && len(rawBodyChunks) > 0 {
		for _, chunk := range rawBodyChunks {
			pairs := strings.Split(chunk, "&")
			for _, p := range pairs {
				if strings.Contains(p, "=") {
					kv := strings.SplitN(p, "=", 2)
					k, _ := url.QueryUnescape(kv[0])
					v, _ := url.QueryUnescape(kv[1])
					queryParams = append(queryParams, model.Param{
						Type: model.ParamQuery,
						KeyValue: model.KeyValue{
							UID:     newUID(),
							Name:    &k,
							Value:   &v,
							Enabled: true,
						},
					})
				}
			}
		}
		rawBodyChunks = nil
	}

	// Build Body
	var body *model.Body
	if fileBodyPath != "" {
		body = &model.Body{
			Mode: model.BodyFile,
			File: []model.FileEntry{
				{
					UID:      newUID(),
					FilePath: &fileBodyPath,
					Selected: true,
				},
			},
		}
	} else if len(multipartItems) > 0 {
		body = &model.Body{
			Mode: model.BodyMultipartForm,
		}
		for _, item := range multipartItems {
			if strings.Contains(item, "=") {
				idx := strings.Index(item, "=")
				name := item[:idx]
				content := item[idx+1:]
				if strings.HasPrefix(content, "@") {
					spec := strings.TrimPrefix(content, "@")
					filePath := spec
					var ct *string
					if strings.Contains(spec, ";") {
						parts := strings.Split(spec, ";")
						filePath = parts[0]
						for _, p := range parts[1:] {
							p = strings.TrimSpace(p)
							if strings.HasPrefix(strings.ToLower(p), "type=") {
								t := strings.Trim(strings.TrimPrefix(p, "type="), `"'`)
								ct = &t
							}
						}
					}
					val := "@file(" + filePath + ")"
					body.MultipartForm = append(body.MultipartForm, model.MultipartEntry{
						UID:         newUID(),
						Type:        model.MultipartFile,
						Name:        &name,
						Value:       &val,
						ContentType: ct,
						Enabled:     true,
					})
				} else {
					body.MultipartForm = append(body.MultipartForm, model.MultipartEntry{
						UID:     newUID(),
						Type:    model.MultipartText,
						Name:    &name,
						Value:   &content,
						Enabled: true,
					})
				}
			}
		}
	} else if len(urlEncodeItems) > 0 {
		body = &model.Body{
			Mode: model.BodyFormURLEncoded,
		}
		for _, item := range urlEncodeItems {
			if strings.Contains(item, "=") {
				idx := strings.Index(item, "=")
				k := item[:idx]
				v := item[idx+1:]
				body.FormURLEncoded = append(body.FormURLEncoded, model.KeyValue{
					UID:     newUID(),
					Name:    &k,
					Value:   &v,
					Enabled: true,
				})
			} else {
				k := item
				v := ""
				body.FormURLEncoded = append(body.FormURLEncoded, model.KeyValue{
					UID:     newUID(),
					Name:    &k,
					Value:   &v,
					Enabled: true,
				})
			}
		}
	} else if len(rawBodyChunks) > 0 {
		data := strings.Join(rawBodyChunks, "&")
		isJSON := strings.Contains(strings.ToLower(contentType), "json")
		trimmed := strings.TrimSpace(data)
		if !isJSON && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
			var js json.RawMessage
			if json.Unmarshal([]byte(data), &js) == nil {
				isJSON = true
			}
		}

		if isJSON {
			body = &model.Body{
				Mode: model.BodyJSON,
				JSON: &data,
			}
			// Check for GraphQL payload
			var gql struct {
				Query     *string `json:"query"`
				Variables any     `json:"variables"`
			}
			if json.Unmarshal([]byte(data), &gql) == nil && gql.Query != nil && *gql.Query != "" {
				var varStr string
				if gql.Variables != nil {
					if vb, err := json.Marshal(gql.Variables); err == nil {
						varStr = string(vb)
					}
				}
				body.GraphQL = &model.GraphQLBody{
					Query:     gql.Query,
					Variables: &varStr,
				}
			}
		} else if strings.Contains(strings.ToLower(contentType), "xml") || strings.HasPrefix(trimmed, "<") {
			body = &model.Body{
				Mode: model.BodyXML,
				XML:  &data,
			}
		} else if strings.Contains(strings.ToLower(contentType), "x-www-form-urlencoded") || (len(rawBodyChunks) > 1 && strings.Contains(data, "=")) {
			body = &model.Body{
				Mode: model.BodyFormURLEncoded,
			}
			pairs := strings.Split(data, "&")
			for _, p := range pairs {
				if strings.Contains(p, "=") {
					kv := strings.SplitN(p, "=", 2)
					k, _ := url.QueryUnescape(kv[0])
					v, _ := url.QueryUnescape(kv[1])
					body.FormURLEncoded = append(body.FormURLEncoded, model.KeyValue{
						UID:     newUID(),
						Name:    &k,
						Value:   &v,
						Enabled: true,
					})
				}
			}
		} else {
			body = &model.Body{
				Mode: model.BodyText,
				Text: &data,
			}
		}
	}

	// Build Auth
	var auth *model.Auth
	if authMode == model.AuthDigest {
		auth = &model.Auth{
			Mode:     model.AuthDigest,
			Username: &basicUser,
			Password: &basicPass,
		}
	} else if authMode == model.AuthNTLM {
		auth = &model.Auth{
			Mode:     model.AuthNTLM,
			Username: &basicUser,
			Password: &basicPass,
		}
	} else if authMode == model.AuthAWSV4 {
		auth = &model.Auth{
			Mode: model.AuthAWSV4,
		}
	} else if bearerToken != "" {
		auth = &model.Auth{
			Mode:  model.AuthBearer,
			Token: &bearerToken,
		}
	} else if basicUser != "" || basicPass != "" {
		auth = &model.Auth{
			Mode:     model.AuthBasic,
			Username: &basicUser,
			Password: &basicPass,
		}
	}

	// Create Request
	req := &model.Request{
		URL:     parsedURL,
		Method:  method,
		Headers: headers,
		Params:  queryParams,
		Auth:    auth,
		Body:    body,
		Settings: model.ItemSettings{
			FollowRedirects: followRedirects,
		},
	}

	// Derive Item Name from URL path or host
	name := deriveItemName(parsedURL)

	item := &model.Item{
		UID:     newUID(),
		Type:    model.ItemHTTPRequest,
		Name:    name,
		Request: req,
	}

	_ = insecure // reported in warnings

	return &Result{
		Item:     item,
		Request:  req,
		Warnings: droppedFlags,
	}, nil
}

func extractURLAndParams(rawURL string) (string, []model.Param) {
	rawURL = strings.TrimSpace(rawURL)
	var queryParams []model.Param

	// Split off query string if present
	qIdx := strings.Index(rawURL, "?")
	if qIdx == -1 {
		return rawURL, nil
	}

	baseURL := rawURL[:qIdx]
	queryString := rawURL[qIdx+1:]

	// Separate fragment if present
	if fIdx := strings.Index(queryString, "#"); fIdx != -1 {
		queryString = queryString[:fIdx]
	}

	pairs := strings.Split(queryString, "&")
	for _, p := range pairs {
		if p == "" {
			continue
		}
		var k, v string
		if strings.Contains(p, "=") {
			parts := strings.SplitN(p, "=", 2)
			k, _ = url.QueryUnescape(parts[0])
			v, _ = url.QueryUnescape(parts[1])
		} else {
			k, _ = url.QueryUnescape(p)
			v = ""
		}
		queryParams = append(queryParams, model.Param{
			Type: model.ParamQuery,
			KeyValue: model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: true,
			},
		})
	}

	return baseURL, queryParams
}

func deriveItemName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil && u.Path != "" && u.Path != "/" {
		clean := strings.Trim(u.Path, "/")
		parts := strings.Split(clean, "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			return parts[len(parts)-1]
		}
	}
	if err == nil && u.Host != "" {
		return u.Host
	}
	return "Imported cURL"
}
