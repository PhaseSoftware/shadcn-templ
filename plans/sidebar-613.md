# Issue #613: the mobile sidebar sheet keeps its modal state after a resize to desktop

- **Planner**: Claude
- **Executor**: Codex
- **Status**: done

## Context

Issue #613 (rchd4): open the mobile sidebar below 768px, widen the viewport to 768px or more. The sidebar content moves back into the desktop container, but the sheet dialog stays open: the backdrop keeps catching clicks, the page stays `aria-hidden`, the desktop page is unusable until Escape is pressed. The reporter suggests one line in `components/sidebar/sidebar.js`: call `window.tui.dialog.close(sidebarId + "-mobile")` before moving the content back.

**The Planner reproduced the bug** on `main` (8158e932) with Playwright on `http://localhost:8090/preview/sidebar-demo`, Chromium and WebKit, viewport 750 then 800 (`tmp/sidebar-613/repro.mjs`, gitignored, kept for the Executor):

- After the resize `window.tui.dialog.isOpen(popup)` is still true, the dialog root is not hidden, and an `elementFromPoint` hit test on the sidebar rail trigger lands on the backdrop, not on the button.
- `aria-hidden="true"` stays on every sibling of the dialog root under `<body>`, so the page content is hidden from assistive technology.
- Only the popup is `display: none`, through the `md:hidden` class on the sheet content in `sidebar.templ`. The root and the backdrop are not.
- The body scroll lock was released in the Planner's build, unlike in the report. That release does not come from `dialog.js` (its `unlockScroll` returns early while a modal dialog is open); another script drops the shared `data-tui-scroll-locked` attribute. Not this plan's subject, noted for the record.
- Cause: `init()` in `sidebar.js` moves the content on resize and never touches the dialog. `openDialog`, `closeDialog` and the `dialog-open-change` and `dialog-close` events in `components/dialog/dialog.js` are the only dialog surface the sidebar uses.

**What shadcn does.** In `sidebar.tsx` (Radix and Base UI variants alike) `openMobile` is `React.useState(false)` in `SidebarProvider`. `Sidebar` returns the `Sheet` only while `isMobile` is true, otherwise the desktop tree. `toggleSidebar` is `isMobile ? setOpenMobile(open => !open) : setOpen(open => !open)`. The sheet is `<Sheet open={openMobile} onOpenChange={setOpenMobile}>`. Nothing resets `openMobile` on a viewport change. Two consequences, both verified by the Planner with Playwright on `https://ui.shadcn.com/view/new-york-v4/sidebar-07` with the same 750 / 800 / 750 sequence:

1. Resize to desktop while the sheet is open unmounts the `Sheet`: overlay, scroll lock, `aria-hidden` marking and focus trap are gone at once (body overflow `visible`, zero `aria-hidden` nodes, no sheet in the DOM). `openMobile` stays true; `useSidebar().openMobile` reports true on desktop.
2. Resize back to mobile mounts the `Sheet` with `open={true}`: the sheet is open again immediately, with overlay, scroll lock (body overflow `hidden`) and `aria-hidden` marking (six nodes), without any user action.

The reporter's one-liner fixes consequence 1 and breaks consequence 2: after `dialog.close` the sheet stays closed when the viewport returns to mobile. It also leaves `window.tui.sidebar.openMobile()` as it is today, and that function is broken: it reads `sheet.open`, but the sheet content is a `<div>` (`data-tui-dialog-content` in `sheet.templ`), whose `open` property is undefined, so it returns false even while the sheet is open (the Planner's probe prints `openMobileApi: false` in the open state). shadcn's `openMobile` is the state, not the DOM.

**Dialog facts the Executor builds on** (`components/dialog/dialog.js`):

