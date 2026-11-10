# Task: Complete Bruno Architecture & Implementation Analysis

You are acting as a senior software architect, reverse engineer, UI/UX analyst and Go systems engineer.

Your task is to thoroughly analyze the entire Bruno source code repository and produce a comprehensive set of Markdown documentation files that will serve as the technical blueprint for building an independent, Go-native HTTP API client inspired by Bruno.

**IMPORTANT:** This phase is strictly for research, analysis and documentation. Do not implement the application yet.

## 1. Repository exploration

Read and analyze the entire relevant source code.

Do not rely exclusively on:

- README files
- Documentation
- Package descriptions
- Existing architecture diagrams

Inspect the actual implementation, including:

- Source files
- Components
- Hooks
- State management
- Services
- Utilities
- HTTP execution logic
- Plugins and extensions
- Storage mechanisms
- Configuration
- Tests
- Build scripts
- Dependencies

Follow function calls and data flow across modules rather than documenting files in isolation.

Identify the important architectural boundaries and dependencies.

## 2. Documentation deliverables

Create a `/docs` directory with the following structure:

```text
docs/
├── README.md
├── architecture/
│   ├── system-overview.md
│   ├── module-breakdown.md
│   ├── dependency-map.md
│   ├── application-lifecycle.md
│   └── design-decisions.md
│
├── flows/
│   ├── application-startup.md
│   ├── request-lifecycle.md
│   ├── response-lifecycle.md
│   ├── collection-lifecycle.md
│   ├── environment-resolution.md
│   ├── authentication-flow.md
│   ├── scripting-flow.md
│   ├── history-flow.md
│   └── error-handling.md
│
├── ui/
│   ├── design-system.md
│   ├── application-layout.md
│   ├── navigation.md
│   ├── components.md
│   ├── request-editor.md
│   ├── response-viewer.md
│   ├── collections-sidebar.md
│   ├── environment-selector.md
│   ├── settings.md
│   └── interaction-patterns.md
│
├── logic/
│   ├── http-engine.md
│   ├── request-builder.md
│   ├── response-parser.md
│   ├── variable-interpolation.md
│   ├── authentication.md
│   ├── cookies.md
│   ├── redirects.md
│   ├── tls-and-certificates.md
│   ├── scripting.md
│   ├── assertions-and-tests.md
│   ├── import-export.md
│   └── error-management.md
│
├── plugins/
│   ├── extension-architecture.md
│   ├── plugin-system.md
│   ├── scripting-runtime.md
│   ├── authentication-providers.md
│   └── extensibility-proposal.md
│
├── data/
│   ├── collection-format.md
│   ├── bru-format.md
│   ├── environment-format.md
│   ├── request-schema.md
│   ├── response-schema.md
│   └── persistence.md
│
├── implementation/
│   ├── go-architecture.md
│   ├── package-structure.md
│   ├── interface-design.md
│   ├── dependency-recommendations.md
│   ├── feature-parity-matrix.md
│   └── implementation-roadmap.md
│
└── research/
    ├── source-map.md
    ├── dependencies.md
    ├── test-coverage.md
    ├── limitations.md
    └── unresolved-questions.md
```

## 3. Analysis requirements

### Architecture

Document:

- Application entry points.
- Major modules and their responsibilities.
- Module dependencies.
- Communication between modules.
- State management.
- Internal APIs.
- Event systems.
- Application initialization and shutdown.

Include Mermaid architecture diagrams.

### Flows

For every major feature, document its complete lifecycle.

For example, the HTTP request lifecycle should cover:

1. User creates a request.
2. Request configuration is collected.
3. Environment variables are resolved.
4. Authentication is applied.
5. Headers and body are constructed.
6. HTTP request is executed.
7. Redirects and cookies are handled.
8. Response is received.
9. Response data is processed.
10. Response is displayed.
11. History is updated.

Include Mermaid sequence diagrams and flowcharts.

Document success paths, error paths, cancellation and edge cases.

### UI

Inspect the actual UI implementation.

Document:

- Overall layout.
- Sidebar structure.
- Tabs.
- Request editor.
- HTTP method selector.
- URL input.
- Query parameters.
- Headers editor.
- Authentication editor.
- Body editor.
- Scripts editor.
- Response viewer.
- Response tabs.
- Status indicators.
- Context menus.
- Keyboard shortcuts.
- Dialogs and modals.
- Settings.

