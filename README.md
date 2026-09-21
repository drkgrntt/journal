# Contour Journal

A server-rendered journaling app: write freely, rate your day, log small actions and things worth being grateful for. It's built to be a place to get thoughts out and mostly leave them there, not a tool for analyzing or revisiting what you wrote — no streaks, no gamification, no AI reading your entries. Patterns (a calendar + charts) surface on their own over time for anyone who wants to notice them, but nothing here is trying to optimize your mood or your habits. See `CLAUDE.md`'s "Product philosophy" section, the `/about` page (`internal/web/landings/about.templ`), and the blog entries under `internal/blog/entries/` for the full reasoning.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests

```bash
make all
```

Build the application

```bash
make build
```

Run the application

```bash
make run
```

Create DB container

```bash
make docker-run
```

Shutdown DB Container

```bash
make docker-down
```

DB Integrations Test:

```bash
make itest
```

Live reload the application:

```bash
make watch
```

Run the test suite:

```bash
make test
```

Clean up binary from the last build:

```bash
make clean
```
