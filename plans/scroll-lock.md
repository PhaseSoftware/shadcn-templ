# scroll-lock: one ScrollLocker for every modal, the @base-ui/utils/useScrollLock pendant

- **Planner**: Claude
- **Executor**: Codex
- **Status**: ready

## Context

Found while fixing #613 (`plans/sidebar-613.md`): a modal dialog loses its page scroll lock while it is still open. The Planner measured on `main` (8158e932 plus #615) with Playwright in Chromium:

- `/docs/components/dialog`, dialog open, then any `childList` mutation on `<body>` (append and remove a `<div>`): `data-tui-scroll-locked` gone, `body.style.overflow` reset, dialog still open. That mutation is every toast, every htmx swap, every menu that opens inside a dialog, and the sidebar content move on resize.
- `/preview/sidebar-demo` at 750px, sheet open, open a dropdown inside the sheet: lock gone while the sheet is open.

The cause is that the lock has five owners instead of one. `dialog.js`, `drawer.js`, `dropdownmenu.js`, `contextmenu.js` and `select.js` each carry a copy of `lockScroll` and `unlockScroll` that set and remove the same `data-tui-scroll-locked` attribute plus `body.style.overflow` and `paddingRight`, each guarded by its own idea of "is anything open":

- `dialog.js` (`anyModalOpen`): its own `openStack` plus the selector `dialog[open][data-tui-dialog-show-modal="true"]` for native-dialog drawers.
- `drawer.js`: only that native-dialog selector. Since the Base UI DOM port, `dialog.js` popups are `<div>`s, so the drawer's guard never sees an open dialog. Its `MutationObserver` calls `unlockScroll()` on every body mutation. That is the release measured above.
- `dropdownmenu.js` and `contextmenu.js` (`anyOpen`) and `select.js` (`allContents().some(isOpen)`): only their own popups. Closing a menu while a dialog is open releases the dialog's lock. That is the second measurement.

`dialog.js` additionally calls `unlockScroll()` from its own `MutationObserver` (guarded correctly) and from `destroyDialog`. Submenus never touch the lock (`openSub` does not call `lockScroll`), which matches Base UI, where nested menus are never modal.

**What Base UI does** (every file read on `mui/base-ui` master, copies in `tmp/scroll-lock/baseui/`, gitignored):

