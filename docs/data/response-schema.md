# Response Schema

Status: VERIFIED unless marked. There is no formal single schema file; this document is reconstructed from the objects actually produced and consumed.

---

## 1. Transport result (main process → IPC)

```ts
interface TransportResult {
  status: number;
  statusText: string;
  headers: Record<string, string | string[]>;
  data: any;                 // parsed JSON if possible, else string ('' for streams)
  dataBuffer: string;        // base64 of the raw bytes
  size: number;              // Buffer.byteLength(dataBuffer)
  duration: number;          // ms
  url: string | null;        // final URL from response.request
  timeline: TimelineEntry[];
  requestSent: RequestSent;
  stream?: Readable;         // present when the response is streamed
  sseChunks?: string[];
  cancelTokenUid: string;
  error?: string;            // transport error message
  statusTextOverride?: string;
  isCancel?: boolean;
}
```

Source: `packages/bruno-electron/src/ipc/network/index.js` return statement.

### Error variants

| Variant | Fields |
|---|---|
| Cancelled | `{ statusText: 'REQUEST_CANCELLED', isCancel: true, error: 'REQUEST_CANCELLED', timeline }` |
| Transport error | `{ statusText, error, timeline }` |
| Outer exception | `{ status, error, timeline, requestSent }` |

---

## 2. Renderer response object (Redux)

```ts
interface UIResponse {
  state?: 'success';
  status?: number | string;   // 'Error' for transport failures
  statusText?: string;
  headers?: Record<string, string>;
  data: any;
  dataBuffer: string;         // base64
  size: number;
  duration: number;
  timeline?: Array<{ timestamp: number; type: string; message: string }>;
  stream?: any;
  isError?: boolean;
  error?: string;
}
```

Source: `packages/bruno-app/src/utils/network/index.js` `sendNetworkRequest` and the catch branch in `actions.js`.

---

## 3. Timeline entry

```ts
interface TimelineEntry {
  timestamp: number | Date;   // serialized to epoch ms across IPC
  type: 'separator' | 'info' | 'request' | 'requestData' | 'requestHeader'
      | 'response' | 'responseHeader' | 'timing' | 'error';
  message: string;
}
```

Defined in `bruno-requests/src/utils/timeline-agent.ts`; produced in `bruno-electron/src/ipc/network/axios-instance.js`.

---

## 4. Request sent

```ts
interface RequestSent {
  url: string;
  method: string;
  headers: Record<string, string>;
  data: any;            // string or object for display
  dataBuffer: string;   // base64 (may be null)
  timestamp: number;
}
```

Produced in `runRequest` before sending (`requestSent`), also emitted as `main:run-request-event type=request-sent`.

---

## 5. Test and assertion results

```ts
interface TestResult {
  status: 'pass' | 'fail' | 'skip';
  description: string;
  expected?: any;
  actual?: any;
  error?: string;
  path?: string;        // script source location
  line?: number;
}

interface AssertionResult {
  status: 'pass' | 'fail';
  lhsExpr: string;
  rhsExpr: string;
  operator: string;
  rhsOperand: any;
  error?: string;
}

interface TestResultsPayload { summary?: { total, passed, failed, skipped }, results: TestResult[] }
```

Stored on items as `testResults`, `assertionResults`, `preRequestTestResults`, `postResponseTestResults`.

---

## 6. Examples (persisted responses)

An `example` block captures a request/response pair:

```bru
example {
  name: 200 OK
  request { url: … method: GET }
  response {
    status: 200
    body:json { … }
  }
}
```

`tryResponseExample` reproduces it in the UI without sending.

---

## 7. Size and duration semantics

| Field | Meaning |
|---|---|
| `size` | Decompressed body byte length |
| `duration` | From request preparation start to last byte (or stream close) |
| `timeline.timing` | Sub-phase durations (prepared, connect, TLS, first byte) |

---

## 8. Go (PROPOSED)

```go
type TimelineEntry struct {
    At      time.Time `json:"at"`
    Type    string    `json:"type"`
    Message string    `json:"message"`
}

type Result struct {
    Status    int               `json:"status"`
    StatusText string           `json:"statusText"`
    Proto     string            `json:"proto"`
    Headers   map[string][]string `json:"headers"`
    Body      BodyRef           `json:"body"`        // id into a body store
    BodyText  string            `json:"-"`          // cached
    Size      int64             `json:"size"`
    Duration  time.Duration     `json:"duration"`
    FinalURL  string            `json:"finalUrl"`
    Timeline  []TimelineEntry   `json:"timeline"`
    Sent      *SentRequest      `json:"sent,omitempty"`
    Tests     []TestResult      `json:"tests,omitempty"`
    Assertions []AssertionResult `json:"assertions,omitempty"`
    Err       *Error            `json:"error,omitempty"`
}
```

Serialize bodies by reference (id + size + content-type) so huge bodies never enter the UI state store. This is the single biggest structural improvement over base64-over-IPC.
