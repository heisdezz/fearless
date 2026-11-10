# Error Management (Logic)

Status: VERIFIED unless marked.

Companion to `flows/error-handling.md`. This document specifies the _types_, _mapping_, and _recovery_ rules.

---

## 1. Typed error model (PROPOSED, modeled on Bruno's actual cases)

Bruno uses ad-hoc objects. The following Go types cover every case observed:

```go
type Kind int
const (
    KindTransport Kind = iota // DNS, connect, TLS, reset, EOF
    KindTimeout
    KindCanceled
    KindTooManyRedirects
    KindInvalidURL
    KindProxySetup
    KindScript        // pre-request script threw
    KindVariableEval
    KindGraphQLVars
    KindAuth          // oauth1/oauth2 signing or token
    KindTLS           // certificate/handshake
    KindFileIO        // body file missing/unreadable
)

type Error struct {
    Kind     Kind
    Op       string  // phase: prepare|script|auth|send|post
    Err      error
    Detail   string  // user-facing summary
    Timeline []TimelineEntry
    Partial  *PartialResults // script/test results gathered before failure
}
```

---

## 2. Mapping from observed implementations

| Bruno signal                                           | Kind                     | Where detected                                                       |
| ------------------------------------------------------ | ------------------------ | -------------------------------------------------------------------- |
| `axios.isCancel(error)`                                | `KindCanceled`           | `runRequest`                                                         |
| `error.response` present                               | _(not an error)_         | `runRequest`                                                         |
| `ECONNREFUSED`, `ENOTFOUND`, `ECONNRESET`, `EAI_AGAIN` | `KindTransport`          | axios/Node                                                           |
| `ETIMEDOUT`, `timeout of Xms exceeded`                 | `KindTimeout`            | axios                                                                |
| Redirect limit hit                                     | `KindTooManyRedirects`   | redirect interceptor                                                 |
| `{ message: 'Invalid URL format' }`                    | `KindInvalidURL`         | `interpolate-vars.js`                                                |
| `Failed to parse GraphQL variables: …`                 | `KindGraphQLVars`        | `runPreRequest`                                                      |
| `OAuth1 signing failed: …`                             | `KindAuth`               | `configureRequest`                                                   |
| Script exception                                       | `KindScript`             | `ScriptRuntime`                                                      |
| Variable expression exception                          | `KindVariableEval`       | `VarsRuntime` (aggregated)                                           |
| `Error reading cert/key file`                          | `KindTLS`/`KindFileIO`   | `cert-utils.js`                                                      |
| `Error reading file:` (body)                           | `KindFileIO` (non-fatal) | `prepare-request.js`                                                 |
| Proxy config incomplete                                | `KindProxySetup`         | `http-https-agents.ts` (`Proxy protocol and hostname are required…`) |

---

## 3. Fatal vs. non-fatal

| Phase              | Failure                   | Effect                                                   |
| ------------------ | ------------------------- | -------------------------------------------------------- |
| Prepare            | Invalid URL, GraphQL vars | **Abort** before send                                    |
| Prepare            | Body file missing         | **Continue** without body (logged)                       |
| Pre-request script | Throws                    | **Abort**; partial results forwarded                     |
| Auth               | Signing/token error       | **Abort**                                                |
| Send               | Transport error           | **Abort**; `onFail` callbacks run                        |
| Send               | Cancel                    | **Abort**                                                |
| Send               | 4xx/5xx                   | **Continue** (normal response)                           |
| Response parse     | JSON parse fails          | **Continue** with text                                   |
| Post vars          | One expression throws     | **Continue**; others evaluated; aggregated error toasted |
| Post script        | Throws                    | **Continue** to assertions/tests with partial results    |
| Assertions/tests   | Fail                      | **Continue**; reported                                   |

---

## 4. Recovery strategies

| Failure             | Recovery available in UI                                          |
| ------------------- | ----------------------------------------------------------------- |
| TLS verification    | Disable verification or add CA in preferences                     |
| Proxy misconfig     | Fix preferences                                                   |
| Redirect loop       | Increase `maxRedirects` or disable follow                         |
| Script error        | Click error → jump to line                                        |
| OAuth token expired | Re-authorize ("Get Token") or enable auto-refresh                 |
| Unresolved variable | Fix scope/environment; the request still ran with literal `{{x}}` |
| Body file missing   | Re-select file                                                    |
| Timeout             | Raise `timeout`                                                   |

---

## 5. Logging and diagnostics

| Channel                 | Contents                                                 |
| ----------------------- | -------------------------------------------------------- |
| Response timeline       | Full hop-by-hop trace, including errors and timing       |
| Devtools console        | Script `console.*` output                                |
| `main:display-error`    | Aggregated errors (variables, OAuth)                     |
| `console.error` in main | File read failures, post-script errors (not user-facing) |
| Runner report           | Per-request status, error message, test results          |

The timeline is the primary artifact for support: a user attaching a timeline reproduces the request precisely (except secrets).

---

## 6. Secret redaction

| Location                 | Redaction                                                                                                                                                                                                                                                                                |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Request body (file mode) | `<request body redacted>`                                                                                                                                                                                                                                                                |
| Sensitive field UI       | `SensitiveFieldWarning`, `utils/sensitive-fields.js`: an explicit path list (`request.auth.*.password`, `bearer.token`, `apikey.value`, `awsv4.secretAccessKey`, `oauth1.consumerSecret`, akamai tokens…) marks fields masked and warns when a plain (non-secret) variable is used there |
| Share/export             | Sensitive value detection warns before sharing                                                                                                                                                                                                                                           |
| Env secrets              | Never written to env files                                                                                                                                                                                                                                                               |

A Go port should redact by **header/field name match** with an allow-list override, and never log bodies by default.

---

## 7. Error UX guidelines (observed)

1. Errors never block the whole window (no global spinner).
2. The response pane always explains _why_ the request failed.
3. Script errors point to a line number in the file the user edited, not the wrapped script.
4. Partial results are shown (a script that fails after 2 passing tests shows 2 passes + 1 error).
5. Aggregated variable errors list every failing variable, not just the first.
