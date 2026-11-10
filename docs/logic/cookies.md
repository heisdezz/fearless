# Cookies

Status: VERIFIED unless marked.

---

## 1. Model

A **single global cookie jar** (per app instance) implemented with `tough-cookie`:

```ts
const cookieJar = new CookieJar();
```

Source: `packages/bruno-requests/src/cookies/index.ts`

There is no per-collection jar and no per-tab jar: cookies from one collection are visible to another if domains match. This is intentional (matches browser behavior) but surprising when testing two tenants on the same host.

---

## 2. API surface (`@usebruno/requests.cookies`)

| Function | Purpose |
|---|---|
| `jar` | The `CookieJar` instance |
| `addCookieToJar(setCookieHeader, requestUrl)` | Parse one `Set-Cookie` line (`{ loose: true }`) and store |
| `getCookiesForUrl(url)` | Cookies matching the URL with `secure` flag set when the origin is "potentially trustworthy" |
| `getCookieStringForUrl(url)` | `name=value; …` for request injection (expired filtered) |
| `getDomainsWithCookies()` | `[{ domain, cookies[], cookieString }]` for the UI |
| `deleteCookie(domain, path, key)` | Remove one |
| `deleteCookiesForDomain(domain)` | Remove all for a domain |
| `addCookieForDomain(domain, cookieObj)` | UI-created/edited cookie |
| `updateCookieObj` / `createCookieObj` | Date normalization (`Infinity` for session cookies) |

### Special cases

- `__Host-` prefixed cookies: `hostOnly` must be true, so the jar must not be given an explicit domain (comment in source).
- The `secure` option passed to `getCookiesSync` is set from `isPotentiallyTrustworthyOrigin(url)` — **not** the URL's own scheme only: https, wss, file, and loopback hosts count as secure.

---

## 3. When cookies are attached

In `configureRequest` (after interpolation, before send):

```js
if (preferencesUtil.shouldSendCookies()) {
  const cookieString = getCookieStringForUrl(request.url);
  if (cookieString) {
    // merge with an explicit Cookie header, jar values winning on conflict
    request.headers[existingCookieHeaderName || 'Cookie'] = combinedCookieString;
  }
}
```

Merge detail: both strings are parsed into `{name: value}` maps; the jar is spread last, so **jar values override** user-supplied duplicates with the same name.

---

## 4. When cookies are stored

```js
if (preferencesUtil.shouldStoreCookies()) saveCookies(request.url, response.headers);
```

- In `saveCookies` (network/index.js line 53): `set-cookie` may be a string or an array (Node returns an array for multiple headers) — both handled.
- Also called during redirects **before** following (per-hop cookie capture).
- Also called when `followRedirects` is false (so a 302 still persists its cookies).

---

## 5. UI synchronization

After any cookie change the main process pushes:

```js
getDomainsWithCookies() → main:cookies-update
```

Emitted after pre-request scripts, after the response, after post-response scripts, and after the tests block. The `Cookies` component renders `ModifyCookieModal`, domain list, and per-cookie edit/delete.

---

## 6. Persistence

`CookiesStore` (`bruno-electron/src/store/cookies.js`):

| Aspect | Detail |
|---|---|
| Storage | `electron-store` file `cookies` |
| Encryption | A random 32-byte passkey is generated once, encrypted with the app-level encryption key (`utils/encryption.js`), and stored as `encryptedPasskey`; cookie **values** are encrypted with it |
| Fallback | If encryption fails, cookies are stored unencrypted with a console warning |
| Write timing | Debounced 5 s (`DEBOUNCE_MS = 5000`) via `saveCookieJar()` |
| Restore | `initializeCookies()` loads and rehydrates the jar at startup |
| Expiry | Filtered on read (`!expires || expires > Date.now()`) |

---

## 7. Manual cookie editing

`Cookies/ModifyCookieModal` lets you create/edit a cookie (domain, path, name, value, expiry, secure, httpOnly). Writes go through `addCookieForDomain` / `updateCookieObj` and are pushed to the jar immediately.

---

## 8. Preferences

| Preference | Default | Effect |
|---|---|---|
| `request.storeCookies` | `true` | Persist response cookies |
| `request.sendCookies` | `true` | Attach jar cookies to requests |

Both default to `true` (VERIFIED, `store/preferences.js` lines 21–22).

---

## 9. Edge cases

| Case | Behavior |
|---|---|
| User sets `Cookie` header and jar has values | Merged; jar wins per-name |
| `Set-Cookie` with invalid syntax | Ignored silently (`loose` parse, `ignoreError: true`) |
| Redirect chain | Each hop stores its cookies, then next hop reads the jar for the new URL |
| Cross-origin redirect | Cookies for the new origin are attached (standard browser behavior) |
| Session cookies | `expires: Infinity` in UI; session-ish in the jar |
| Multiple `Set-Cookie` headers | Handled as an array from Node |

---

## 10. Go implementation (PROPOSED)

Use `net/http/cookiejar` with a custom `PublicSuffixList` (e.g. `golang.org/x/net/publicsuffix`) and a custom `Policy` to implement the "potentially trustworthy origin" secure rule.

```go
type CookieStore interface {
    AddRaw(setCookieLine, requestURL string)
    CookiesFor(url string) []*http.Cookie
    HeaderFor(url string) string
    Domains() []DomainCookies
    Delete(domain, path, name string) error
    Persist() error
}
```

Notes:

- `cookiejar` does not reject `__Host-` misconfiguration; validate manually for parity.
- Encrypt cookie values at rest with an OS-keyring-held key.
- Keep the 5 s debounce; cookie-heavy sites otherwise hammer the disk.
