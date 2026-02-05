# AGENTS.md - Radiology Assignment Engine

## Project Overview
This project is a Radiology Assignment Engine written in Go. It automates the assignment of diagnostic imaging studies to radiologists based on complex rules, shifts, and roster availability. It includes a management UI built with standard Go templates and BeerCSS.

## Architecture
*   **Backend**: Go (Golang)
    *   `cmd/api`: Main entry point for the Web Server/API. Currently uses **in-memory storage** for Shifts, Radiologists, and Rules.
    *   `internal/assignment`: Core domain logic. Contains the `Engine` which handles matching, filtering, and load balancing.
    *   `internal/models`: Data structures.
*   **Frontend**: Server-side rendered HTML using `html/template`.
    *   **Framework**: [BeerCSS](https://www.beercss.com/) (Assets vendored in `ui/static`).
    *   **Templates**: Located in `ui/templates`.
*   **Testing**:
    *   **Unit Tests**: `internal/assignment/engine_test.go` (Mocks DB/Services).
    *   **Benchmarks**: `internal/assignment/benchmark_test.go`.
    *   **E2E Tests**: `cmd/api/e2e_test.go` using `chromedp` (Headless Chrome).

## Key Directives for Agents

### 1. Template Syntax
*   **Do NOT nest `{{ define "..." }}` blocks.**
*   The `layout.html` defines the shell. Individual pages (e.g., `rules.html`) should only define the `content` block:
    ```html
    {{ define "content" }}
      <!-- Page content here -->
    {{ end }}
    ```
*   Violating this causes `template.ParseFiles` to fail with "unexpected <define>" and breaks the app (HTTP 500).

### 2. Testing
*   **Execution**: Run tests from the project root to ensure relative paths to `ui/` are resolved correctly by the test server.
    ```bash
    go test -v ./...
    ```
*   **Chromedp / E2E**:
    *   These tests spin up a local HTTP server (`httptest`).
    *   **Environment**: Headless Chrome is required. Tests may time out or fail with "unknown IPAddressSpace value: Loopback" in restrictive container environments without proper GPU/Sandbox flags.
    *   **Selectors**: Use robust selectors. Prefer `chromedp.WaitVisible` before clicking or typing.
    *   **Modals**: BeerCSS modals work by adding the `open` attribute. In tests, you may need to force this if the JS click handler is flaky or slow:
        ```go
        chromedp.Evaluate(`document.getElementById('my-modal').setAttribute('open', 'true')`, nil)
        ```

### 3. Known Limitations / Tech Debt
*   **In-Memory Store**: The application currently resets data on restart. Production use requires a persistent database (Postgres).
*   **Security**: There is **NO CSRF protection** on POST endpoints. This must be added before deployment.
*   **Authentication**: The system is currently open (no login).

### 4. Code Style
*   Run `go fmt ./...` before submitting.
*   Ensure imports are cleaned up (`go mod tidy`).
