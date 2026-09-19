# Dropdown menu regressions on main

- **Planner**: Claude
- **Executor**: Codex
- **Status**: done

## Context

Two bugs the user hit in Safari and Firefox after `plans/a11y-600.md` landed. Both exist on `3d23ed1f`, the commit before PR #600, so neither comes from the PR or from the a11y-600 commits. The Planner verified that with three servers side by side (before the PR, after the PR, current main).

**Bug 1, the submenu is invisible and dead in every browser.** Reproduced in WebKit, Chromium and Firefox on `3d23ed1f` and on current main. Hovering "Invite users" in the first dropdown demo sets `data-open` on the sub content and positions it at x 871 to 967, but the root popup ends at x 875 and carries `overflow-x-hidden overflow-y-auto` (1:1 shadcn's `DropdownMenuContent` class string). The sub content is `position: absolute` inside that popup, so the popup clips it. `document.elementFromPoint` over a sub item returns the page underneath, a click there is an outside click and closes the menu, and hover never highlights a sub item. Screenshot in `tmp/a11y-600/sub-webkit-8092.png`: only the chevron is visible.

Cause: commit `643bc8a5` (2026-08-15, "complete Base UI state and interaction parity") switched the root positioner from `fixed` to `absolute`, which is right for the root because it is portaled to `<body>`. The same commit swept the sub content along, class `hidden fixed inset-auto left-0 top-0` became `hidden absolute ...` and `strategy: "fixed"` became `strategy: "absolute"` in `openSub`. The sub content is not portaled here, it stays nested inside the root popup, so `absolute` puts it under the popup's overflow clip. The context menu kept `fixed` for its sub content (`components/contextmenu/contextmenu.templ` line 620, `strategy: "fixed"` in `contextmenu.js`) and its submenu works: same test, `elementFromPoint` returns `context-menu-item`.

What shadcn does: `DropdownMenuSubContent` renders `DropdownMenuContent`, which is `Menu.Portal` → `Menu.Positioner` → `Menu.Popup`, so Base UI's sub popup lives in `<body>` and no ancestor overflow can clip it. This repo does not portal sub content. The context menu's `fixed` layer inside the root popup is the established pendant here and has worked since `c726ffda`.

**Bug 2, the first dropdown demo opens slowly in Safari.** Not reproduced yet. Playwright's WebKit opens it in 90 to 150 ms on every build, Firefox in about 90 ms, with no console errors and no animation frame loop. Playwright's WebKit is not Safari. Two diagnostic servers are running for the user: 8094 serves `3d23ed1f` with an on-page timing overlay in the bottom left (each step of `open()` with milliseconds since the press: lockScroll, mounted, positioned, finish, frame 1, frame 2, animationend), 8095 serves `3d23ed1f` with the body scroll lock disabled. The overlay says where the time goes; 8095 tests the one Safari-specific suspect, `body { overflow: hidden }` on a page with a sticky header, which forces a full relayout and repaint in Safari. A task for this bug is added once the user reports the overlay numbers.

Test tooling from `plans/a11y-600.md` still applies (`tmp/a11y-600/`, gitignored). New scripts there: `sub3.mjs <webkit|chromium> <port>` reports popup rect, popup overflow, sub rect, sub position and what `elementFromPoint` hits over the first sub item, and saves a screenshot; `ctxsub.mjs` is the same for the context menu.

## Decisions

- **Sub content goes back to `fixed`, the context menu's pattern.** Restore the state before `643bc8a5` for the submenu only: class `hidden fixed inset-auto left-0 top-0 data-closed:fill-mode-forwards` and `strategy: "fixed"` in `openSub`. The root positioner stays `absolute`. Portaling the sub content to `<body>` would be the literal Base UI structure, but it means template lifting, ownership sweeps and a second positioner lifecycle for a result the `fixed` layer already gives, and the context menu proves the pattern in this codebase. Not chosen.
- **Known limit, accepted.** A `fixed` sub content does not scroll with a root popup that overflows its `max-h`. Base UI repositions its portaled sub on ancestor scroll. Nobody scrolls a root menu with a submenu open in practice, and the context menu has lived with the same limit since July.
- **No other change in the dropdown.** Bug 2 gets its own task after diagnosis.

## Tasks

### 1. Submenu back on a fixed layer

- [x] Done

`components/dropdownmenu/dropdownmenu.templ`: in `SubContent`, change `hidden absolute inset-auto left-0 top-0` to `hidden fixed inset-auto left-0 top-0`. `components/dropdownmenu/dropdownmenu.js`: in `openSub`, `strategy: "absolute"` becomes `strategy: "fixed"`. Nothing else. Do not touch the root positioner. Check the sub content's comment above the class string still describes a fixed layer and fix it if the sweep changed the wording.

Done when: `tmp/a11y-600/sub3.mjs` reports `subPosition: "fixed"` and `hitOnSubItem: "dropdown-menu-item"` in webkit and chromium against a server running the change, the sub items highlight on hover and a click on "Email" closes the menu through the item, the screenshot shows the submenu next to "Invite users", and `a11y.mjs` still reports 30 of 30 in both engines.

Checks: `sub3.mjs` webkit and chromium, `a11y.mjs` webkit and chromium, `go test ./components/dropdownmenu/...`.

## Executor log

### Task 1

Restored fixed positioning for SubContent in the template and openSub's Floating UI strategy, exactly two source lines. The root positioner remains absolute. The existing comment already described a fixed layer. The running watcher updated dropdownmenu_templ.go; no generator or minifier was invoked.

Verified against a temporary server on 8096 (8091 and 8092 were occupied by existing diagnostic servers). In both WebKit and Chromium, sub3.mjs reports subPosition="fixed" and hitOnSubItem="dropdown-menu-item". Visually inspected both screenshots: the submenu appears beside Invite users without clipping. Additional interaction checks confirm hover focuses Email, pointer hit testing resolves to the item, and Email receives a click while its root menu is still open before the item handler closes it. a11y.mjs passes 30/30 in each engine; go test ./components/dropdownmenu/... and git diff --check pass.

Evidence remains in tmp/a11y-600/menu-sub-{webkit,chromium}.log, menu-a11y-{webkit,chromium}.log and sub-{webkit,chromium}-8096.png. Stopped the temporary 8096 server. Existing diagnostic servers and the user's pending plans/a11y-600.md review changes were left untouched. Ready for Planner review; bug 2 still awaits the user's real-Safari timing and scroll-lock comparison.

## Planner review

- Task 1 (`b51d0453`): accepted. Re-checked by the Planner against the user's own dev server on 8090 after the commit: `sub3.mjs` in WebKit reports `subPosition: "fixed"` and `hitOnSubItem: "dropdown-menu-item"`. The two lines match the context menu exactly. Diagnostic servers 8091 and 8093 ran older commits without the fix and are stopped so nobody tests the wrong build; 8092 (before the PR), 8094 (timing overlay) and 8095 (no scroll lock) stay up for bug 2.

- Bug 2: parked by the user on 2026-09-19, no task. Evidence gathered: the on-page overlay in real Safari showed the JavaScript open path finishing in 78 ms and the enter animation ending at 194 ms while the menu still appeared with a delay of about a second; in other runs one forced layout after the scroll lock cost 283 ms and `computePosition` another 300 ms, both wildly variable between clicks. Playwright's WebKit, Firefox and Chromium open the same menu in 30 ms with a 2 ms relayout, and a private Safari window is just as slow, so extensions are not the cause. The context menu on the same page is instant in Safari, so the page weight alone (20k elements, 18k of them shiki spans, 2.2 MB CSS with 1139 `:has()` rules) does not explain it. Whatever Safari does differently sits in rendering after our DOM changes, not in component code. Diagnostic servers and worktrees are removed. If this is picked up again, the first step is Safari's Timelines panel on a real click, or the "Allow JavaScript from Apple Events" switch so the Planner can measure inside Safari directly.

All tasks accepted. Remaining outside the plan: the push.
