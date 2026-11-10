# Environment Format

Status: VERIFIED unless marked.

---

## 1. Locations

| Kind | Location | Managed by |
|---|---|---|
| Collection environment | `<collection>/environments/<Name>.bru` \| `.yml` | Collection (git-tracked) |
| Workspace environment | Workspace config (`store/workspace-environments.js`) | Workspace |
| Global environment | `electron-store` file `global-environments` | App |
| Collection `.env` file | `<collection>/.env` | → `process.env.*` |

---

## 2. Model (canonical)

```ts
interface Environment {
  uid: string;
  name: string;
  color?: string | null;
  extends?: string | string[] | null;   // parent environment name(s)
  variables: Array<{
    uid: string;
    name: string;
    value: string | number | boolean | object | null;
    dataType?: 'string' | 'number' | 'boolean' | 'object';
    enabled: boolean;
    secret: boolean;
    description?: string | null;
    inheritedFrom?: { name: string; uid: string };  // computed at resolve time
  }>;
  externalsecrets?: Record<string, Record<string, unknown>>;  // e.g. { vault: { path } }
}
```

Sources: `bruno-schema-types/src/collection/environment.ts`, `bruno-lang/v2/src/envToJson.js`, `bruno-common/src/utils/environment-inheritance.ts`.

---

## 3. `.bru` representation

```bru
vars {
  host: http://localhost:8080
  @number
  retries: 3
  @boolean
  debug: true
  @object
  headers: '''
    {"x-tenant":"acme"}
  '''
  ~disabledVar: nope
}

vars:secret [
  clientSecret,
  clientId
]

vars:externalsecrets:vault {
  path: secret/data/bruno
}

extends: Base

color: #3b82f6
```

| Element | Rule |
|---|---|
| `vars` dictionary | `name: value` pairs; `~` disables; annotations set type |
| `'''…'''` | Multiline value, 4-space de-indent |
| `vars:secret [ … ]` | Names only — **no values on disk** |
| `vars:externalsecrets:<provider>` | Provider config dictionary |
| `extends: Name` or `extends [ A, B ]` | Top-level rule; list requires commas; quotes allowed |
| `color: #rrggbb` | Top-level rule |

---

## 4. YAML representation

```yaml
name: Test
variables:
  - name: envVar
    value: original
    enabled: true
    secret: false
```

Verified fixture: `packages/bruno-tests/workspaces/onfail/onfail-collection/environments/Test.yml`.
Global environment YAML (`workspaces/<ws>/environments/Global.yml`) uses the same shape with `name: Global`.

---

## 5. Secrets

| Aspect | Behavior |
|---|---|
| Storage | `EnvironmentSecretsStore` (`store/env-secrets.js`), encrypted |
| Key | (collection path / env name / variable name) |
| File | Only the name appears in `vars:secret` |
| Read | Watcher substitutes the decrypted value into the variable list |
| Write | Saving an env encrypts and stores secret values; deletes removed ones |
| Failure | `decryptStringSafe` returns a placeholder rather than crashing |
| Protection | `vars:secret` values are never interpolated from `.env`; they are plain stored values |

Global-environment secrets use `encryptStringSafe` in `GlobalEnvironmentsStore`.

---

## 6. External secrets

```bru
vars:externalsecrets:vault {
  path: secret/data/bruno
}
```

Resolved through `node-vault` (`bruno-requests/src/utils/node-vault.ts`). Requires Vault env config (`VAULT_ADDR`, `VAULT_TOKEN`). Only Vault is implemented (VERIFIED by `node-vault.ts` presence and provider grammar).

---

## 7. Typed values

`@number` / `@boolean` / `@object` annotations → `dataType` → `parseValueByDataType` coercion. Without an annotation, values are strings. The object form requires valid JSON inside a multiline block.

---

## 8. Inheritance

- `extends` points one or more parents by **name**.
- Resolution flattens parents (in order), then applies the child; a child variable overrides a parent variable of the same name; disabled variables are skipped.
- Cycles and missing parents are reported (`cyclicInheritancePath`, `missingInheritedEnvironmentName`).
- Inheritance is resolved **before** sending (`resolveEnvironmentInheritance` in the renderer) — the transport always receives a flat environment.

---

## 9. `.env` files (process.env)

Sourced from three places with priority:

```text
collection .env  >  workspace .env  >  OS environment
```

Available only under `{{process.env.NAME}}` (nested object in the interpolation scope). Variables named `process` in a scope would be shadowed by the injected object.

---

## 10. Import/export

`ImportEnvironmentModal` accepts:
- `.bru`/`.yml` env files
- `.env` files
- Postman environment JSON (`postman-env-to-bruno-env.js`)
- Insomnia environment exports

---

## 11. Go implementation (PROPOSED)

```go
type Variable struct {
    Name        string
    Value       any
    DataType    DataType   // string|number|boolean|object
    Enabled     bool
    Secret      bool
    Description string
}

type Environment struct {
    UID       string
    Name      string
    Color     string
    Extends   []string
    Variables []Variable
    External  map[string]map[string]any
}

type Secrets interface {
    Get(collection, env, name string) (string, error)
    Put(collection, env, name, value string) error
    Delete(collection, env, name string) error
}

func Resolve(envs []*Environment, name string) (*Environment, []string /*warnings*/, error)
```

Match on: name-based `extends`, parent-then-child override, secret values never serialized, typed coercion.
