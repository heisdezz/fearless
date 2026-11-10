Update the existing documentation to enforce a strict Go-only application architecture.

IMPORTANT:

- Do not rewrite, regenerate or restructure the existing documentation.
- Do not modify unrelated documentation.
- Preserve all existing analysis, source references, diagrams, findings and conclusions.
- Only make targeted additions or corrections where necessary.

Requirements:

1. The entire application, including its GUI, must be written in Go.
2. No React, Vue, Svelte, HTML/CSS frontend, Electron, Wails or separate JavaScript frontend.
3. JavaScript is permitted only for embedded scripting compatibility with Bruno.
4. Add a GUI framework evaluation covering Shirei, Gio, Fyne and other relevant native Go GUI frameworks.
5. The HTTP engine, application services, state management, collections and persistence must remain independent of the GUI framework.
6. Add a proposed Go-native presentation architecture using interfaces and typed events.
7. Update the implementation roadmap and dependency recommendations to reflect the Go-only requirement.

Modify only the relevant existing files:

- `docs/implementation/go-architecture.md`
- `docs/implementation/package-structure.md`
- `docs/implementation/dependency-recommendations.md`
- `docs/ui/design-system.md`
- `docs/ui/application-layout.md`
- `docs/architecture/system-overview.md`

If a relevant document does not exist, create it only when necessary.

If existing documentation recommends Wails or another web-based GUI, replace only that recommendation and its directly affected explanations.

Do not duplicate existing content. Integrate new information into the appropriate sections.

Finally, provide a concise report listing:

- Files modified.
- Sections added or corrected.
- Any architectural conflicts discovered.
- Any remaining decisions requiring my input.
