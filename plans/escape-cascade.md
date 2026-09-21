# escape-cascade: Escape closes the innermost layer only, the useDismiss pendant

- **Planner**: Claude
- **Executor**: Codex
- **Status**: ready

## Context

Found during the scroll-lock review: one Escape closes every open layer at once. Measured on `main` (`tmp/scroll-lock/escape.mjs`, Chromium): sidebar sheet open, dropdown open inside it, focus on a menu item, one Escape closes the dropdown and the sheet. shadcn closes the dropdown and keeps the sheet.

**What Base UI does** (`packages/react/src/floating-ui-react/hooks/useDismiss.ts` on `mui/base-ui` master, copy in `tmp/escape/baseui/`, gitignored). Every dismissable primitive (Dialog, Popover, Menu, Select, Tooltip, PreviewCard, Combobox, ContextMenu through Menu) calls `useDismiss`, which installs one function, `closeOnEscapeKeyDown`, in three places:

1. as `onKeyDown` in the floating element's props (`getFloatingProps`, line 780), so the popup handles Escape when the focus is inside it;
2. as `onKeyDown` in the reference props (`getReferenceProps`, line 771), so the trigger or input handles Escape when the focus sits there (tooltip, combobox, closed select);
3. as a `document` keydown listener (line 714), the fallback for focus anywhere else.

The function (lines 204 to 234): return unless open, enabled and `key === 'Escape'`; return while an IME composition is pending; return when a tree child is open and does not bubble (`hasBlockingChild('__escapeKeyBubbles')`: nested dialogs, submenus); `setOpen(false, escapeKey)`; `event.preventDefault()` unless the change was canceled; `event.stopPropagation()` unless `bubbles.escapeKey`. The layering is nothing more than that `stopPropagation`: the innermost element that contains the focus handles the key during bubbling, stops it, and the outer layers' document listeners never run. No registry of layers, no z-order, no cross-component knowledge.

Per component: Menu passes `bubbles: { escapeKey: closeParentOnEsc && parent.type === 'menu' }` (`MenuRoot.tsx:472`), so Escape in a submenu bubbles to the parent menu and the whole tree closes, the default. Dialog (`useDialogRoot.ts:29`), Popover (`PopoverRoot.tsx:238`), Select (`SelectRoot.tsx:353`), PreviewCard (`PreviewCardRoot.tsx:97`), Tooltip (`TooltipRoot.tsx:261`, plus `referencePress`) and Combobox (`AriaCombobox.tsx:1333`, `enabled: !disabled && !inline`) use the defaults: no bubbling, stop propagation. `event.defaultPrevented` is not consulted for Escape, only for outside presses.

**Ours.** Nine scripts listen for Escape, all on `document`, all in the bubble phase except the drawer, none stops propagation:

- `dialog.js:537`: topmost of `openStack`, IME guard, `preventDefault`, `requestOpenChange(popup, false)`.
- `drawer.js:521`: capture phase, returns on `defaultPrevented` or when any `[data-tui-portal][data-open]` popup exists (a workaround for exactly this bug, one direction only), then the deepest open drawer (`hasOpenNested`).
- `dropdownmenu.js:685` `requestCloseAll(true)`; `contextmenu.js:578` `closeAll()`; `select.js:900` closes every open content and refocuses the trigger; `popover.js:316` and `tooltip.js:242` `requestCloseAll()`; `hovercard.js:220` closes every content; `combobox.js:644` `closeAll()`.
- `chart.js:1964` is Recharts' `TooltipBoundingBox` Escape, not a dismissable layer; untouched.

Document listeners fire in registration order, which is bundle order (`chart`, `combobox`, `contextmenu`, `dialog`, `drawer`, `dropdownmenu`, `hovercard`, `popover`, `select`, `tooltip`), so the dialog closes before the dropdown inside it even sees the key. Element-level listeners are the fix and need no ordering: the DOM runs them before any document listener.

**Repo facts.** Every script lifts or portals its content to `<body>` in one place (`ensureDialog`, `ensureDrawer`, `liftTemplates` or `portal`) and walks its triggers in `init`; both are the places to attach element listeners once. Each script's `requestOpenChange` dispatches a cancelable `*-open-change` event and honours a `-controlled` attribute; `contextmenu.js` returns whether it closed, the others return nothing. Previews for the probe: `/preview/sidebar-demo`, `/preview/drawer-nested`, `/preview/dialog-demo`, `/preview/popover-form`, `/preview/combobox-demo`, `/preview/tooltip-keyboard`, `/preview/context-menu-demo`, `/preview/context-menu-submenu`, `/preview/select-demo`, all 200 on the dev server. Playwright: `tmp/a11y-600/node_modules`.

## Decisions

