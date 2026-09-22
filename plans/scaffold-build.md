# scaffold-build: `task build` and a Dockerfile in the scaffold, so deploying is one command

- **Planner**: Claude
- **Executor**: Codex
- **Status**: done

## Context

Since #614 the component JavaScript is a build artifact: `shadcn-templ bundle` writes `assets/js/shadcn-templ-<hash>.js` (gitignored) and `components/scripts_bundle.go` (committed). A deploy from a clean checkout therefore has to run three things before `go build`: Tailwind, `bundle`, `templ generate`. The scaffold (`cmd/shadcn-templ/templates/templ-app/`) wires all of that into `task dev` for development and into nothing for deployment. `installation.md:299` says in bold that a clean checkout must run `bundle` before `go build`, `cli.md:84` repeats it. The failure when someone forgets is silent: the scaffold embeds `assets/` into the binary (`assets/assets.go.tmpl`, served through `http.FS(assets.Assets)` in production), so a binary built without the bundle serves 404 for the script tag and every component is dead, while CSS missing at least looks broken.

The pipeline exists; it has no name and no place to run. This plan gives it both. Nothing in the library changes.

**Facts the Executor builds on.**

- Scaffold today: `.gitignore` (`assets/css/output.css`, `assets/js/shadcn-templ-*.js`), `Taskfile.yml` (`templ`, `tailwind`, `scripts-watch`, `dev`, a `FREE_PORT` helper), `assets/`, `components/`, `layouts/`, `pages/`, `go.mod.tmpl` (module placeholder plus `go 1.24`), `main.go.tmpl`. `templates.Create` copies everything under `templ-app` verbatim, strips a `.tmpl` suffix and replaces the module placeholder; a new file in the directory ships automatically. `init` runs `add component-example` into the scaffold and prints `cd <name>`, `go mod tidy`, `task dev`.
- `main.go.tmpl` binds `PORT` exactly when set and otherwise walks from 8090 to the next free port. Production serving is the default; `SHADCN_TEMPL_DEV=true` (set by `task dev`) or `TEMPL_DEV_MODE=true` switches to serving `./assets` from disk.
- The repo's own `Dockerfile` is the reference shape: `golang` build stage that `go install`s templ pinned to `go list -m -f '{{.Version}}' github.com/a-h/templ`, runs `templ generate`, downloads the Tailwind standalone binary for the architecture, runs Tailwind with `--minify`, runs `go run ./cmd/shadcn-templ bundle`, then `go build`; an `alpine` final stage with the binary. `.dockerignore` exists at the repo root.
- The repo's `go.mod` pins templ as a Go 1.24 tool (`tool github.com/a-h/templ/cmd/templ`) and its `Taskfile.yml` runs `go tool templ generate --watch`. A `tool` directive is Go's pendant of a devDependency: pinned in the module file, run with `go tool <name>`, built from the module cache, no install step. `go mod tidy` adds the `require` for a `tool` line.
- The "Create Taskfile" snippet for existing projects (`installation.md:211-243`) predates #614: it has no `scripts-watch` task, so a hand-made project never runs the bundle watcher. A gap to close in the same docs pass.
- Tests: `cmd/shadcn-templ/commands/add_scripts_test.go:138` scaffolds with `RunInit` and asserts on files; the place for scaffold assertions.
- Environment: Docker on this machine is `colima`, not running at the time of planning (`docker version` hangs until `colima start`). `task` 3.52, Go 1.26, `tailwindcss` on PATH.

## Decisions

- **`task build` is the pipeline, defined once.** In the scaffold's `Taskfile.yml`, in this order: Tailwind with `--minify`, `shadcn-templ bundle`, `templ generate`, `go build -o bin/app .`. The order is the dependency order: the bundle must exist before `go build` embeds `assets/`, `templ generate` must run so committed watch-mode `_templ.go` files (templ's `--watch` writes files that read from temp text files) never reach a binary. `bin/` joins the scaffold `.gitignore`.
- **The scaffold pins its two Go tools with `tool` directives.** `go.mod.tmpl` gains `tool github.com/a-h/templ/cmd/templ` and `tool github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ`; `go mod tidy` (already in the printed next steps) resolves their versions together with the module `require` that the copied components pull in. The Taskfile runs `go tool templ ...` and `go tool shadcn-templ ...`, in `dev` and in `build`, the way the repo's own Taskfile already runs `go tool templ`. Result: a clone needs Go and Tailwind, nothing else, and the CLI that bundles is always the version the components came from. Tailwind stays a binary on PATH; it is not a Go tool.
- **A Dockerfile that runs `task build`.** Three stages in the shape of the repo's: `golang:1.25` build stage that downloads the Tailwind standalone binary for the architecture (the repo's `ARCH` block, same pinned version as the repo), `go install`s `task` pinned by version, copies the source and runs `task build`; a final `alpine` stage with `bin/app`, `ENV PORT=8090`, `EXPOSE 8090`, `CMD ["./app"]`. No `SHADCN_TEMPL_DEV`, so assets come from the embed. `task` in the image is the price of defining the pipeline once; four duplicated lines that can drift are exactly the failure this plan removes. A `.dockerignore` next to it: `.git`, `bin`, `assets/css/output.css`, `assets/js/shadcn-templ-*.js`, `tmp`.
- **The docs describe one command per situation.** `installation.md` gets a `## Build and Deploy` section after `## Serve Assets`: scaffold users run `task build` locally and `docker build` for an image; existing-project users add the `build` task from the scaffold to their Taskfile and copy the Dockerfile. The bold sentence in `## JavaScript` and the sentence in `cli.md` `## bundle` point at that section instead of instructing by hand. The "Create Taskfile" snippet gains `scripts-watch` and `build` so it equals the scaffold's Taskfile.
- **The library and the registry do not change.** No component, no CLI command, no `components.json` field. `init`'s printed next steps stay `go mod tidy` and `task dev`.

