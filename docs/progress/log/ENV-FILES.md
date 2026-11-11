# ENV-FILES — environment loading, `extends`, secrets, `.env`

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/core/environment/`
**Reference:** `bruno-common/src/utils/environment-inheritance.ts`, `bruno-electron/src/app/dotenv-watcher.js`

## Layout

| File | Responsibility |
| --- | --- |
| `resolve.go` | `extends` chains, cycle/missing detection, provenance, secrets, `Inheritable`, `Select` |
| `loader.go` | Directory scanning, codec dispatch, save/remove with path confinement |
| `dotenv.go` | `.env` grammar, loading, merge precedence |

## Resolution semantics

Taken from `resolveEnvironmentInheritance`, and deliberately faithful to its
quirks:

- **The chain is walked child → parent, then reversed.** The nearest ancestor
  wins over a farther one.
- **A name the child defines is *deleted* from the inherited set**, not merely
  overridden later. This is why `Resolution.Inherited` can be shorter than the
  union of ancestors: shadowed entries are gone, and the caller reads the
  child's own values from `Environment.Variables`.
- **Secrets and non-secrets are tracked in separate maps.** A plaintext variable
  cannot shadow a secret of the same name purely by ordering.
- **Non-secrets come first, then secrets sorted by name.** Bruno's map insertion
  order makes this incidental; sorting makes it deterministic here, which
  matters for tests and for a stable UI.
- **Duplicate environment names resolve to the first definition**, matching
  Bruno's `Array.find`.

### Deviations, and why

| Bruno | Here | Reason |
| --- | --- | --- |
| Returns partial ancestors alongside `missingInheritedEnvironmentName` / `cyclicInheritancePath`, and sends anyway | Returns typed errors (`MissingParentError`, `CycleError`) wrapping `ErrMissingParent` / `ErrCycle` | Sending with a half-resolved environment is how a request reaches production with the wrong host. The types still carry the name and path, so the UI can render exactly the same message. |
| Cycle reported as `[...walked.slice(start), parent.name]` | Same shape, `[a b a]` | Preserved so the message reads identically. |

`Resolution.Chain` is exposed even though Bruno does not surface it, because the
UI needs the ancestry for the "inherited from" column.

## Loader behaviour

- **Extension is chosen by dialect**, not sniffed: `.bru` collections read
  `environments/*.bru`, YAML collections read `environments/*.yml`. A stray file
  of the other extension is ignored rather than confusing the codec.
- **One malformed file fails the whole load.** Skipping it would quietly leave
  the picker one environment short, and the user would select a different one by
  mistake.
- **A missing `environments/` directory** returns `ErrNoEnvironments`, distinct
  from an I/O failure, because "this collection has no environments" is normal.
- **A file with no `name:`** falls back to the filename stem, as Bruno does.
- **Files are sorted case-insensitively by name** so the picker is stable.
- **Saves are confined to `environments/`.** Display names are sanitised
  (`/`, `\`, `:`, NUL) so an environment named `Prod/US` writes `Prod-US.bru`
  rather than trying to create a subdirectory.
- **Removes take `filepath.Base(Filename)`**, so a crafted `../../x` cannot
  delete outside the collection — verified by a test that plants a file one level
  up and asserts it survives.
- **Reads are capped at 4 MiB.** Not a real limit on legitimate input, but a
  guard against a symlink resolving to something enormous.

## `.env` support

Bruno watches `.env` and `.env.*` at the collection root
(`dotenv-watcher.js:7-9`), feeds the parsed map into `process.env`, and also
shows the variables in the UI. Only a plain `.env` populates `process.env`.

Grammar supported: `#` comments (start of line, or after whitespace inside an
unquoted value), `export` prefix, double quotes with `\n \r \t \\ \"`, single
quotes literal, unquoted values trimmed, trailing-backslash continuations,
CRLF, last assignment wins.

`MergeInto` gives an exported shell variable precedence over the file. The file
supplies what the shell did not; a CI job that exports a variable expects its
value.

`.env.*` variants are recognised by `IsDotEnvFile` for the UI, but not merged —
matching Bruno.

## Verification

```
go vet ./internal/core/environment/          # clean
go test -race -cover ./internal/core/environment/
ok  ...  coverage: 91.2% of statements
```

Coverage of the acceptance criterion (the precedence table from
`flows/environment-resolution.md`):

- Chain order: nearest ancestor wins over the root; target's own values are not
  in `Inherited`.
- Child shadows parent → the inherited entry is removed entirely.
- Disabled ancestor variables are skipped.
- Provenance records the ancestor's name and UID.
- Two-node, three-node and self-referential cycles rejected, with a path that
  begins and ends with the same name.
- Missing parent rejected with both names in the error.
- Secrets stay out of `Values()`; `SecretNames()` deduplicates across ancestors
  and the target's own `vars:secret` list.
- `Inheritable` excludes self and all descendants, keeps unrelated environments.
- `Select` by UID; empty or unknown UID yields nil rather than a wrong match.
- Duplicate environment names resolve to the first definition.
- Loader: both dialects, extension filtering, filename fallback, sorting,
  missing directory, unknown codec, malformed file, save round-trip.
- Path safety: sanitised filename stays in `environments/`; a traversal
  `Filename` cannot delete outside it; missing file removal is idempotent.
- Dotenv: full grammar table, last-wins, continuation, CRLF, four malformed
  cases, lenient unknown escape, missing file, merge precedence.

## Not covered here (deliberately)

- **Interpolation between scopes.** That is `vars.Scopes`, already delivered;
  this package produces the environment map it consumes.
- **The secret store.** Values live outside the collection (Bruno encrypts them
  in `electron-store`). `SecretNames()` says which names need values; retrieval
  belongs to the settings/secret layer.
- **File watching.** Bruno uses chokidar and pushes `main:dotenv-file-update`.
  Watching belongs to the platform layer; `Loader` is stateless and can be
  re-run on a change event.
- **Global and workspace environments.** Both are stored in app preferences
  rather than the collection, so they belong to the settings/workspace layer.
  They feed the same `vars.Scopes.Global` slot.

## Sources

- `bruno-common/src/utils/environment-inheritance.ts` —
  `getInheritedEnvironments`, `resolveEnvironmentInheritance`,
  `getInheritableEnvironments`, `validatedEnvironmentExtendsFrom`
- `bruno-electron/src/app/dotenv-watcher.js` — `.env` detection, parsing, update events
- `bruno-electron/src/store/process-env.js` — per-collection `process.env` slots
- `flows/environment-resolution.md` §2 (precedence), §4 (`extends`), §7 (algorithm)