- **One function per script, `closeOnEscapeKeyDown(event)`, the body of Base UI's.** Return unless `event.key === "Escape"` and the script has an open instance that owns the event (see next point); return while composing where the script has the IME guard (dialog keeps its); return when a blocking child is open (dialog: a nested open dialog, which `openStack`'s topmost already expresses; drawer: `hasOpenNested`; menus: none, submenus bubble); close through `requestOpenChange`; `event.preventDefault()` when the close was accepted; `event.stopPropagation()` always, since no port sets `bubbles`. `requestOpenChange` returns the accepted flag in every script so the `preventDefault` condition can be literal.
- **Installed in Base UI's three places.** On the popup or content element and on the reference element (trigger, or the input for the combobox) with `addEventListener("keydown", closeOnEscapeKeyDown)`, once per element at the place the script lifts the content and walks the triggers, guarded by a `WeakSet` so a re-init never doubles a listener. Plus the existing `document` listener, now calling the same function. Which instance owns the event: for the element listeners, the instance whose popup or reference is `event.currentTarget`; for the document listener, every open instance, in order, like Base UI's per-instance document listeners (a dialog checks its `openStack` topmost, a drawer its deepest open, the menus their open content).
- **The drawer loses its workaround.** Bubble phase like the others, no `defaultPrevented` check, no `OPEN_POPUP_SELECTOR`. A popup open inside the drawer with focus in it stops the key at its own element; that is the mechanism, not a selector.
- **Submenus close the tree.** `closeParentOnEsc` defaults to true, so Escape inside a submenu closes the root menu, which is what `requestCloseAll` and `closeAll` do today. Submenu content sits inside the root content in our DOM, so the root's element listener sees the key.
- **Select refocus stays.** Base UI returns focus to the trigger when the select closes; `select.js` does that on Escape today and keeps it.
- **Tooltip and hover card keep the document fallback semantics of the source.** An Escape that reaches `document` closes every open tooltip and hover card; an Escape stopped inside a menu or dialog does not. That is Base UI's behaviour and it follows from the mechanism, nothing to special-case.
- **Nothing else changes.** No templ, class, markup, focus, outside-press, or scroll-lock changes. `chart.js`, `command.js` (dismissed through its dialog) and `sidebar.js` (its sheet is a dialog) are untouched.

## Tasks

### 1. Probe and failing baseline

- [x] Done

`tmp/escape/probe.mjs <chromium|webkit>` (gitignored). Every Escape is sent by dispatching `new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })` on `document.activeElement` from `page.evaluate`, which returns `false` when the event was prevented; a probe listener registered on `document` after page load records whether the event reached the document. Both go into every assertion: the layer that handles Escape prevents it and stops it before the document. Scenarios:

- A `/preview/sidebar-demo` at 750px: open the sheet, open the dropdown inside it, focus is on a menu item. Escape: dropdown closed, sheet open, focus on the dropdown trigger, prevented, not at document. Escape: sheet closed, prevented.
- B `/preview/drawer-nested`: open outer, open nested. Escape: nested closed, outer open. Escape: outer closed.
- C `/preview/dialog-demo`: open. Escape: closed, prevented, not at document.
- D `/preview/popover-form`: open the popover, `document.activeElement.blur()`. Escape: closed (document path, reached the document).
- E `/preview/combobox-demo`: focus the input, ArrowDown opens the list. Escape: list closed, input still focused and its value kept, prevented, not at document.
- F `/preview/tooltip-keyboard`: Tab to the trigger, tooltip open. Escape: tooltip closed, prevented, not at document.
- G `/preview/context-menu-demo`: right click opens. Escape: closed. `/preview/context-menu-submenu`: open the submenu, focus inside it. Escape: root and submenu closed.
- H `/preview/select-demo`: open with the keyboard. Escape: closed, trigger focused, prevented, not at document.

Save `tmp/escape/baseline-{chromium,webkit}.log` from untouched `main`.

Done when: the baseline shows A failing (both layers close) and the "not at document" assertions failing everywhere, while B, D and the close assertions of C, E, F, G, H pass.

Checks: `node tmp/escape/probe.mjs chromium`, same for webkit.

### 2. Dialog and drawer

- [ ] Done

`dialog.js`: `closeOnEscapeKeyDown` per Decisions, attached to `state.popup` in `ensureDialog` and to every trigger of that popup when `init` walks them; the document listener's Escape branch calls it; `requestOpenChange` returns the accepted flag. `drawer.js`: same on the `<dialog>` element in `ensureDrawer` and on its triggers; the capture-phase listener becomes a bubble-phase call of the same function without the two guards.

Done when: probe A's sheet half, B and C pass in both engines, including the propagation assertions; nothing else in the probe regresses beyond the baseline.

Checks: `node tmp/escape/probe.mjs chromium`, same for webkit, `go build ./...`, `git diff --check`.

### 3. Menus, select, popover, hover card, tooltip, combobox

- [ ] Done

The same function in `dropdownmenu.js`, `contextmenu.js`, `select.js`, `popover.js`, `hovercard.js`, `tooltip.js` and `combobox.js`, attached to each lifted content and to each trigger (the input for the combobox) at lift and `init`, the document listeners calling it, `requestOpenChange` returning the accepted flag where it does not yet.

Done when: every probe scenario passes in both engines, and `grep -n '"Escape"' components/*/*.js` shows one Escape branch per script (plus the chart's), each inside `closeOnEscapeKeyDown`.

Checks: `node tmp/escape/probe.mjs chromium`, same for webkit, `node tmp/scroll-lock/probe.mjs chromium` (the lock releases on Escape are unchanged), `go build ./...`, `git diff --check`.

## Executor log

### Task 1 (Codex, 2026-09-21)

Added gitignored `tmp/escape/probe.mjs` with all eight scenarios, synthetic cancelable Escape dispatched on the active element, a post-load document observer, popup state, focus assertions, and page-error collection. Baselines on unchanged implementation at `1d1ee416` (Scroll-Lock Planner review committed separately): `tmp/escape/baseline-{chromium,webkit}.log`. Both engines exit 1 with 21 failed expectations. A closes menu and sheet together and loses the menu trigger focus; all element-path document-propagation assertions fail. B's nested/outer close order and the close assertions for C-H pass. Most primitives also fail preventDefault today; D correctly reaches document but does not prevent default. No page errors. No implementation changes in this task.

## Planner review
