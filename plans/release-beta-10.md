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

- [ ] Done

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

## Planner review