- `window.tui.dialog.open(popup)` and `.close(popup)` change the state without firing `dialog-open-change`. `close` runs the exit transition, releases the scroll lock and the `aria-hidden` marking immediately, hides the root once the animation finished, returns focus to the trigger or the previously focused element, then dispatches `dialog-close` (bubbles) on the popup.
- User dismissals (Escape, backdrop press, `data-tui-dialog-close` button, `dialog.toggle`) go through `requestOpenChange`, which dispatches the cancelable `dialog-open-change` event (bubbles, `detail.open`) on the popup before opening or closing. This event is the pendant of Base UI's `onOpenChange`. `dialog-close` is the pendant of `onOpenChangeComplete(false)` and also fires when `init` closes the dialog on desktop, so it is the wrong hook for the state.
- With the popup at `display: none` (`md:hidden`) there is no running animation, so `whenAnimationsFinish` settles at once.
- Returning focus on `close` matches the React unmount: Radix `FocusScope` and Base UI `FloatingFocusManager` both return focus in their unmount cleanup.

**Repo facts.** The component JS is bundled by `shadcn-templ bundle --watch` (`task dev`, `scripts-watch`, see `plans/scripts-611.md`). An edit to `sidebar.js` changes the bundle hash, so the watcher rewrites `components/scripts_bundle.go`; that generated file is committed and belongs in the task commit. The `task dev` the Planner found running predates the merge of #614 and answers every page with a templ watch-mode 500 (`failed to get watched strings`, missing temp file); the Executor restarts `task dev` before testing. `sidebar.js` has no copies, the CLI ships it from `components/sidebar/`. The `useSidebar` table in `internal/service/content/docs/components/sidebar.md` (lines 199 to 225) already describes `openMobile` as "whether the sidebar is open on mobile" and needs no change.

## Decisions

- **`openMobile` becomes sidebar state, the dialog is its rendering.** Like `SidebarProvider`'s `useState(false)`, one flag per sidebar, default false, never server rendered, never persisted (no cookie: shadcn persists only `open`). It lives as the attribute `data-tui-sidebar-open-mobile` on the `[data-tui-sidebar-wrapper]` element, present for true, absent for false, set only by the script. An attribute rather than a `Map` because a wrapper swapped out and in by htmx starts at false again, exactly like a remounted provider, and because the state is inspectable in the DOM like `data-state` already is.
- **`setOpenMobile(open)` is the one writer.** It sets the attribute and syncs the dialog: `open` and mobile and the dialog closed calls `window.tui.dialog.open(popup)`; not `open` and the dialog open calls `.close(popup)`. On desktop it only writes the attribute (shadcn on desktop renders no sheet, so `setOpenMobile(true)` there changes state and nothing visible; the next mobile viewport shows the sheet).
- **`toggleSidebar` on mobile calls `setOpenMobile(!openMobile)`**, the literal port of `setOpenMobile((open) => !open)`. It stops calling `window.tui.dialog.toggle` directly.
- **User dismissals write the state back, through `dialog-open-change` only.** A document level listener on `dialog-open-change` whose target `id` is `<sidebarId> + "-mobile"` for a sidebar on the page calls `setOpenMobile(event.detail.open, sidebarId)`. This is `onOpenChange={setOpenMobile}`. `dialog-close` is not listened to, see the dialog facts.
- **`init()` is the mount and unmount.** After moving the content it syncs the dialog with the viewport and the state: mobile and `openMobile` and dialog closed opens it (Sheet mounted with `open={true}`); desktop and dialog open closes it (Sheet unmounted). The state is not touched by `init`. Nothing else in `init` changes; the resize listener and the MutationObserver already re-run it.
- **`window.tui.sidebar.openMobile()` returns the state**, on every viewport, like the hook. `setOpenMobile` and `isMobile` keep their signatures; `setOpenMobile` calls the internal writer.
- **The reporter's `dialog.close` line does not land as such.** Its effect (no backdrop, no `aria-hidden`, no scroll lock, focus back on the trigger, on desktop) follows from the unmount sync in `init`.
- **Nothing else changes.** No templ, class, markup, docs or demo changes, no change in `dialog.js` or `sheet.templ`. The desktop toggle, cookie, tooltips and keyboard shortcut paths stay byte for byte.

## Tasks

### 1. Probe and failing baseline

- [x] Done

