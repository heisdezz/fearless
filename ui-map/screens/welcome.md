# Screen — welcome / onboarding

Mode: **Experience** (first run). Shown once, on first launch.

Sources: `components/WelcomeModal/` (`index.js`, `WelcomeStep/`, `ThemeStep/`,
`StorageStep/`, `GetStartedStep/`, `StyledWrapper.js`), `components/EmptyAppState/`,
`components/WorkspaceHome/`. Test hooks: `data-testid="welcome-modal"`.

---

## Layout tree

```text
        ┌──────────────────────── welcome-card ────────────────────────┐
        │                     [ Bruno logo 48 ]                        │
        │                 Welcome to Bruno  (step 1)                   │
        │        A fast, Git-friendly, and open-source API client.     │
        │                                                              │
        │                     <step body>                              │
        │                                                              │
        │  ● ─ ● ─ ○ ─ ○            Skip   Back   Get Started/Next     │
        └──────────────────────────────────────────────────────────────┘
                 (dimmed backdrop over the empty app shell)
```

## Steps (4 total)

Heading by step (`index.js:108-112`): step 1 **"Welcome to Bruno"**, step 4
**"Ready to go!"**, steps 2–3 **"Set up Bruno"**. Tagline (step 1 only):
**"A fast, Git-friendly, and open-source API client."**

| # | Step | Label / Title | Content |
| --- | --- | --- | --- |
| 1 | Welcome | — | Four highlight cards (`WelcomeStep/index.js:9-28`) |
| 2 | Theme | `Appearance` / `Choose your theme` | mode buttons + variant grids (`ThemeStep/index.js:40-42`) |
| 3 | Storage | `Storage` / `Where should we store your collections?` | path picker + `Browse` (`StorageStep/index.js:6-30`) |
| 4 | Get started | `Your first collection` / `You're all set! What's next?` | action cards (`GetStartedStep/index.js:7-47`) |

### Step 1 — highlights (verbatim, `WelcomeStep/index.js:9-28`)

| Icon (Tabler) | Title | Description |
| --- | --- | --- |
| `IconFolder` | Filesystem only | Collections are plain files on your disk. No cloud sync, no proprietary lock-in. |
| `IconGitFork` | Git-friendly | Every request is a readable file. Commit, branch, review, and collaborate using the tools you already know. |
| `IconLock` | Privacy-focused | No account, no login. Bruno works entirely offline, your API keys never leave your machine. |
| `IconRocket` | Fast and lightweight | Built to be snappy. No bloated runtimes, just a fast, focused tool for exploring and testing APIs. |

Icon size 18, stroke 1.5.

### Step 2 — theme

Description: **"Pick a look that feels right. You can always change this later
in Preferences."** Mode buttons (light / dark / system, `themeModes`). For each
selected appearance a grid of variant cards with a `ThemePreviewBox` and the
variant name; `aria-pressed` reflects selection (`ThemeStep/index.js:52-90`).
Variant list comes from `getLightThemes()` / `getDarkThemes()`
(`themes/index.js`) — the same ~21 presets listed in
`design-system.md` §9.

### Step 3 — storage

Folder picker: placeholder text **"Click to choose a folder..."**, button
**"Browse"** (`StorageStep/index.js:28-30`).

### Step 4 — get started

| Card | Title | Description |
| --- | --- | --- |
| primary | Create Collection | Start fresh with a new API collection |
| primary | Import Collection | (import an existing collection) |
| secondary | Open existing collection | Open a Bruno collection from your filesystem |
| secondary | Get started with a request | Jump right in with a new HTTP request |

Source `GetStartedStep/index.js:7-47`.

## Footer

- Progress dots (one per step, `aria-label="Go to step N"`,
  `aria-current="step"`) — clicking jumps to that step (`index.js:127-135`).
- Buttons (`index.js:137-158`): **`Skip`**, **`Back`** (steps > 1),
  **`Get Started`** (step 1) / **`Next`** (steps 2–3), and on the last step
  **`I'll explore on my own`**.

## Empty app state (post-onboarding, no collection)

`EmptyAppState` / `WorkspaceHome` render in the main column when no request is
open. Copy is component-local; see `screens/sidebar.md` for the sidebar's own
**"No requests found"** empty state. Recorded here so the post-onboarding
screen is not left blank.

## Interactions

- Entire flow is keyboard navigable; dots and cards are real `<button>`s.
- `Skip` and `I'll explore on my own` dismiss and persist
  (`handleSaveAndDismiss`, `index.js:52-56`).

## Open Questions

- `EmptyAppState` and `WorkspaceHome` copy was not exhaustively read this pass;
  capture a screenshot with an empty workspace to finish it.
