# release-beta-10: v2.0.0-beta.10

- **Planner**: Claude
- **Executor**: Codex
- **Status**: ready

## Context

`v2.0.0-beta.9` was tagged on 2026-09-03 at `f083888a`. `main` has 50 commits since, four merged PRs plus the a11y and menu work that landed directly:

- **#614 scripts bundle** (`plans/scripts-611.md`): the component JS is no longer served by a runtime handler. `components/scripts.go` and `components/embed.go` are gone from the `scripts` registry item; `shadcn-templ bundle` writes `assets/js/shadcn-templ-<hash>.js` and a generated `components/scripts_bundle.go`; `components.json` gains `"scripts": {"dir", "path"}`; `add` rebuilds the bundle and `bundle --watch` is the dev loop. This is the one breaking change of the release: a user project has to run `bundle` once and serve the file.
- **#612 chart** (`plans/chart-612.md`): gaps, `Ticks`, `Domain`, `AllowDataOverflow`, `StrokeDasharray`, `DotProps.Show`, `Hide`, curve constants, value scale computed in Go.
- **#600 accessibility** (`plans/a11y-600.md`) and `plans/menu-regressions.md`: tabs, accordion, menus, popover, select, combobox ARIA and keyboard, submenu positioning.
- **#615 sidebar** (`plans/sidebar-613.md`): `openMobile` survives the viewport change like shadcn, `tui.sidebar.openMobile()` reports the state.
- **scroll lock** (`plans/scroll-lock.md`): lands before the tag, see Decisions.
- Site only, not release notes: Umami instead of Plausible, Compose deploy.

Release mechanics: no version constant in the code, no changelog file to maintain (`changelog.md` is a header). A release is a git tag plus a GitHub release whose body follows the beta.9 shape (`### Patch Changes` with one bullet per change; this release adds `### Breaking Changes` and `### Minor Changes` above it). The docs site deploys on every push to `main` (`.github/workflows/deploy.yml`), so the site is already current; the tag is what `go install ...@latest` and the registry version pick up.

## Decisions

- **The scroll lock ships in this release.** beta.9 already has the bug; the fix is small and probe-verified. Task 1 waits for `plans/scroll-lock.md` to be done.
- **The tag and the GitHub release are Axel's.** The Executor prepares and verifies; nothing here pushes a tag or publishes.
- **One breaking bullet, written as a migration.** The scripts bundle gets its own `### Breaking Changes` section with the three steps a user runs (`go get` the new version, `shadcn-templ bundle`, serve `assets/js`), pointing at `/docs/installation#javascript`.
- **Notes are written from the plans, not from commit messages.** Each merged plan's Decisions section is the source; commits like `chart-612 3: ...` are not user facing.

## Tasks

### 1. Migration smoke test of the CLI

- [x] Done

Blocked until `plans/scroll-lock.md` is done. In a scratch directory outside the repo (`tmp/release-beta-10/` is fine, gitignored): `shadcn-templ init` built from `main`, `add dialog sidebar dropdown-menu`, check that `components.json` carries the `scripts` field, that `assets/js/shadcn-templ-<hash>.js` exists and contains `components/baseui/scroll_lock.js`, that `components/scripts_bundle.go` names that hash, that `go build ./...` in the scratch project passes, and that a page rendering `scripts.Scripts()` links the file. Then simulate a beta.9 project: remove the `scripts` field and the bundle, run `add button`, confirm the field is written back and the bundle rebuilt with the one printed line from scripts-611 task 2.

Done when: both flows pass and the Executor log records the exact commands and output lines.

Checks: the flows above, `go test ./cmd/shadcn-templ/...`.

### 2. Full verification on main

- [ ] Done

`go build ./...`, `go test ./...`, `git status` clean, `task dev` serves `/docs/components/dialog`, `/docs/components/chart`, `/preview/sidebar-demo` without console errors in Chromium and WebKit (Playwright, `tmp/a11y-600/node_modules`), and the scroll-lock probe and the sidebar-613 probe pass once more on the exact commit to be tagged.

Done when: every check passes on the commit named in the Executor log.

Checks: as listed.

### 3. Release notes draft

- [ ] Done