Restart `task dev` (the running one is stale, see Context) and confirm `curl -s -o /dev/null -w '%{http_code}' http://localhost:8090/preview/sidebar-demo` prints 200 and the page loads a `/assets/js/shadcn-templ-<hash>.js` that exists. Extend `tmp/sidebar-613/repro.mjs` (keep it gitignored) into `tmp/sidebar-613/probe.mjs <chromium|webkit>` that runs four scenarios on `/preview/sidebar-demo`, printing per step the fields the existing script prints (dialog open, root hidden, popup display, backdrop hit test on the rail trigger, scroll lock attribute, whether `main` sits under `aria-hidden`, where the content lives, `window.tui.sidebar.openMobile()`, active element), and exits non-zero on any expectation from the list below that fails:

- A, the issue: 750, click the trigger, 800, then 750. Expected after 800: dialog closed, root hidden, hit test lands on the trigger, no `aria-hidden` over `main`, no scroll lock, `openMobile()` true, active element the trigger. Expected after the second 750: dialog open, scroll locked, `aria-hidden` over `main`, `openMobile()` true.
- B, dismiss then resize: 750, click the trigger, press Escape, 800, 750. Expected: `openMobile()` false after Escape and after both resizes, dialog closed after the second 750.
- C, the API on desktop: 800, `window.tui.sidebar.setOpenMobile(true)`, expected `openMobile()` true and dialog closed; then 750, expected dialog open; `setOpenMobile(false)`, expected dialog closed and `openMobile()` false.
- D, desktop untouched: 800, click the trigger twice, expected `data-state` on the wrapper `collapsed` then `expanded`, `openMobile()` false throughout, dialog never open.

Run it on untouched `main` in both engines and save the output as `tmp/sidebar-613/baseline-chromium.log` and `baseline-webkit.log`.

Done when: the probe exists, runs in both engines, and the baseline logs show scenario A failing after 800 (dialog open, hit test on the backdrop, `aria-hidden` over `main`) and `openMobile()` false while the sheet is open, while D passes.

Checks: `node tmp/sidebar-613/probe.mjs chromium`, same for webkit, both exiting non-zero on `main` with only the expected failures.

### 2. `openMobile` state in `sidebar.js`

- [x] Done

`components/sidebar/sidebar.js`, per Decisions: `openMobileOf(sidebarId)` reads the attribute; `setOpenMobile(open, sidebarId)` writes it and syncs the dialog through `window.tui.dialog.open` and `.close` on `document.getElementById(sidebarId + "-mobile")`; `toggleSidebar` below md calls `setOpenMobile(!openMobileOf(sidebarId), sidebarId)`; `init()` syncs mount and unmount after the content move; a `document.addEventListener("dialog-open-change", ...)` maps the popup id back to the sidebar and calls `setOpenMobile(event.detail.open, sidebarId)`; `window.tui.sidebar.openMobile` returns `openMobileOf`, `window.tui.sidebar.setOpenMobile` calls the writer. Comments name the shadcn members they port (`openMobile`, `setOpenMobile`, `toggleSidebar`, the `Sheet` mount and unmount). Let `scripts-watch` rewrite `components/scripts_bundle.go` and commit it with the change.

Done when: `probe.mjs` passes all four scenarios in Chromium and WebKit, and `git diff --stat` touches only `components/sidebar/sidebar.js` and `components/scripts_bundle.go`.

Checks: `node tmp/sidebar-613/probe.mjs chromium`, same for webkit, `go build ./...`, `go vet ./components/...`, `git diff --check`.

## Executor log

### Task 1 (Codex, 2026-09-21)

Restarted the stale `task dev` through the normal watcher workflow. Preview returns HTTP 200 and references the existing `assets/js/shadcn-templ-d973fedcd37cd54a.js`. Added gitignored `tmp/sidebar-613/probe.mjs` with scenarios A through D and per-step state assertions. Both baseline engines report 13 expected failed expectations: the desktop modal remains open, the backdrop intercepts the rail, main stays aria-hidden, focus is not returned, the mobile API always reports false, and desktop API opening incorrectly opens the dialog. The known resize scroll-lock release is also captured. Scenario D passes in both engines. Logs: `tmp/sidebar-613/baseline-{chromium,webkit}.log`. Focus assertion accepts either sidebar trigger because the rail becomes the first DOM trigger on desktop, while opening on mobile uses the header trigger. No implementation changes for this task; probe and logs remain gitignored as required.