- `packages/utils/src/useScrollLock.ts` is the only place that touches the page. A module singleton `SCROLL_LOCKER = new ScrollLocker()` with `lockCount`, `restore`, `timeoutLock`, `timeoutUnlock`. `acquire(referenceElement)` increments the count and, on the first acquisition with no `restore` pending, starts a 0ms timeout to `lock`; it returns `release`. `release` decrements and, at zero with a `restore` present, starts a 0ms timeout to `unlock`, which calls `restore` if the count is still zero. Nothing else can release the lock; no observer inspects the DOM for open popups.
- `lock` first checks `isPageScrollLocked` (computed `overflowY` of the viewport scroller is `hidden` or `clip`). If the page is already locked by someone else it waits with a `MutationObserver` on `html` and `body` attributes and locks once the foreign lock clears. Otherwise it picks `preventScrollOverlayScrollbars` when `platform.os.ios` or the scrollbars take no width (`hasInsetScrollbars` false), else `preventScrollInsetScrollbars`.
- `preventScrollOverlayScrollbars`: `overflowY` and `overflowX` `hidden` on the viewport scroller (`getViewportScroller`: `html` when `isOverflowElement(html)`, else `body`), restored on release.
- `preventScrollInsetScrollbars`: on WebKit with pinch zoom it does nothing. Otherwise it reads everything first (scroll offsets, computed `scrollbarGutter` of `html` for a `both-edges` value, scrollability, constant `overflow: scroll`, scrollbar width and height as `innerWidth - body.clientWidth` clamped at 0, body margins), then writes. When `supportsStableScrollbarGutter` (a `CSS.supports('scrollbar-gutter', 'stable')` probe that also measures whether the gutter really keeps the width) it sets only `html.style.scrollbarGutter = 'stable'` (or `'stable both-edges'`) and `overflow hidden` on the scroller. Otherwise it sets `scrollbarGutter`, `overflowY`, `overflowX` on `html` (`scroll` where the page was scrollable or had constant overflow), `position: relative`, `height: calc(100dvh - margins - scrollbarHeight)`, `width: calc(100vw - margins - scrollbarWidth)`, `boxSizing`, `overflow hidden`, `scrollBehavior: unset` on `body`, restores the scroll offsets on `body`, sets `data-base-ui-scroll-locked` on `html`, and re-applies itself on `resize` through cleanup plus an animation frame. Cleanup restores the recorded inline styles, the scroll offsets on `html`, the attribute and `scrollBehavior`.
- `useScrollLock(enabled, referenceElement)` is a layout effect: acquire while enabled, release on cleanup or when `enabled` flips.
- Callers: `useDialogRoot.ts:86` `useScrollLock(open && modal === true, popupElement)`. `MenuPositioner.tsx:270` and `SelectPositioner.tsx:89` go through `packages/react/src/utils/useAnchoredPopupScrollLock.ts`: `useAnchoredPopupScrollLock(enabled, touchOpen, positionerElement, referenceElement)` locks when enabled and not touch-opened; for a touch open it locks only when the positioner is at least `viewportWidth - 20` wide (`VIEWPORT_WIDTH_TOLERANCE_PX`). Menu: `enabled = open && (menubarModal || popupModal)` where `popupModal = modal && lastOpenChangeReason !== 'trigger-hover'`, `touchOpen = openMethod === 'touch'`. Select: `enabled = (alignItemWithTriggerActive || modal) && open`, same `touchOpen`. Nested menus have `modal` undefined and never lock.
- `platform/os.ts`: `ios = /^i(os$|p)/.test(lowerPlatform) || (lowerPlatform === 'macintel' && maxTouchPoints > 1)`; `platform/engine.ts`: `webkit = CSS.supports('-webkit-backdrop-filter:none')`, `gecko`, `blink`. `platform/shared.ts` reads `navigator.userAgent`, `navigator.platform`, `navigator.maxTouchPoints` (its `userAgentData` branch is development only).
- `isOverflowElement` (`@floating-ui/utils/dom`): `/auto|scroll|overlay|hidden|clip/.test(overflow + overflowY + overflowX) && display !== 'inline' && display !== 'contents'`. Our `floating_ui_dom.js` does not export it.
- `Timeout` (`useTimeout.ts`: `create()`, `start(delay, fn)`, `clear()`) and `AnimationFrame` (`useAnimationFrame.ts`: `create()`, `request(fn)`, `cancel()`) are thin wrappers over `setTimeout` and `requestAnimationFrame`.

**shadcn** uses Base UI as is, so a shadcn page with an open dialog has exactly these styles on `html` and `body`, not a `paddingRight` on `body`.

**Repo facts the Executor builds on.**