Write the GitHub release body into the Executor log, in the beta.9 shape: `### Breaking Changes` (the scripts bundle, as a migration), `### Minor Changes` (chart, sidebar state, accessibility, scroll lock), `### Patch Changes` (the remaining fixes from the a11y and menu plans). One bullet per change, no commit references, no plan names.

Done when: the Planner accepts the text in the review; Axel tags `v2.0.0-beta.10` on the verified commit and publishes the release with it.

Checks: `gh release view v2.0.0-beta.9 --json body` as the format reference.

## Executor log

### Task 1 (Codex, 2026-09-21)

Fresh and simulated-beta.9 flows pass against the local registry serving main, including the unshipped ScrollLocker and Escape fixes. Scratch project: `/tmp/release-beta-10-smoke/templ-app`. Logs: `tmp/release-beta-10/{init,add,migration,migration-fixed,cli-tests}.log`.

Commands run from the repository unless a working directory is given:

```sh
go build -o /tmp/shadcn-templ-beta10 ./cmd/shadcn-templ
mkdir -p /tmp/release-beta-10-smoke
/tmp/shadcn-templ-beta10 init -t templ --cwd /tmp/release-beta-10-smoke --registry http://localhost:8090
/tmp/shadcn-templ-beta10 add dialog sidebar dropdown-menu --cwd /tmp/release-beta-10-smoke/templ-app --registry http://localhost:8090
# In /tmp/release-beta-10-smoke/templ-app:
go mod tidy
PATH="/tmp/release-beta-10-bin:$PATH" task dev PORT=8101
go mod tidy
go build ./...
```

`/tmp/release-beta-10-bin/shadcn-templ` points to the freshly built CLI. Only the scratch Taskfile's templ proxy port was changed to 7332, to keep the repo's 7331 proxy running. The scaffold's nested task selected its free app port 8091 despite the outer PORT=8101 argument; the page was checked on the actual reported port. The normal task-dev watcher generated the scratch `_templ.go` files; no manual generation/minification. The scratch dev processes were then stopped before the legacy simulation, so a watcher could not mask a missing bundle.

Fresh output:

```text
Project initialization completed.
Bundle: /tmp/release-beta-10-smoke/templ-app/assets/js/shadcn-templ-e72b6a40bbb6a67d.js. Render @components.Scripts() once in your layout <head>.
```

Assertions pass: config scripts is `{dir: assets/js, path: /assets/js}`; exactly one hashed bundle exists; it contains `// components/baseui/scroll_lock.js`; its SHA-256 prefix matches the name; `components/scripts_bundle.go` references that name; `go build ./...` passes. `curl http://localhost:8091/` renders `<script defer nonce="" src="/assets/js/shadcn-templ-e72b6a40bbb6a67d.js"></script>` and the asset returns HTTP 200. The actual package is components, not the plan's illustrative scripts.Scripts spelling.

For the legacy simulation a Python script removed only the scripts property from the scratch components.json and deleted its hashed JS bundle. Then:

```sh
/tmp/shadcn-templ-beta10 add button --cwd /tmp/release-beta-10-smoke/templ-app --registry http://localhost:8090
```

**Release-blocking defect found and fixed within this task:** on the initial implementation the command skipped the existing button template, wrote no scripts config and rebuilt no bundle. `add` only checked the newly added files for JS. The minimal fix treats a defaulted scripts config as requiring a bundle build as well, so existing scripts migrate even on a template-only add. Added a regression case using the non-default components alias; it failed before the fix (`scripts config was not migrated: <nil>`, log `migration-regression-before.log`) and passes after it. No component behavior changed.

Rebuilt the CLI with the same build command and repeated the same add command on that unmigrated scratch project. Output:

```text
Checking registry.
Updating files.
Skipped 1 file: (use --overwrite to overwrite)
  - components/button/button.templ
Bundle: /tmp/release-beta-10-smoke/templ-app/assets/js/shadcn-templ-e72b6a40bbb6a67d.js. Render @components.Scripts() once in your layout <head>.
Serve assets/js at /assets/js.
```

The serving hint occurs once, the persisted config and regenerated bundle/hash/manifest assertions pass, and the scratch `go build ./...` passes. `go test ./cmd/shadcn-templ/...` and `git diff --check` pass. The review should include the two CLI files changed to repair the failed migration expectation.

## Planner review