### Task 2 (Codex, 2026-09-21)

Implemented wrapper-owned `openMobile` state with a single setter, mobile toggling through that setter, viewport mount/unmount synchronization in `init`, and user dismissal synchronization through `dialog-open-change`. The public API now reads state independently of the dialog. Desktop toggle, cookie, tooltips and keyboard shortcut paths remain unchanged. The normal scripts watcher generated bundle `950ca0a6a66c4dec` and updated `components/scripts_bundle.go`; no manual generation or minification was run.

All four probe scenarios pass in Chromium and WebKit (zero failed expectations); logs are `tmp/sidebar-613/fixed-{chromium,webkit}.log`. This includes desktop backdrop removal, aria-hidden cleanup, focus return to a sidebar trigger, mobile reopen with scroll lock, Escape state reset, desktop API behavior, and unchanged desktop toggling. `go build ./...`, `go vet ./components/...`, and `git diff --check` pass. Build/vet required normal cache access outside the sandbox. Implementation diff contains only `components/sidebar/sidebar.js` and the watcher-generated `components/scripts_bundle.go`; the task log/checkmarks are also committed as the plan rules require. The pre-existing `plans/chart-612.md` modification is untouched. Both tasks are ready for Planner review; status remains Planner-owned.

## Planner review

### Review (Claude, 2026-09-21)

Read both commits (`8fbb081d`, `e34c0a71`) and re-ran every check on the Executor's `main`.

- **Task 1: accepted.** `tmp/sidebar-613/probe.mjs` covers scenarios A to D with the fields the plan asked for and exits non-zero on a failed expectation. The baseline logs show the issue's failure and the `openMobile()` false-while-open bug.
- **Task 2: accepted.** The diff is what Decisions describe, nothing more: `openMobileOf`, one writer `setOpenMobile`, `toggleSidebar` through the writer, the mount and unmount sync in `init`, the `dialog-open-change` listener, the API reading the state. `dialog-close` is not listened to. `dialog.js`, `sheet.templ`, templ, classes and docs are untouched; the diff touches `sidebar.js` and the watcher-generated `components/scripts_bundle.go` (`950ca0a6a66c4dec`, the served bundle carries the new attribute).
- **Verification by the Planner.** `probe.mjs` in Chromium and WebKit: 0 failed expectations each. A second run in a visible Chromium with OS level window resizes through CDP `Browser.setWindowBounds` (`tmp/sidebar-613/headed.mjs`, real `resize` events, 750/800/750): after the resize to 800 the dialog is closed, the root hidden, the rail trigger receives the hit test, `main` is not `aria-hidden`, the scroll lock is released and focus sits on the header trigger, the element focused before the sheet opened, which is the React unmount's return focus; `openMobile()` stays true; back at 750 the sheet is open again with scroll lock and `aria-hidden` marking, as on ui.shadcn.com. Escape resets the state and survives both resizes. The desktop API sets the state without opening anything and the next mobile viewport opens the sheet. The desktop toggle collapses and expands as before. After the unmount a real click on the rail collapses the sidebar. Screenshots `tmp/sidebar-613/headed-A-desktop.png` (no backdrop) and `headed-A-mobile-again.png` (sheet with backdrop). `go build ./...`, `go vet ./components/...`, `git diff --check` pass.
- **Not verified in real Safari and real Chrome.** Safari's "Allow remote automation" is off and the Claude Chrome extension is not connected; no Chrome is installed. `tmp/sidebar-613/safari.mjs` drives the same sequence through `safaridriver` and can run once that setting is on.
- **Noted, out of scope.** The shared body scroll lock is released on resize by another script while a modal dialog is open (seen on `main` before the fix). Since the fix closes the dialog on desktop and reopens it on mobile the sidebar no longer exposes this, but the shared lock remains a separate topic.

No new tasks. Status done.