For every major component, document:

- Purpose.
- Layout.
- Properties.
- State.
- Events.
- User interactions.
- Dependencies.
- Behaviour under different states.

Include Mermaid component hierarchies where useful.

### Logic

Analyze the actual implementation, not just the expected behaviour.

Document algorithms, input/output structures, validation, execution order, edge cases and error handling.

Pay particular attention to:

- HTTP request construction.
- Authentication.
- Variable interpolation.
- Request chaining.
- Cookie management.
- Redirect handling.
- Multipart uploads.
- File uploads.
- Streaming responses.
- TLS configuration.
- Proxy configuration.
- JavaScript execution.
- Tests and assertions.

### Plugins and extensions

Investigate whether Bruno uses a formal plugin architecture or relies on built-in integrations and extension mechanisms.

Do not invent a plugin system if one does not exist.

Document:

- Existing extension points.
- Authentication integrations.
- Scripting APIs.
- External dependencies.
- Runtime boundaries.
- How new integrations can be introduced.

Then propose a suitable Go-native extensibility architecture.

### Data formats

Document the actual on-disk formats.

Pay particular attention to:

- `.bru` files.
- Collection configuration.
- Folder organization.
- Environment files.
- Variables.
- Authentication configuration.
- Request metadata.

Include real examples based on source code and explain how they should be parsed and represented in Go.

## 4. Go-native implementation proposal

The target application will be written entirely in Go, including its GUI.

Do not assume that the original application's framework or programming language must be replicated.

Propose:

- Go package boundaries.
- Interfaces.
- Core data structures.
- HTTP transport architecture.
- State management.
- GUI integration.
- Scripting runtime.
- Persistence.
- Plugin architecture.
- Concurrency model.
- Event communication.

Prefer standard Go libraries where appropriate.

Consider GUI frameworks such as:

- Shirei
- Fyne
- Gio
- Wails (only if a web-based frontend is justified)

Evaluate their suitability rather than choosing one arbitrarily.

The HTTP engine must remain independent of the GUI.

## 5. Source references and accuracy

Every significant technical claim must reference its original source.

Use this format:

```text
Source: path/to/file.go
Relevant symbols: FunctionName, TypeName
```

Where possible, include line numbers.

Clearly distinguish:

- VERIFIED: directly observed in source code.
- INFERRED: conclusion derived from multiple source files.
- PROPOSED: your recommended implementation.
- UNKNOWN: could not be established.

Never fabricate functionality, APIs, dependencies or architectural decisions.

If something cannot be established, document it in `research/unresolved-questions.md`.

## 6. Feature parity

Produce a comprehensive feature parity matrix.

For every feature, identify:

- Feature name.
- How Bruno implements it.
- Relevant source files.
- Required Go implementation.
- Complexity (low, medium, high).
- Dependencies.
- Priority.
- Whether compatibility with Bruno is possible.

Separate essential HTTP client functionality from optional conveniences.

## 7. Implementation roadmap

Create a phased implementation roadmap:

Phase 1: Core architecture and HTTP engine.

Phase 2: Request editor and response viewer.

Phase 3: Collections and persistence.

Phase 4: Authentication and environments.

Phase 5: Scripting and assertions.

Phase 6: Import/export and Bruno compatibility.

Phase 7: Advanced functionality and extensibility.

For each phase, define deliverables, dependencies, acceptance criteria and testing requirements.

## 8. Final quality checks

Before finishing:

1. Verify that all documentation files exist.
2. Check that important modules have not been overlooked.
3. Ensure diagrams match the actual implementation.
4. Verify source references.
5. Identify contradictions between documentation and implementation.
6. Identify incomplete analysis.
7. Ensure the Go proposal is independent of Bruno's original architecture.
8. Update `docs/README.md` with a complete documentation index.

Do not stop after inspecting a few representative files.

Explore the repository systematically. If the repository is too large to analyze in one pass, divide the work into modules and maintain a research checklist until every relevant module has been covered.

**Final deliverable:** A complete, source-grounded technical documentation package that another Go engineer can use to implement the application without repeatedly needing to rediscover Bruno's architecture.
