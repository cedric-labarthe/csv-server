# csv-server - Project Instructions

## Stack

- **Language**: Go 1.26+
- **HTML Generation**: [Templ](https://templ.guide/) - type-safe HTML templates compiled to Go
- **UI Components**: [templUI](https://templui.io) v1.6.0 — installed via CLI, components live in `components/`
- **CSS**: Tailwind CSS (CDN) + custom layout styles in `web/static/style.css`
- **Interactivity**: Alpine.js (`web/static/alpine.min.js`, embedded)
- **HTTP**: `net/http` standard library with Go 1.22+ enhanced routing
- **Static assets**: `go:embed` + `http.FileServerFS`

## Core Principles

### Standard Library First

Maximize use of Go's standard library. Before adding an external dependency, verify there is no stdlib alternative:

| Need                | Use                           | Do NOT use            |
| ------------------- | ----------------------------- | --------------------- |
| HTTP server         | `net/http`                    | gin, echo, fiber, chi |
| CSV parsing         | `encoding/csv`                | any external package  |
| Logging             | `log/slog`                    | zap, logrus, zerolog  |
| JSON                | `encoding/json`               | jsoniter, sonic       |
| Config from env     | `os.Getenv` / `os.LookupEnv`  | viper, godotenv       |
| Static file serving | `http.FileServerFS` + `embed` | any external package  |
| Filesystem          | `os`, `io/fs`, `path/filepath` | -                    |

External Go dependencies currently allowed:
- `github.com/a-h/templ` — HTML templating
- `github.com/Oudwins/tailwind-merge-go` — pulled in by templUI utils

### Modern Go Practices (1.22+)

- Use **enhanced `net/http` routing** with method + path: `mux.HandleFunc("GET /view/{path...}", h.view)`
- Use **`log/slog`** for structured logging, never `log.Printf`
- Use **`iter`** package and range-over-func where appropriate (Go 1.23+)
- Use **`errors.Join`** instead of manual error wrapping chains
- Use **named return values** only when it meaningfully clarifies intent
- Prefer **`any`** over `interface{}`
- Avoid `init()` functions — configure explicitly in `main`

### Code Style

- No `var` blocks at package level — use `const` or initialize in functions
- Early returns, no deeply nested `if` blocks
- Functions do one thing — keep them small and testable
- Handler functions receive `(w http.ResponseWriter, r *http.Request)` — no global state
- Errors are always handled — never `_ = err` silently
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- **Avoid anonymous parameters that are hard to read at the call site** — booleans especially: `entryItem("foo", "/browse/foo", true)` tells you nothing. When a Go function has multiple parameters or any boolean, use a named struct instead: `doSomething(doSomethingParams{name: "foo", href: "/browse/foo", isDir: true})`
- **Templ components always take a single `Props` struct** — even for one or two fields, for consistency: `templ myComponent(props myComponentProps)`. Never use positional parameters in templ components.

### Project Layout

```
csv-server/
├── cmd/server/main.go        # Entry point — wiring only, no business logic
├── components/               # templUI components (copied via `templui add <name>`)
│   ├── button/button.templ   # Button component
│   └── icon/                 # Lucide icon set (icon.go, icon_defs.go, icon_data.go)
├── utils/templui.go          # templUI utilities (TwMerge, If…)
├── internal/
│   ├── config/config.go      # Config struct loaded from environment
│   ├── csv/reader.go         # CSV parsing logic
│   └── handler/handler.go    # HTTP handlers
├── templates/
│   ├── pages/                # package pages — one .templ per page
│   │   ├── index.templ       # Directory listing page
│   │   ├── table.templ       # CSV table page
│   │   └── table_helpers.go  # truncate, parentURL (table-specific)
│   └── components/           # package components — one .templ per component
│       ├── layout.templ      # Base HTML shell (<link>, <script> only)
│       ├── breadcrumb.templ  # Breadcrumb component
│       ├── breadcrumb_helpers.go  # BreadcrumbTitle, JoinPath
│       ├── copy_button.templ # CopyButton (templUI button + icon + Alpine)
│       └── entry_item.templ  # File/dir list item
├── web/
│   ├── web.go                # go:embed declaration
│   └── static/
│       ├── style.css         # Layout & custom styles (non-Tailwind)
│       ├── app.js            # Global JS (keep minimal — prefer Alpine for component logic)
│       └── alpine.min.js     # Alpine.js v3 (embedded, no CDN at runtime)
├── data/                     # Default CSV file directory (dev only)
└── go.mod
```

### Templ — Component Practices

- **One `templ` declaration per file** — each `.templ` file contains exactly one `templ` component
- **No inline CSS or JS** in `.templ` files — `layout.templ` only contains `<link>` and `<script src>` tags
- **All custom CSS** lives in `web/static/style.css` — use Tailwind utilities for component-level styling
- **Component-scoped interactivity** → Alpine.js `x-data`/`x-on`/`x-show`/`x-bind` directly in the template
- **Global interactivity** → `web/static/app.js` (keep minimal)
- Never edit `*_templ.go` files by hand — they are regenerated

### templUI Components

Components are **copied into your project** (not imported from an external package) via:

```bash
templui add <component-name>    # e.g. templui add button icon badge table
```

After adding, import from your own module:

```go
import (
    "csv-server/components/button"
    "csv-server/components/icon"
)

templ MyComponent() {
    @button.Button(button.Props{
        Size:    button.SizeIcon,
        Variant: button.VariantGhost,
    }) {
        @icon.Copy(icon.Props{Size: 16})
    }
}
```

Available icon names come from the Lucide set — browse `components/icon/icon_defs.go`.

**Always use templUI components** instead of raw HTML elements — never write a bare `<button>`, `<a>`, `<input>`, etc. when a templUI component exists for it. Use `button.Button`, `badge.Badge`, `card.Card`, etc.

### Static Assets

Custom styles and JS are embedded at build time via `go:embed` in `web/web.go` and served by `http.FileServerFS`. Tailwind CSS is loaded from CDN (acceptable for an internal tool — use the standalone Tailwind CLI for production builds).

### Commands

```bash
# After editing .templ files:
templ generate ./templates/... ./components/...

go build ./cmd/server    # Build
go run ./cmd/server      # Run (dev)
go test ./...            # Tests
go vet ./...             # Lint

# Adding a new templUI component:
templui add <name>
```

### Environment Variables

| Variable   | Default  | Description                    |
| ---------- | -------- | ------------------------------ |
| `PORT`     | `8080`   | HTTP listen port               |
| `DATA_DIR` | `./data` | Directory containing CSV files |
