Update the existing documentation to incorporate Lobe MyGO as the chosen UI component library alongside the already-established MyGO GUI framework.

IMPORTANT:

- Do not rewrite or regenerate existing documentation.
- Do not modify the existing MyGO architecture.
- Do not modify the existing Goja or scripting architecture.
- Do not introduce alternative GUI frameworks.
- Only make targeted additions to relevant existing documentation.

### Lobe MyGO analysis

Inspect the actual Lobe MyGO repository, source code and documentation.

Document:

1. Available UI components and their APIs.
2. Component composition and customization.
3. Theme system, colors and design tokens.
4. Typography, icons, spacing and layout.
5. Form controls and input components.
6. Dialogs, popovers, menus and tooltips.
7. Tables, lists and tree components.
8. Component state and event handling.
9. Integration with MyGO.
10. Reusable component and extension patterns.

### Application-specific UI mapping

Identify how Lobe MyGO components can be used to implement the HTTP client's interface:

- Application navigation and sidebar.
- Collection explorer and folder tree.
- Request tabs.
- HTTP method selector.
- URL input.
- Query parameter editor.
- Headers editor.
- Authentication editor.
- Request body editor.
- Response viewer.
- JSON/XML presentation.
- Environment selector.
- Settings and dialogs.

Clearly distinguish existing library components from components that must be implemented specifically for this application.

### Documentation updates

Integrate the findings into the relevant existing files:

- `docs/ui/design-system.md`
- `docs/ui/components.md`
- `docs/ui/application-layout.md`
- `docs/ui/interaction-patterns.md`
- `docs/implementation/dependency-recommendations.md`

Create no additional documents unless genuinely necessary.

Reference actual APIs and source files. Do not invent functionality.

Finally, report the files modified and any Lobe MyGO capabilities that require custom implementation.
