# Product

<!-- impeccable:product-schema 1 -->

## Platform

adaptive

Desktop application: window chrome is a custom title bar drawn by the toolkit, with native window controls, native menus and native file dialogs. One design language across macOS, Linux and Windows rather than per-OS restyling — the `adaptive` label records that the chrome must respect each OS's window-control insets and menu conventions.

## Stack

Recorded, not chosen: the repository already answers this.

- **Language:** Go 1.27, entire application, GUI included (`go.mod`).
- **GUI toolkit:** **MyGo** (`github.com/egoist/mygo` v0.8.4) for layout, painting and the frame.
- **Component layer:** **Lobe MyGo** (`github.com/lobehub/lobe-mygo`) for primitives — buttons, selects, tabs, modals, toasts, draggable side nav.
- **Hard constraint from the docs:** no React/Vue/Svelte, no HTML/CSS, no Electron, no Wails. JavaScript appears only as the compatibility script runtime (`goja`) for `script:*`, `tests {}`, `assert {}` and PAC.

## Users

API developers and QA engineers who already use Postman, Insomnia or Bruno, or who want to leave one of them (STATED BY USER). They work on a desktop, often against private and local services, and usually have a collection checked into a git repository next to the code it tests.

The job is the everyday loop of API work: open a collection from disk, find a request, change it, send it, read the response, check the assertion results, switch environment, and commit the resulting `.bru` file as a normal code change.

## Product Purpose

An independent, Go-native HTTP API client that is compatible with Bruno's on-disk formats (`.bru`, collections, environments) and Brunos' scripting API. Success means a Bruno user can point fearless at an existing collection and keep working, and a Go team can ship and audit an API client they build themselves without a Node/Electron toolchain.

## Positioning

**The format is the product.** Unlike a hosted or database-backed API client, the collection on disk is the source of truth: plain text, human-readable, git-diffable, in a folder that belongs to the user's repository. Fearless inherits that contract and makes it run as a single native binary with no Node runtime.

## Operating Context

- Collections are git repositories or subdirectories of one. Diffs are reviewed; a save that rewrites a file wholesale is a visible cost.
- Several collections can be mounted at once (workspaces).
- Environment files carry secrets; some values are deliberately kept out of the repo.
- Requests are chained through list-style runners and scripts; responses are inspected in a body/headers/timeline/tests view.
- The reference product is Bruno at commit `60d7517d0545ef82c032c10dcbb6a3c815d246cf`; the docs under `docs/` are the specification, and the parity matrix in `docs/implementation/feature-parity-matrix.md` defines scope.

## Capabilities and Constraints

Deliberately deferred from v1 (`docs/research/unresolved-questions.md` §UQ-D6, and the roadmap): gRPC, WebSocket, a mock server, AI assistance, an embedded terminal, OpenAPI sync. The UI must therefore not be designed around them, though it should not be built in a way that forbids them later.

Open product decisions recorded rather than resolved: MyGo binding depth (UQ-D1), script compatibility mode (UQ-D5), `.bru` writer fidelity — byte-exact vs semantic round-trip (UQ-D4), and history retention (UQ-D3). These change what the UI can promise about save and history.

The UI layers are implemented and unit-tested (presenters, widgets, screens, theme, layout, app assembly); the application has not been through a visual design pass.

## Evidence on Hand

- Specification: `docs/` — architecture, flows, UI, logic, plugins, data formats, implementation plan, research, all evidence-labelled.
- Reference implementation to compare against: the `bruno/` clone at the pinned commit.
- Rendered evidence: golden captures in `internal/ui/screens/shell/testdata/` and `internal/ui/screens/navigation/testdata/` (placeholder fixtures only — they prove layout, not design).
- No brand assets exist: no logo, no wordmark, no marketing site, no user testimonials or usage evidence. Design work must not invent any.

## Product Principles

1. **The file is the contract.** Anything that makes a collection harder to read, diff or hand-edit in git is a regression, however convenient it is in the editor.
2. **The request loop is the product.** Open, find, edit, send, read, assert — the path between selecting a request and understanding a response outranks every secondary feature.
3. **Compatibility is a user-visible promise.** A Bruno user must recognise their own collection, environment and tests, and not be surprised by a different interpretation of them.
4. **Native, not a web app in a costume.** The interface should use the toolkit's real affordances — real splitters, native menus, real focus and keyboard routing — instead of imitating DOM habits.
5. **Facts beat decoration.** Status, timing, sizes, failures and dirty state are the interface's most important content and must never be obscured by chrome.

## Accessibility & Inclusion

No product-specific standard has been established by the project yet. Bruno's observed baseline (docs/ui/design-system.md §5) is a strong point that the design must at least preserve: every interactive element has hover/focus/disabled states, focus is always visible, modals trap and restore focus, and tab strips never overflow off-screen. Light and dark must both be fully legible; current dark-mode rendering is known to be broken.
