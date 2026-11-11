# Screen — workspace switcher, create, import

Mode: **Operate**. The workspace dropdown in the title bar and its dialogs.

Sources: `components/AppTitleBar/index.js` (dropdown), `components/ManageWorkspace/`,
`components/WorkspaceSidebar/CreateWorkspace`, `WorkspaceSidebar/ImportWorkspace`,
`components/WorkspaceHome/` (overview, docs, environments). Screenshot:
`refs/workspace_switcher_bruno.png` (note: this capture shows the switcher, not
environments).

---

## Workspace dropdown (title bar)

Trigger: workspace name + chevron in the title bar (see `app-shell.md`).
Menu (`MenuDropdown`, `AppTitleBar/index.js`): each workspace row has
name + pin toggle; actions to create and import a workspace
(`CreateWorkspace`, `ImportWorkspace`). Rows use `padding 4px 10px`
(`AppTitleBar/StyledWrapper.js:105-110`).

## Workspace home (no workspace/collection open)

`WorkspaceHome` = tabs/pages: `WorkspaceOverview`, `WorkspaceDocs`,
`WorkspaceEnvironments`. Observed tabs: **`Collections`**,
**`Environments`**, **`Quick Actions`** (`WorkspaceHome` labels, `:146-155`).

Empty state (verbatim, `WorkspaceHome`):

- Title **`No collections yet`**
- Body **`Create your first collection or open an existing one to get started.`**

Environments pane empty states: **`No Environments`**, **`No environments`**,
**`No .env files`**.

Unsaved-changes guard when leaving an environment: **`Hold on..`** /
**`You have unsaved changes in this environment.`**

## Create workspace

Modal from `WorkspaceSidebar/CreateWorkspace`. Name field, confirm. New
workspaces are created with a unique name (`createWorkspaceWithUniqueName`,
`AppTitleBar/index.js:11`); a nameless workspace displays **"Untitled
Workspace"** (`AppTitleBar/index.js:31-34`).

## Import workspace

Modal from `WorkspaceSidebar/ImportWorkspace`; file picker labelled
**`Workspace File`** (`.bru`-workspace format). See `screens/import-export.md`
for the collection import flow (different from workspace import).

## Git actions on a workspace collection row

Context menu copy observed (`WorkspaceHome`): **`Rename`**, **`Share`**,
**`Copy Git URL`**, **`Connect to Git`**, **`Remove Git Remote`**,
**`Remove`**, **`Delete`**, plus states **`Failed to open`**, **`Not cloned`**.

## Open Questions

- Create/Import workspace dialogs have no verbatim field labels captured beyond
  `Workspace File`; read those two components fully when building them.