## Tasks

### 1. `task build` and the `tool` directives

- [x] Done

`go.mod.tmpl`: the two `tool` lines. `Taskfile.yml`: `templ`, `scripts-watch` and the new `build` task through `go tool`, `build` per Decisions. `.gitignore`: `bin`. Verify by scaffolding against the dev server's registry (`shadcn-templ init -t templ --cwd <scratch> --registry http://localhost:8090`, then `go mod tidy`, `task build`) and running `bin/app` from a directory that is not the project (so nothing can come from disk) with `PORT=8093`: `/` returns 200 with a script tag naming `assets/js/shadcn-templ-<hash>.js`, that URL returns 200 with `Cache-Control: public, max-age=31536000, immutable`, and `/assets/css/output.css` returns 200. Then `task dev` in the scratch project still works (watchers start, a page loads).

Done when: those checks pass and `go.mod` in the scratch project shows both tools pinned to the same shadcn-templ version the `require` line carries.

Checks: the flow above, `go test ./cmd/shadcn-templ/...`, `git diff --check`.

### 2. Dockerfile and `.dockerignore`

- [x] Done

Both files in the scaffold per Decisions. Verify with `colima start`, then in the scratch project from task 1: `docker build -t scaffold-build .` and `docker run --rm -p 8094:8090 scaffold-build`; `curl` `/`, the script URL from the page and `/assets/css/output.css` return 200; the image contains no `assets/` directory next to the binary (`docker run --rm scaffold-build ls` shows `app` only). If colima cannot start on this machine, the Executor records that and the Planner runs the Docker half in review.

Done when: the image builds from a clean `git clone` of the scratch project (clone it to a second scratch directory first, so nothing gitignored is present) and serves all three URLs.

Checks: the flow above, `git diff --check`.

### 3. Docs and the scaffold test

- [x] Done

`installation.md`: the `## Build and Deploy` section, the two pointers, the "Create Taskfile" snippet with `scripts-watch` and `build` (and `go tool` where the scaffold uses it, with a one-line note that `go get -tool` pins the CLI the way the scaffold does). `cli.md` `## bundle`: the last sentence points at the section. `add_scripts_test.go`: the scaffold assertions gain `Dockerfile`, `.dockerignore`, a `build` task in `Taskfile.yml` and both `tool` lines in `go.mod`.

Done when: `go test ./cmd/shadcn-templ/...` passes and the docs pages `/docs/installation` and `/docs/cli` render the new text on the dev server.

Checks: `go test ./cmd/shadcn-templ/...`, `go build ./...`, `git diff --check`, the two pages in a browser.

## Executor log

### Task 1 (Codex, 2026-09-22)

Added the sequential production build, switched the dev commands to `go tool`, declared both tools and ignored `bin/`. The user explicitly approved running generation within production builds in temporary scaffold projects after automatic approval initially rejected the AGENTS.md exception. No manual generation was run in this repository.

Verified with a locally built `/tmp/scaffold-build-cli`: `init -t templ --cwd /tmp/scaffold-build-smoke --registry http://localhost:8090`, then `go mod tidy` and `task build` in the generated `templ-app`. The tool declarations resolve to templ v0.3.1020 and shadcn-templ v2.0.0-beta.10 in the same module graph as the application (no replacement or global CLI used for the build). The tools intentionally belong to different modules and have different versions; the CLI uses the sole shadcn-templ require version.

Ran `/tmp/scaffold-build-smoke/templ-app/bin/app` from `/tmp` with `PORT=8093`: page, hashed JS and CSS all return 200 with nonempty bodies; JS has `Cache-Control: public, max-age=31536000, immutable`. Then `task dev` starts all three watchers through the new tool commands, and both app 8091 and proxy 7332 return 200. Only the scratch proxy port was temporarily changed to avoid the repository's 7331; restored afterward. Scratch processes stopped. Logs and reusable HTTP assertions: `tmp/scaffold-build/`.

