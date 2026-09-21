---
title: "September 2026 - Component script build artifact"
description: "Build component JavaScript with the CLI and serve it with your other assets."
date: 2026-09-21
---

Component JavaScript is now built by `shadcn-templ bundle`. `add` rebuilds it when it writes component scripts, and `bundle --watch` keeps it current during development.

The copied `components/scripts.go` HTTP handler and `components/embed.go` are gone. `scripts.templ` renders one script tag using the URL in the generated `scripts_bundle.go`. Configure the output directory and public URL through `scripts.dir` and `scripts.path` in `components.json`.

The hashed JS asset is ignored and rebuilt before deployment, like Tailwind output. Commit the Go manifest and component sources, and run `shadcn-templ bundle` before `go build`. Serve the hashed file with one-year immutable caching.

Scaffolded apps default to production asset serving. `SHADCN_TEMPL_DEV=true` explicitly enables development, and `task dev` sets it. `TEMPL_DEV_MODE=true` is also supported. `GO_ENV=development` remains a deprecated alias for this minor version.

See [Installation](/docs/installation#javascript) for setup and migration instructions.
