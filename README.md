# csv-server

A minimal Go HTTP server that lets you browse a directory tree and open CSV files rendered as HTML tables.

## Requirements

- Go 1.26+
- [Templ](https://templ.guide/) (only needed when editing `.templ` files)

## Getting started

```bash
go run ./cmd/server
```

The server listens on `:8080` by default and serves files from `./data`.

## Configuration

| Variable   | Default    | Description                    |
| ---------- | ---------- | ------------------------------ |
| `HOST`     | `0.0.0.0`  | HTTP listen address            |
| `PORT`     | `8080`     | HTTP listen port               |
| `DATA_DIR` | `./data`   | Directory containing CSV files |

```bash
DATA_DIR=/path/to/csvs PORT=9000 go run ./cmd/server
```

## Routes

| Route               | Description                              |
| ------------------- | ---------------------------------------- |
| `GET /`             | Root directory listing                   |
| `GET /browse/{path}`| Browse a subdirectory                    |
| `GET /view/{path}`  | View a CSV file as a table               |
| `GET /health`       | Health check — returns `200 OK` / `ok`   |


## Docker (development)

Build the image (optional, `docker compose up --build` also builds it):

```bash
docker build -t csv-server-dev .
```

Start the development container with source bind-mounted and port `8080` published (the app is launched by the Dockerfile default command):

```bash
docker compose up --build
```

Run one-off commands in the container (full Go toolchain available).  
Use these when you don't want to keep `docker compose up` running:

```bash
# Run the app once (alternative to `docker compose up`)
docker compose run --rm --service-ports app

# Build the app
docker compose run --rm app go build -o /tmp/server ./cmd/server

# Run tests
docker compose run --rm app go test ./...
```

## Development

```bash
# After editing .templ files:
templ generate ./templates/...

go build ./cmd/server   # Build
go test ./...           # Tests
go vet ./...            # Vet
```

## Project structure

```
csv-server/
├── cmd/server/main.go        # Entry point
├── internal/
│   ├── config/               # Config from environment
│   ├── csv/                  # CSV parsing and directory listing
│   └── handler/              # HTTP handlers
├── templates/
│   ├── pages/                # Page-level templ components
│   └── components/           # Reusable UI components
├── components/               # templUI components (button, icon, badge…)
├── web/static/               # Embedded CSS and JS
└── data/                     # Default CSV directory (dev)
```