`go test ./cmd/shadcn-templ/...` and `git diff --check` pass.

### Task 2 (Codex, 2026-09-22)

Added three stages: Go 1.25 tools (Tailwind v4.3.3 for x64/arm64 and Task v3.52.0), application build invoking only `task build`, and Alpine runtime with just the binary in `/app`. `CGO_ENABLED=0` makes the Go binary usable in Alpine. Module download is cached separately from source copying. Added the specified .dockerignore entries.

Colima started successfully (aarch64, 2 CPUs, 2 GiB). Copied the two container files into Task 1's scratch scaffold, initialized that throwaway fixture as a git repository and cloned it into `/tmp/scaffold-build-clone`. The clone is clean and contains no `bin/`, CSS output or hashed JS; assertions recorded in `tmp/scaffold-build/clone.json`. It does contain templ output from the preceding dev watcher, so the Docker production generation also replaces any watch-mode output.

`docker build -t scaffold-build .` from that clone passes, producing image `3a1d73f719eb`. The host has the legacy Docker builder; an initial attempt using the optional `--progress=plain` logging flag was rejected, then the exact plain build command passed without changing Docker configuration. `docker run --rm -d -p 8094:8090 --name scaffold-build-smoke scaffold-build` serves `/`, `/assets/js/shadcn-templ-2323cae6e0874e49.js` and `/assets/css/output.css` with HTTP 200 and nonempty bodies. JS has `public, max-age=31536000, immutable`; CSS has `no-cache`. `docker run --rm scaffold-build ls -a` shows only `.`, `..`, `app`. The test container was stopped afterward. Logs: `docker-build.log`, `assets-docker.log`, `image-files.log` in `tmp/scaffold-build/`.

`git diff --check` passes. The image was tested on arm64; the x64 download branch was not executed on this machine.

### Task 3 (Codex, 2026-09-22)

Added Build and Deploy with the local build, embedded binary and container flows. The JavaScript and CLI deployment instructions link to it. Existing projects get both tool-pinning commands and the exact scaffold Taskfile, including the missing scripts watcher, production build and dev asset flag. Scaffold integration assertions now cover Dockerfile, .dockerignore, the build task and both tool directives.

`go test ./cmd/shadcn-templ/...`, `go build ./...` and `git diff --check` pass. Playwright in Chromium and WebKit loads `/docs/installation` and `/docs/cli` with HTTP 200 and no page errors; assertions confirm the new heading, build/tool/watcher commands and deployment links. Logs and browser probe: `tmp/scaffold-build/`.

At the user's request, all work stays uncommitted for review. The initial local Task 1 commit was undone with its changes preserved. No repository commits or pushes remain from this task. A separate throwaway scaffold repository exists solely to verify the clean-clone Docker build.

## Planner review

### Review (Claude, 2026-09-22)

Read the working tree diff (uncommitted at Axel's request) and repeated the whole verification from scratch with a CLI built from the working tree, a fresh scaffold against the dev server's registry, and a second clean clone for Docker.

- **Task 1: accepted.** `task build` runs the four steps in dependency order through `go tool`; `go.mod.tmpl` pins both tools and `go mod tidy` resolves them to templ v0.3.1020 and shadcn-templ v2.0.0-beta.10, the same version the components' `require` carries. Planner: `task build` in the scratch scaffold, `bin/app` started from `/tmp` on `PORT=8095`: `/` 200 with the script tag, the hashed JS 200 (182,371 bytes, `public, max-age=31536000, immutable`), `output.css` 200 (67,109 bytes). Nothing came from disk.
- **Task 2: accepted.** The Dockerfile is the repo's shape reduced to what the scaffold needs: a `tools` stage with the pinned Tailwind binary and `task` v3.52.0, a build stage that only runs `task build` under `CGO_ENABLED=0`, an `alpine` stage with the binary. Planner: `git clone` of the scratch scaffold into an empty directory (no `bin/`, no CSS output, no bundle present), `docker build` on colima, `docker run` on 8096: all three URLs 200 with identical byte counts to the local binary; `ls -a` in the image shows `app` only. The x64 Tailwind branch was not executed here, the machine is arm64; the asset name matches the Tailwind release naming the repo's own Dockerfile relies on.
- **Task 3: accepted.** `## Build and Deploy` says one command per situation; the two former hand-instructions point at it; the existing-project Taskfile equals the scaffold's, including the `scripts-watch` task that was missing since #614 and the `go get -tool` lines. The scaffold test asserts the Dockerfile, `.dockerignore`, the `build` task and both `tool` directives. `go test ./cmd/shadcn-templ/...`, `go build ./...`, `git diff --check` pass.

No new tasks. Status done. Nothing is committed; Axel commits.