- `origin/feat/shadcn-base-ui-1to1-parity` (one WIP commit, 2026-09-04, "wip commit soft reset later", 147 files, 56 commits behind `main`) already has the ownership half of this: `components/runtime/runtime.js` exposes `setScrollLocked(owner, locked)` over a `Set` of owners and is the only writer of `body.style.overflow` and `paddingRight`; the five scripts call it with a module-level owner object and keep their timers and guards. It is today's lock with one writer, not the Base UI lock (no gutter, no overlay branch, no foreign-lock wait), it sits inside a component lifecycle runtime that is a different topic, and its bundle ordering hack lives in `components/scripts.go`, which #614 deleted. Nothing from it is cherry-picked; the shape it found (one writer, per-owner release) is what this plan does with Base UI's own code.
- The bundle is `components/*/*.js` in lexical path order (`cmd/shadcn-templ/utils/updaters/update_scripts.go:22`, `plans/scripts-611.md`). Shared scripts are referenced through `window` at call time (`window.FloatingUIDOM` in `dropdownmenu.js:100`). A dialog rendered with `Open: true` calls `lockScroll` synchronously from `init()` while the bundle is still executing (deferred scripts run before `DOMContentLoaded`, `readyState` is `interactive`, so `init()` runs inline), so the locker must sit earlier in the bundle than its first consumer: the directory name has to sort before `contextmenu`.
- Shared files are not registry items. `registry.json` lists them in the `files` of every item that needs them (`dropdown-menu`, `context-menu`, `select`, `popover`, `combobox`, `hover-card` list both `floatingui` files with `"type": "registry:ui"`); `BuildStyleItem` (`internal/registryapi/styleitems.go:149`) reads that list and derives the `scripts` dependency from any `.js` file. `static/llms.txt` is generated from `registry.json` by `task generate-llms` and is tracked. The docs page of each component lists every shipped file as a `<ComponentSource name="..." title="components/..."/>` line (`internal/service/content/docs/components/dropdown-menu.md:30-36`).
- No CSS in the repo reads `data-tui-scroll-locked`; the only readers are the gitignored probes in `tmp/sidebar-613/`, which check `body[data-tui-scroll-locked]` and become stale with this plan (see task 1).
- `select.js` records the opening pointer type on the trigger (`trigger._tuiOpenMethod = e.pointerType`, `select.js:790`) and already uses `content._tuiOpenMethod !== "touch"` for the align mode. `dropdownmenu.js` opens on `pointerdown` (`dropdownmenu.js:594`) and on detail-0 click for the keyboard, without recording the pointer type. `contextmenu.js` opens through `openAt(content, x, y)`.
- The drawer (`drawer.js`) still runs on a native `<dialog>` opened with `show()`; its lock copy is the same code. shadcn's drawer is vaul, whose body styling is a separate 1:1 topic (`overlay-1to1` in memory); here the drawer only stops owning a lock copy.
- Test tooling: Playwright in `tmp/a11y-600/node_modules`, probes as `tmp/<topic>/*.mjs`, dev server `task dev` on 8090, previews at `/preview/<name>`, docs pages at `/docs/components/<name>`. The scripts watcher rewrites `components/scripts_bundle.go` on every JS edit; that generated file is committed with each task.

## Decisions

