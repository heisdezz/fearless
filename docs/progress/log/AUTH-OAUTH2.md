# AUTH-OAUTH2 — OAuth2 grants, PKCE, cache and refresh

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/core/engine/auth/oauth2/`
**Reference:** `bruno-requests/src/auth/oauth2-helper.ts`, `bruno-electron/src/utils/oauth2.js`

## Scope

Four grants, a credentials-keyed token cache, refresh, and the token placement
rules.

| Grant | Bruno source | Status |
| --- | --- | --- |
| `client_credentials` | `oauth2-helper.ts` | implemented |
| `password` | `oauth2-helper.ts` | implemented |
| `authorization_code` (+ PKCE) | `oauth2.js` (electron utils) | implemented |
| device code | not in Bruno | implemented (package brief asks for it) |
| `implicit` | not supported in Bruno | rejected with a clear error |

## Why a separate package

The flow is stateful in a way no other provider is: it needs a token endpoint, a
cache, refresh policy, and for the interactive grant a browser and a loopback
listener. Keeping it out of `internal/core/engine/auth` means the platform
interfaces never leak into the provider registry.

`Authorizer` is an interface for exactly that reason: the GUI supplies one backed
by `internal/platform` (system browser plus loopback), and the CLI supplies one
that prints the URL — the acceptance criterion "CLI prints the URL instead of
opening a browser" is satisfied by `Client.AuthorizeURL` plus a stdin-reading
`Authorizer` above this package, not by a browser dependency here.

## Cache and refresh

- Tokens are keyed by **credentials ID**, not by token URL. Two requests share a
  token only when the user says they are the same credentials.
- A token with **no `expires_in` is never considered expired**. Bruno does the
  same; refetching on every request against a server that omits the lifetime
  would be worse than the occasional stale token.
- An expired token is **refreshed first**, because that avoids the interactive
  step entirely. A refresh failure falls back to a full fetch, and a refresh that
  omits the new refresh token keeps the old one — dropping it would make the
  second refresh impossible.
- `expires_in` arriving as a **string** is parsed. Some servers quote it, and a
  struct decode would reject the whole response.

### The `FetchDisabled` field

A `bool` cannot distinguish "unset" from "false", so a hand-built `Settings` with
`AutoFetchToken` left at zero has two bad readings: fetch against the caller's
wishes, or fetch nothing and emit an empty `Authorization` header with no
explanation. `Settings.FetchDisabled` states the intent explicitly; `SettingsFrom`
sets it from the model's flags, whose default is *on* because Bruno's UI ships
them enabled.

### Defaulting lives in two places, on purpose

`SettingsFrom` is the normal entry point, but `Token`/`Fetch`/`Refresh` also call
`applyDefaults`, because a caller that assembled `Settings` by hand must get the
same behaviour. A test caught this: every literal-built `Settings` failed with
"automatic fetching is disabled" until the defaulting moved down.

## PKCE and state

- **S256 only.** `plain` offers no protection against an attacker who can read
  the authorization request.
- The verifier is 32 random bytes base64url-encoded — 43 characters, the RFC
  minimum — and `ChallengeFor` is exported so the pairing can be verified rather
  than trusted.
- **The state is always randomised**, and the user's configured state is kept as a
  *prefix*. A fixed state is no defence against an injected callback; the prefix
  keeps a user's own handler working.

## Token placement

`header` (default), `queryparams`, and `body`. The first two are implemented; the
body placement reports `SupportsBody() == false` rather than silently doing
nothing, because appending a token to an already-serialised payload — including a
multipart one — is a transport concern.

The header prefix defaults to the server's `token_type` before falling back to
`Bearer`, since a server issuing DPoP or MAC expects that scheme. An explicitly
blank prefix yields a bare token with no leading space.

## Bugs found while testing

Three, all real and all fixed:

1. **Device polling treated "pending" as success.** The loop checked only
   `err == nil`, but a pending poll returns `(zero Token, retryAfter=true, nil)`,
   so the first `authorization_pending` response ended the flow with an empty
   token. Found by the happy-path test, which reported one poll instead of three.
2. **A nil header map panicked.** `basicHeaders` returned nil when not using basic
   auth, and the next line wrote an additional header parameter into it.
3. **A misconfigured device URL was parsed as an empty verification.** Now the
   response is validated for both codes before polling starts, with the received
   values in the message.

## Verification

```
go vet ./internal/core/engine/auth/oauth2/     # clean
go test -race -cover ./internal/core/engine/auth/oauth2/
ok  ...  coverage: 89.0% of statements
```

Coverage: client-credentials in both placements plus the public-client case;
caching; refresh with token preservation; refresh failure falling back to fetch;
`FetchDisabled` in both directions; no-expiry tokens staying valid; expiry
arithmetic including the "no lifetime" case; `tokenSource = id_token`;
password grant; **authorization code end to end with PKCE**, verifying every
authorize-URL parameter and that the verifier really hashes to the challenge;
the non-PKCE flow; a missing authorizer, a failing authorizer, and a printable
`AuthorizeURL`; provider errors carrying `error` and `error_description`; a
non-JSON 502; a missing access token; six missing-config cases; implicit and
unknown grants rejected; additional parameters in all three destinations plus the
disabled case; `SettingsFrom` defaulting and an explicit false surviving;
independent caches per credentials ID; 16 concurrent token requests under
`-race`; the cache contract; 20 PKCE pairs for length, alphabet and hashing;
state randomisation and the preserved prefix; and seven device-grant tests
(pending-then-approved, `slow_down`, cancellation, terminal error, context
timeout, incomplete response, URL fallback) plus the placement rules.

Total: 46 tests.

## Not delivered

- **The GUI authorizer.** It needs `internal/platform`'s browser and loopback
  wired together, which belongs with the OAuth2 settings screen.
- **A `Provider` implementation for the auth registry.** `internal/core/engine/auth`
  expects providers to implement its `Provider` interface, and importing it from
  here would create a cycle (`auth` → `oauth2` → `auth`). The wiring belongs on
  the `auth` side, using `Deps.TokenFetcher` and `Deps.TokenStore`, which that
  package already declares for exactly this purpose.
- **Persistent token cache.** `MemoryCache` loses tokens on restart, so the user
  re-authorizes after relaunch. A disk-backed cache belongs with the secret store
  (PLATFORM), because a refresh token is a credential.

## Sources

- `bruno-requests/src/auth/oauth2-helper.ts` — `getOAuth2Token`,
  `fetchTokenClientCredentials`, `fetchTokenPassword`, `isTokenExpired`,
  `applyAdditionalParameters`, the `autoFetchToken`/`tokenSource` semantics
- `bruno-electron/src/utils/oauth2.js` — the authorization-code flow, PKCE
  (`code_challenge_method=S256`), state generation, token placement
- RFC 7636 (PKCE), RFC 8628 (device grant), RFC 6749 §4
