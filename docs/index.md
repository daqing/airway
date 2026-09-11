---
layout: home

hero:
  name: Airway
  text: Full-stack API framework in Go
  tagline: Inspired by Ruby on Rails — Gin, a dialect-aware SQL builder, a generics-based repo/ORM, SQL migrations, unified storage, and pluggable engines.
  actions:
    - theme: brand
      text: Get Started
      link: /cli-standalone
    - theme: alt
      text: View on GitHub
      link: https://github.com/daqing/airway

features:
  - title: One CLI for everything
    details: Install with `go install github.com/daqing/airway@latest` — scaffold projects and engines, generate code, and run migrations.
  - title: PostgreSQL, MySQL, SQLite
    details: One codebase, three databases. The driver is inferred from the DSN at runtime.
  - title: Engines
    details: Rails Engine-style feature modules shipped as independent Go modules, enabled with a single blank import.
  - title: Server-rendered views
    details: Type-safe HTML with templ, plus a WebSocket hub and unified local/S3/R2/COS storage.
---