- **One file, one locker, literal port.** `components/baseui/scroll_lock.js` is the port of `packages/utils/src/useScrollLock.ts` under the same names: `getViewportScroller`, `isPageScrollLocked`, `hasInsetScrollbars`, `supportsStableScrollbarGutter`, `preventScrollOverlayScrollbars`, `preventScrollInsetScrollbars`, `class ScrollLocker` with `acquire`, `release`, `lock`, `unlock`, and the singleton `SCROLL_LOCKER`. The `Object.assign` style writes and the read-then-write order stay as in the source. The attribute is `data-tui-scroll-locked` on `<html>`, set exactly where Base UI sets `data-base-ui-scroll-locked`, so it exists only in the inset-scrollbar branch without gutter support, like upstream. `useScrollLock`'s effect semantics ("hold while enabled") are what each consumer does by keeping the release function for the lifetime of its open state.
- **Its four helpers live in the same file.** Base UI splits them into modules for tree shaking; the bundle has none, and a file with one consumer is a file too many. So `scroll_lock.js` also carries, each under its source name and with a comment naming the source file: `Timeout` and `AnimationFrame` (`useTimeout.ts`, `useAnimationFrame.ts`, the two small classes they are), `isOverflowElement` (`@floating-ui/utils/dom`, four lines, not exported by our `floating_ui_dom.js`), and the two platform flags the locker reads, `ios` from `platform/os.ts` and `webkit` from `platform/engine.ts`, evaluated once at load from `navigator.platform`, `navigator.maxTouchPoints` and `CSS.supports`. `ownerDocument` and `ownerWindow` are `referenceElement?.ownerDocument || document` and its `defaultView`. The rest of `platform` is not ported; nothing here needs it. When a second script needs a platform flag, that is the moment `platform.js` gets split out, not before.
- **`anchoredPopupScrollLock` is the fifth helper in the same file.** The port of `packages/react/src/utils/useAnchoredPopupScrollLock.ts`: `anchoredPopupScrollLock(enabled, touchOpen, positionerElement, referenceElement)` returns a release function (a no-op when it did not acquire). The touch rule is the source's: `positionerElement.offsetWidth >= documentElement.clientWidth - 20`, measured when called, which is after the popup is positioned.
- **The public surface is two functions.** `window.tui.scrollLock = { acquire, anchoredPopup }`; `acquire(referenceElement)` returns the release function. Nothing else is exported.
- **The directory is `baseui`**, the pendant of the `@base-ui/utils` package, the way `floatingui` is the pendant of `@floating-ui/dom`. It sorts before every consumer, so the singleton exists before the first `init()` acquires it. The ordering comment in `update_scripts.go` names both pairs.
- **Every consumer holds its own release.** `dialog.js`: `state.releaseScroll = window.tui.scrollLock.acquire(popup)` in `openDialog` when modal, called and cleared in `closeDialog` and `destroyDialog`. `drawer.js`: `dialog._tuiReleaseScroll` set in `openDrawer` under the existing `show-modal` condition, called in `cleanupClosed` and when `init` removes a drawer whose portal owner left the document. `dropdownmenu.js`, `contextmenu.js`, `select.js`: `content._tuiReleaseScroll = window.tui.scrollLock.anchoredPopup(...)` in `open`, `openAt` and the select's open, called where they call `unlockScroll` today. The five `lockScroll`, `unlockScroll`, `applyScrollLock`, `lockTimer`, `anyModalOpen` copies and the `unlockScroll()` calls inside the `MutationObserver`s of `dialog.js` and `drawer.js` are deleted. The guards `anyOpen()` in the menus and `[...allContents()].some(isOpen)` in the select disappear with them where they served only the lock.
- **Lock conditions are the source's.** Dialog: modal (today's `isModal`). Drawer: `show-modal` true (today's condition). Dropdown menu: `enabled = true` for a root menu (Base UI `modal` defaults to true, our port has no hover-opened root menus), `touchOpen` from the pointer type recorded on the trigger at `pointerdown`, the select's pattern; a keyboard open records `"keyboard"`. Context menu: `enabled = true`, `touchOpen` true when the port's long-press path opened it and false otherwise. Select: `enabled = true` (modal by default, and align mode is a subset), `touchOpen = content._tuiOpenMethod === "touch"`. Submenus stay lock-free.
- **The visible result changes, on purpose.** With gutter support the page keeps its width through `scrollbar-gutter: stable` on `html`; `body` gets no `paddingRight`. On overlay scrollbars only `overflow` on the scroller changes. That is what a shadcn page shows.
- **Registry and docs follow `floatingui`.** `registry.json`: `dialog`, `drawer`, `dropdown-menu`, `context-menu` and `select` list `components/baseui/scroll_lock.js` with `"type": "registry:ui"`, after the component's own files and before `floatingui`. `sheet`, `alert-dialog` and `command` inherit through their `dialog` dependency. The five docs pages get one `ComponentSource` line for it in the same position. `task generate-llms` regenerates `static/llms.txt`.
- **Nothing else changes.** No templ, class, markup or demo changes; no change to what opens, closes, focuses or marks `aria-hidden`.

## Tasks

### 1. Probe and failing baseline

- [x] Done

`tmp/scroll-lock/probe.mjs <chromium|webkit>` (gitignored) against the dev server. It reads the lock the way Base UI defines it: `locked` is `/hidden|clip/.test(getComputedStyle(viewportScroller).overflowY)` with `getViewportScroller` from the Context, plus `html.style.scrollbarGutter`, `body.style.paddingRight`, `body.style.overflow`, `scrollY`, and the `left` of the docs page's `main` (or the preview's first `[data-slot]` element) for shift checks. Scenarios, each asserting and exiting non-zero on failure:

- A dialog survives mutation: `/docs/components/dialog`, open the first dialog, expect locked; append and remove a `<div>` on `<body>`, expect still locked; Escape, expect unlocked after two frames.
- B dialog survives a menu: `/preview/sidebar-demo` at 750px, open the sheet, expect locked; open a dropdown inside the sheet, expect locked; Escape once (menu), expect locked; Escape again (sheet), expect unlocked.
- C no shift and no scroll: `/docs/components/dialog` scrolled to `scrollY = 300`, record `main.left` and `scrollY`, open the dialog: `main.left` unchanged, `body.style.paddingRight` empty, `window.scrollBy(0, 200)` leaves `scrollY` unchanged; close: `scrollY` back at 300, `html.style.scrollbarGutter` and every inline style the lock wrote are empty again.
- D drawer: `/docs/components/drawer`, open, expect locked; close, expect unlocked.
- E swap out: open a dialog, then remove its `_tuiPortalOwner` declaration site from the DOM (the element the dialog root's `_tuiPortalOwner` points at), expect unlocked.
- F menus and select: `/docs/components/dropdown-menu`, `/docs/components/select`, `/docs/components/context-menu`: open, locked; close, unlocked. Then in a context with `hasTouch: true`, tap-open the dropdown and the select: not locked, since the popups are narrower than the viewport.
- G stack: open a dialog, open a select inside it if a demo offers one, else open a dropdown inside the sheet as in B; close the inner popup: still locked; close the outer: unlocked.

Run it on untouched `main` in both engines, save `tmp/scroll-lock/baseline-{chromium,webkit}.log`.

Done when: the baselines show A, B and E failing (lock released while the dialog is open), C failing on `paddingRight` (today's lock pads the body), F failing on the touch case (today's menus always lock), D and G otherwise passing.

Checks: `node tmp/scroll-lock/probe.mjs chromium`, same for webkit.

### 2. `components/baseui/scroll_lock.js`

- [x] Done

The one file per Decisions, an IIFE like every component script, headed by a comment naming `useScrollLock.ts` and listing the helper sources it carries. `registry.json`, the five docs pages and `static/llms.txt` per Decisions. `update_scripts.go:22` comment extended to name `baseui` before its consumers. No consumer changes yet; the bundle carries the new file unused.

Done when: `go test ./internal/registryapi/` passes (the invariant test compiles the new file through the JS inliner), `go run ./cmd/docs` serves `/docs/components/dialog` with the new `ComponentSource` block, and `window.tui.scrollLock.acquire(null)` on any page locks after a frame and the returned function unlocks, observed with the probe's `locked` reader in both engines.

Checks: `go test ./internal/registryapi/`, `git diff --check`, the manual `acquire` check in the console of both Playwright engines.

### 3. Dialog and drawer hold their own release

- [ ] Done

`dialog.js` and `drawer.js` per Decisions: acquire in open, release in close and destroy, copies and observer calls deleted, `anyModalOpen` deleted. The dialog's `MutationObserver` still calls `init()`; the drawer's still calls `init()` and `syncInert()`.

Done when: probe scenarios A, B (sheet part), C, D, E and G pass in both engines; nothing else in the probe regresses.

Checks: `node tmp/scroll-lock/probe.mjs chromium`, same for webkit, `node tmp/sidebar-613/probe.mjs chromium` after changing its `scrollLocked` reader to the Base UI definition (it is gitignored; adjust and rerun, do not commit), `go build ./...`, `git diff --check`.

### 4. Menus and select through the anchored popup rule

- [ ] Done

`dropdownmenu.js` records `trigger._tuiOpenMethod` on `pointerdown` and `"keyboard"` on the detail-0 click, copies it to the content on open like the select does; `contextmenu.js` passes its long-press state; `select.js` passes `content._tuiOpenMethod === "touch"`. All three acquire in open and release in close, copies and guards deleted.

Done when: every probe scenario passes in both engines, including F's touch case.

Checks: `node tmp/scroll-lock/probe.mjs chromium`, same for webkit, `go test ./internal/registryapi/`, `go build ./...`, `git diff --check`, `grep -rn "data-tui-scroll-locked\|lockScroll\|unlockScroll" components/*/*.js` lists only `components/baseui/scroll_lock.js`.

## Executor log

### Task 1 (Codex, 2026-09-21)

Baseline recorded on unchanged `main` `5f9a0ab8` in `tmp/scroll-lock/baseline-{chromium,webkit}.log`; both engines exit 1 with eight failed lock expectations. Probe: `tmp/scroll-lock/probe.mjs`, scenarios A-G, computed viewport-scroller overflow, inline styles, layout and scroll positions, popup state and page errors. A loses its lock after a body mutation. B/G lose it on opening and closing the inner menu. All three standalone anchored popups in F lose it too. D and E pass. No JavaScript errors.

Evidence requires corrections to the anticipated baseline and checks (Decisions otherwise retained):

- E already releases correctly on owner removal, rather than failing. G is the same nested menu case as B and fails for the same reason. These remain regression assertions.
- This macOS headless environment uses overlay scrollbars, so C has no old body padding to expose; the no-padding assertion passes already. F touch also appears unlocked already because the observers erase the old unconditional lock. The mouse-lock assertions distinguish that false success.
- `overflow:hidden` prevents user scrolling, not `window.scrollBy`: both engines move from 300 to 500 under an intact lock. C asserts an actual wheel gesture is blocked and records the imperative scroll as a diagnostic, then restores the position before closing. Adding interception of programmatic scrolling would diverge from the supplied Base UI source.
- The original Escape sequence closes both menu and sheet with one Escape, confirmed in both engines and in `dialog.js`'s unfiltered document Escape listener. Original runs retained as `baseline-original-escape-{chromium,webkit}.log`. To honor the decision not to alter closing behavior, B/G close the inner menu through its trigger and close the sheet with Escape. Asked Axel about this separation; proceeding with the scope-preserving default. The unrelated Escape behavior remains for Planner review.
- `contextmenu.js` has no custom long-press path or state at all. Task 4 will classify the existing native contextmenu event from pointer type / the preceding pointer press without adding a new opening gesture.
- The supplied `useAnimationFrame.ts` includes a shared Scheduler, rather than only a thin native rAF wrapper. Task 2 retains that production scheduler along with AnimationFrame; React effect adapters and test-only scheduler resets are not needed.

Task 1 is checked as completed with these measured baseline corrections, not as a claim that the anticipated failure list was accurate. Probe and logs remain gitignored.

### Task 2 (Codex, 2026-09-21)

Added `components/baseui/scroll_lock.js`: the supplied useScrollLock implementation with TypeScript/React adapters removed, helper classes including the production rAF scheduler, platform flags, viewport overflow helper, and the anchored popup rule. Exported only acquire and anchoredPopup. Registry and five ComponentSource listings include the shared file before Floating UI; the bundler ordering comment documents baseui first. `task generate-llms` ran as requested; its only output delta is the previously missing Resizable entry (the generator does not list component source files).

`go test ./internal/registryapi/`, `node --check components/baseui/scroll_lock.js` and `git diff --check` pass. The normal `task dev` docs server serves the new ComponentSource block, verified by `tmp/scroll-lock/shared.mjs` in both engines. That check also passes direct acquire/release, two-owner refcount, immediate acquire/release, preserving a foreign lock, and taking over after a foreign lock clears. No consumers changed yet. The scripts watcher updated the committed bundle reference.

### Task 3 implementation (Codex, 2026-09-21)

Dialog state and drawer nodes now hold and clear their own release function on close and retirement. Removed both lock copies and observer unlock calls. Drawer replacement through a fresh portal template also releases the stale drawer, in addition to owner removal, because a retained release must cover both existing unmount paths. No focus, opening/closing, or aria-hidden semantics changed.

`go build ./...`, syntax checks and `git diff --check` pass. The sidebar probe, updated only in its gitignored computed-lock reader, passes in Chromium. Scroll-lock probe A, C, D and E pass in both engines; B/G keep the lock on menu open but still lose it on menu close. F now exposes the old unconditional touch lock. These four failures per engine are dependencies on task 4, not changes to the planned dialog/drawer behavior. Consequently the task-3 checkbox stays open until the integrated task-4 run passes (the task-3 done-when line cannot hold while the remaining three scripts still own lock copies). Logs: `tmp/scroll-lock/task3-{chromium,webkit}.log` and `task3-sidebar-chromium.log`. Watcher-generated bundle included.

## Planner review
