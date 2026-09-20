# PR #612 chart: gaps, fixed y ticks, dashed lines, per-row dots, hidden series

- **Planner**: Claude
- **Executor**: Codex
- **Status**: ready

## Context

PR #612 (miguelcsilva, `chart-line-gaps-ticks-dashes`, 185 added, 60 removed in `chart.js`, `chart.templ` and the generated `chart_templ.go`) adds five things to the chart. The author needed them locally and offers to tweak. The Planner built and vetted `components/chart` on the PR head: it compiles, the generated file comes from the same templ version (v0.3.1001) and the running watcher would regenerate it anyway. `maintainerCanModify` is true on the PR, so commits can go straight to the fork branch.

**Why the PR exists.** The chart is a port of shadcn's `chart.tsx` plus the parts of Recharts it needs, driven by the shadcn demos. Four things Recharts supports and the demos never use were therefore missing: a data row without the series key draws as zero instead of a gap, the value axis always starts at zero, a line cannot be dashed and a series cannot be hidden. All four are Recharts props that shadcn passes straight through (`connectNulls`, `ticks` and `domain`, `strokeDasharray`, `hide`), so they belong to the 1:1 surface even without a demo. The fifth thing, dots on chosen rows, has no prop in Recharts, see below.

**What the Planner verified.** Every claim below was read in the Recharts 2.15.4 sources, the version whose internals the renderer ports (react-smooth clock, `prevPointsDiffFactor`, `isAnimationFinished`). shadcn v4 pins recharts 3.8.0 today; the semantics named here are unchanged there (`connectNulls: false`, `hide: false`, `filterNull: true`, `getStrokeDasharray` still merges the pattern into the sweep).

1. **Gaps, right direction, incomplete.** Recharts with `connectNulls` off: `Curve.getPath` hands d3 a `defined` predicate, so the path restarts after every null. `Line.renderDots` builds a `Dot` per point and `Dot` renders nothing when `cy` is not a number. `Tooltip` has `filterNull: true` and drops every payload entry whose value is null, and `TooltipBoundingBox` sets `visibility: hidden` when the payload is empty. The active dot goes through the same `Dot`, so none at a gap. `getDomainOfDataByKey` filters non numbers, so a null never touches the domain. The PR covers the line curve (`gappedPath`), the dots, the tooltip row and the active dot. It misses three things: `Area` has the same `connectNulls: false` but the PR keeps drawing an area's missing value as zero; a line `LabelList` still prints `0` at a gap where Recharts' `Label` returns null for a nil value; and the tooltip stays visible with zero rows. Stacked series: Recharts' d3 stack coerces null to 0, so a gap in a stacked area or bar is a zero there too, and a bar of height 0 is not rendered by `Rectangle`, visually what the port does today. `Gaps` sits on `ModelSeries` for every kind already, which is right.
2. **Fixed y ticks, not 1:1.** In Recharts `ticks` only fixes the tick values. The domain is still the default `[0, 'auto']` (`getDefaultDomainByAxisType`), extended so it contains every specified tick (`detectReferenceElementsDomain`, last argument `specifiedTicks`) and then rounded by `getNiceTickValues` because the domain has an `auto` end (`getTicksOfScale`). The axis still starts at zero unless `domain` is set as well. `domain` itself is a small grammar, `parseSpecifiedDomain`: each end is a number, `'auto'`, `'dataMin'`, `'dataMax'`, or `'dataMin - 100'` and `'dataMax + 100'`; a number is not a hard bound, the data domain wins on the outside unless `allowDataOverflow` is true; an unparseable string keeps the data end. A domain without an `auto` end takes the second branch of `getTicksOfScale`, `getTickValuesFixedDomain`. The PR makes `Ticks` imply `domain = [ticks[0], ticks[last]]`, which is none of these. Related pre-existing gap: `YAxisProps.TickFormatter` is silently ignored on a numeric y axis (only a vertical layout reads it), because the nice ticks are computed in the browser by `niceTickValues` and the formatter is a Go func. The two never meet. `Model.DomainMax` does not exist in Go, so the `m.domainMax` branch in `renderCartesian` is dead code.
3. **Dashed lines, prop right, animation not.** `Line.getStrokeDasharray(length, totalLength, lines)` repeats the user's pattern up to the drawn length and appends the rest as empty, so the dashes are visible while the line draws in. The PR lets the sweep replace the pattern during the entrance and applies the pattern only afterwards.
4. **Dots on chosen rows, no Recharts prop.** Recharts does this through the function form of `dot`, a render prop receiving `payload` and `index` and returning an element or null. The repo already encodes the demos' render props as `DotProps` fields (`DataFill`, `Icon`) and mirrors Recharts function props as Go funcs (`TickFormatter`, `LabelList.Formatter`). `Indices []int` is a new shape with no upstream pendant. A literal port is impossible: React returns UI from that function, Go sends data to the browser. The closest honest form is a yes or no per row next to the existing `Icon`.
5. **Hidden series, right need, wrong mechanism.** Recharts `hide` is a prop on `Line`, `Area` and `Bar`: the item renders nothing (`render` returns null), its tooltip row is filtered (`Tooltip` drops `entry.hide`), its values leave the domain (`getAxisMapByAxes` filters `!itemHide`) and the stacks (`getStackGroupsByAxisId` skips `hide`), while the legend keeps the entry (`getLegendProps` does not look at `hide`, and shadcn's `ChartLegendContent` ignores `inactive`). The PR instead reads a `data-tui-chart-hidden` attribute on the container at render time through a `MutationObserver`, adds `data-key` to every line group, keeps hidden values in the domain and handles lines only. The author's reason, "so a legend can toggle a line without re-rendering the model", is not how Recharts works either: `hide` changes on a React re-render. The templ pendant of that re-render is a server swap, or the SSR variants `chart_bar_interactive` already switches through `data-tui-chart-series-panel`.
6. **Curve constants, fine.** `CurveMonotone` and `CurveLinear` name curves the renderer draws. It also draws `"step"`, which the linear and step demos spell as string literals.
7. **Housekeeping.** The added struct fields (`YTicks`, `YTickLabels`, `Gaps`, `StrokeDasharray`, `Indices`) are not column aligned with their neighbours.

Test tooling: `tmp/` is gitignored, Playwright and browsers live in `tmp/a11y-600/` from `plans/a11y-600.md`. The dev server on 8090 serves `/docs/components/chart`, which loads `chart.js`; `chart.js` re-runs `init` on every body mutation, so a fixture injected into that page boots like a real chart. The Go half gets a tracked test in `components/chart/chart_test.go`, the pattern of `components/button/button_test.go`.

## Decisions

- **The work lands on the PR branch.** `gh pr checkout 612`, one commit per task on top of the author's commit, messages `chart-612 N: ...`, pushed to the fork branch (`maintainerCanModify` is true). The user merges the PR and answers the author. No rewrite of the author's commit.
- **The value scale moves to Go.** Domain and ticks of the cartesian value axis are computed in `buildModel` and shipped as `Model.Domain`, `Model.Ticks` and `Model.TickLabels`; `TickFormatter` formats every label. The browser only draws. This is where the Recharts logic runs anyway, on the data, before render, and it is the only way a Go formatter reaches the ticks. `chart.js` keeps `niceTickValues` for the radar, which has no axis props and is out of scope. Existing charts must render byte for byte the same axis labels and curves, the probe proves it against a baseline taken before the change.
- **`Ticks`, `Domain` and `AllowDataOverflow` on `YAxisProps` and `XAxisProps`, the numeric axis reads them.** `Ticks []float64` is Recharts' `ticks`. `Domain []any` is Recharts' `domain` with the same grammar, each entry a float64 or one of the strings above, exactly two entries or nil (any other length panics with a message naming the prop). `AllowDataOverflow bool` is Recharts' prop with its default false. The functions `parseSpecifiedDomain`, `getNiceTickValues`, `getTickValuesFixedDomain` and the ticks fold of `detectReferenceElementsDomain` come over literally under those names; no function form of `domain`, Go has no way to ship it.
- **Gaps stay and cover every non stacked cartesian series.** A row whose value under the key is nil is a gap: line and area curves break there, no dot, no label, no tooltip row, no active dot, and the value stays out of the domain. Stacked series treat a gap as zero, like d3 stack. A tooltip with zero rows stays hidden, like `hasPayload`.
- **`LineProps.StrokeDasharray` stays, the animation ports `getStrokeDasharray`.** `Line.getStrokeDasharray` and `Line.repeat` come over literally; the existing sweep is `generateSimpleStrokeDasharray`.
- **`DotProps.Indices` becomes `DotProps.Show func(index int, row Datum) bool`.** The Go mirror of the `dot` render prop's inputs and its null return; nil draws every row. The model carries `DotModel.Shown []bool` next to `Fills`.
- **`Hide bool` on `LineProps`, `AreaProps` and `BarProps`, no attribute protocol.** `ModelSeries.Hidden` is set in `buildModel`. A hidden series draws nothing (curve, area, bars, dots, labels, active dot), has no tooltip row, stays out of the domain and the stacks, and keeps its legend entry. `hiddenKeys`, the `MutationObserver` on the container and the `data-key` attribute do not land. A page that toggles a series re-renders the chart with the prop or swaps SSR variants.
- **Curve constants**: `CurveLinear`, `CurveMonotone`, `CurveStep` next to `CurveNatural`. The demos keep their literals.
- **Nothing else changes in the chart.** No class, markup or demo changes. New fields align with their neighbours. Go code for the scale goes into `components/chart/scale.go`, one file, one job. The watcher regenerates `chart_templ.go`; nobody runs `templ generate`.

## Tasks

### 1. Test scaffolding and baseline

- [x] Done

`components/chart/chart_test.go`: a helper that renders a `Container` with a config, data and children to a string and extracts the JSON model from `<script type="application/json" data-tui-chart-model>`. One first test on the existing behaviour: a line chart with three rows yields three `values` and no `gaps` key. `tmp/chart-612/fixture/main.go` (gitignored): a `package main` that renders the fixtures the tasks below name to `tmp/chart-612/fixture.html`, one `<section id="...">` per fixture. `tmp/chart-612/probe.mjs <webkit|chromium> [out.json]`: opens `http://localhost:8090/docs/components/chart`, records for every chart already on the page its y axis tick texts and the `d` of its first `.recharts-line-curve`, `.recharts-area-curve` or the first bar path, then injects `fixture.html` at the end of `main`, waits for each fixture's `svg.recharts-surface`, and prints per fixture the `d` attributes of `.recharts-line-curve` and `.recharts-area-curve`, the count of `.recharts-line-dots circle`, the y axis tick texts, the `stroke-dasharray` of every line curve, the legend entry count, and, after hovering the panel at a given category index, the tooltip's row texts and whether the tooltip wrapper is visible. With `out.json` it writes everything to that file. Run it once on the untouched PR head: `tmp/chart-612/baseline-chromium.json` and `baseline-webkit.json`.

Done when: `go test ./components/chart/` passes, the two baseline files exist and contain every demo on the page.

Checks: `go test ./components/chart/`, `node tmp/chart-612/probe.mjs chromium tmp/chart-612/baseline-chromium.json`, same for webkit.

### 2. The value scale in Go

- [x] Done

`components/chart/scale.go`: literal ports of recharts-scale `getDigitCount`, `getFormatStep`, `getNiceTickValues` and `getTickValuesFixedDomain` under those names, plus Recharts' `parseSpecifiedDomain`. `buildModel` computes the cartesian value axis once: data domain over the visible, non gap values (stacked sums when stacked, `[0, 1]` for `expand`, like `domainTicks` today), default specified domain `[0, "auto"]`, nice ticks, `Model.Domain [2]float64`, `Model.Ticks []float64`, `Model.TickLabels []string` formatted by the numeric axis' `TickFormatter` or `fmt` like `fmtF` today (three decimals, trailing zeros dropped). `chart.js`: `renderCartesian` reads `m.domain`, `m.ticks` and `m.tickLabels` instead of calling `domainTicks`, `domainOf` or the dead `m.domainMax` branch; the morph pin uses `m.domain`. Remove what becomes unused (`Model.DomainMin`, `domainOf`, the `m.domainMax` branch); `domainTicks` and `niceTickValues` stay for the radar. `chart_test.go`: the model of a two series line chart carries the ticks `niceTickValues` produced before (take the expected numbers from the baseline).

Done when: `probe.mjs` output for every demo on the page is identical to the baseline in both engines (tick texts and `d` strings), and the y axis of `chart_area_axes` still shows three ticks.

Checks: `go test ./components/chart/`, `probe.mjs` chromium and webkit diffed against the baselines, `git diff --check`.

### 3. Ticks, Domain and AllowDataOverflow

- [x] Done

`chart.templ`: `Ticks []float64`, `Domain []any` and `AllowDataOverflow bool` on `YAxisProps` and `XAxisProps`, doc comments naming the Recharts props and the grammar. `buildModel` takes them from the numeric axis (YAxis in the default layout, XAxis when vertical): specified ticks extend the data domain (the ticks fold of `detectReferenceElementsDomain`), `parseSpecifiedDomain(domain, dataDomain, allowDataOverflow)`, then `getNiceTickValues` when an end is `"auto"` (or no domain given), else `getTickValuesFixedDomain`; drawn ticks are `Ticks` when set, else the computed ones; the scale domain spans the drawn ticks in the `auto` case like `getTicksOfScale` does (`scale.domain([min(ticks), max(ticks)])`) and the parsed domain otherwise. A `Domain` of length other than 0 or 2 panics with a message naming the prop. Fixtures `ticks`: a line living between 180 and 220 with `Ticks` only, `Domain: []any{"dataMin", "dataMax"}` only, `Domain: []any{100, 200}` with data reaching 220 without and with `AllowDataOverflow`, `Domain: []any{"dataMin - 10", "dataMax + 10"}`, and `Ticks` together with `Domain`.

Done when: `probe.mjs` shows for `Ticks` only a y axis whose lowest label is `0` and whose labels are the given ticks; for the `dataMin`/`dataMax` fixture an axis from 180 to 220; for `[100, 200]` without overflow an axis top at or above 220 and with overflow a top of 200; for the `- 10`/`+ 10` fixture 170 to 230; for `Ticks` with `Domain` the given labels on the given domain. Every demo still matches the baseline.

Checks: `go test ./components/chart/` with a test per fixture, `probe.mjs` in webkit and chromium, `git diff --check`.

### 4. Gaps

- [x] Done

`chart.templ`: `modelSeries` records `Gaps []bool` where `d[key] == nil` (the PR's diff); the data domain from task 2 skips gap rows. `chart.js`: the PR's `isGap`, `gappedPath` and the skips in the dot loop, `tooltipHTML` and `showActiveDots`. Extend: the line label loop skips gap rows; the non stacked area branch draws one subpath per run for both `recharts-area-area` and `recharts-area-curve` (an `areaPathBetween` per run, the same run loop as `gappedPath`); stacked branches ignore `gaps`. `positionTooltip` keeps the wrapper hidden when `tooltipHTML` produced no rows for the index and shows it again when it does. Fixture `gaps`: three lines, one row missing the first key, one row missing every key; the same data as an area chart.

Done when: `probe.mjs` shows two `M` in the gapped line's `d` and in the gapped area's `d`, one dot less than rows for that line, no label at the gap, the tooltip at the partial row lists two rows, the tooltip at the empty row is not visible, and the demos match the baseline. `chart_test.go` asserts `gaps` for the partial row and its absence for a full series.

Checks: `go test ./components/chart/`, `probe.mjs` in webkit and chromium, `git diff --check`.

### 5. Dashed lines with the Recharts entrance

- [x] Done

`chart.templ`: `LineProps.StrokeDasharray string` into `ModelSeries.StrokeDasharray` (the PR's diff). `chart.js`: port `Line.repeat` and `Line.getStrokeDasharray` literally under those names; in the line branch, when `alpha < 1` and the series has a pattern, the dash attribute is `getStrokeDasharray(total * alpha, total, pattern)`, otherwise the existing sweep; when `alpha >= 1` the pattern verbatim. Fixture `dashes`: one dashed line `"4 4"`.

Done when: `probe.mjs` shows `stroke-dasharray="4 4"` after the entrance, and a screenshot taken 300 ms after injection, mid entrance, shows dashes on the drawn part instead of a solid sweep.

Checks: `go test ./components/chart/`, `probe.mjs` chromium, the screenshot.

### 6. Dot predicate

- [x] Done

`chart.templ`: `DotProps.Indices` is replaced by `DotProps.Show func(index int, row Datum) bool`; `buildModel` fills `DotModel.Shown []bool` when set, next to the `Fills` loop. `chart.js`: the dot loop skips rows where `s.dot.shown` is set and false; the PR's `indices` check goes. Fixture `dots`: a line whose dots show on rows 1 and 3 only.

Done when: `probe.mjs` counts two dots for that line, and `chart_line_dots` on the demo page matches the baseline.

Checks: `go test ./components/chart/`, `probe.mjs` chromium.

### 7. Hidden series

- [x] Done

`chart.templ`: `Hide bool` on `LineProps`, `AreaProps`, `BarProps` into `ModelSeries.Hidden`; the data domain from task 2 skips hidden series. `chart.js`: remove `hiddenKeys`, `isHidden` reading `m.hidden`, the container `MutationObserver` and the `data-key` attribute; `expandValues` and the stack bases skip hidden series; the line, area and bar branches push placeholder geometry for a hidden series (so indices stay aligned, as the PR does) and draw nothing; `tooltipHTML` and `showActiveDots` skip `s.hidden`; the legend is untouched. Fixture `hidden`: three lines with one hidden, a stacked bar chart with one hidden series.

Done when: `probe.mjs` shows two line curves and three legend entries for the line fixture, the y axis top follows the two visible lines, and the stacked bars have two segments per row. No `data-tui-chart-hidden` or `data-key` remains in `components/chart`.

Checks: `go test ./components/chart/`, `probe.mjs` in webkit and chromium, `grep -rn 'data-tui-chart-hidden\|data-key' components/chart` is empty, `git diff --check`.

### 8. Curve constants

- [ ] Done

`chart.templ`: `CurveLinear`, `CurveMonotone`, `CurveStep` in one const block with `CurveNatural`.

Done when: `go build ./...` passes and `go vet ./components/chart/` is clean.

Checks: `go build ./...`, `go vet ./components/chart/`.

## Executor log

### Task 1

- Added tracked `components/chart/chart_test.go`: renders Container/root/children and decodes the JSON payload; three values and omitted gaps verified.
- Added local fixture renderer and Chromium/WebKit probe under `tmp/chart-612/`. Both baseline JSON files contain all six documentation charts plus 68 registry demos. The galleries use iframes, so the probe injects server-rendered registry demos at a fixed 600px width; this additionally covers `chart-area-axes` (0, 300, 600) and the line demos absent from the docs page.
- `go test ./components/chart/` and both browser runs passed on unchanged PR component sources. The referenced button test is absent on this PR branch; the helper uses templ's public child-rendering API.
- Existing development watchers are running. No generator or minifier was run manually. Plan status remains Planner-owned per README.


### Task 2

- Moved the cartesian value domain, ticks and formatted labels into `scale.go` / `buildModel`; removed the old YTicks/YTickLabels/DomainMin model fields and dead browser domainOf/domainMax paths. Bars and morph targets read the shipped domain; radar retains its existing scale.
- Added the numeric-axis formatter test (two series, ticks 0/8/16/24/32), and escaped formatted SVG tick text.
- `go test ./components/chart/`, JS syntax and `git diff --check` passed. Both browser captures match all baseline tick texts and paths, including the three area-axis ticks.
- Implementation note: the upstream control flow is ported with float64 arithmetic, preserving the existing JS renderer's arithmetic rather than adding Decimal.js-equivalent dependencies. Single-valued and reversed intervals follow recharts-scale; expand retains the existing evenly spaced 0..1 ticks.
- The watcher regenerated `chart_templ.go`.

### Task 3

- Added Ticks, Domain and AllowDataOverflow on both axes, domain-length validation with prop names, specified-tick domain extension, auto/fixed tick selection and numeric X-axis labels. Overflow clips series geometry on the numeric axis.
- Six domain cases are tested for both axis orientations; extra parser checks cover auto, unparseable strings, decimal offsets and numeric overflow. Both browser runs pass the six fixtures and match every baseline demo.
- `go test ./components/chart/`, JS syntax and `git diff --check` passed. Tick fixtures use 0/100/200/300 so Recharts-style collision culling does not obscure any expected label.
- Clarification of the plan's “drawn ticks” wording: auto domains follow computed nice ticks, while explicit Ticks controls drawn labels, matching getTicksOfScale; explicit labels alone do not replace the default zero-based domain.
- Browser numeric comparisons tolerate float64 representation noise (e.g. 220.00000000000003); displayed labels remain exact.

### Task 4

- Non-stacked areas now split both fill and outline at gaps, including valid singleton runs. Line labels skip gaps. Empty tooltip payloads hide the wrapper; nested labels use the filtered payload size.
- Existing Go gap collection is retained and tested with a missing key and a full companion series; missing values do not lower a dataMin domain.
- Both engines pass gap/tooltip assertions and all baseline comparisons. First line and area each have two M commands; tooltip at the partial row has two entries and the empty row is hidden.
- Fixture clarification: with one partial row plus a separate completely empty row, the first series necessarily has two missing values (four dots and labels for six rows); the other series each have five dots. The plan's “one dot less” cannot hold for that combined fixture.
- `go test ./components/chart/`, JS syntax and `git diff --check` passed.

### Task 5

- Ported Line.repeat, getStrokeDasharray and generateSimpleStrokeDasharray; dashed series keep their pattern during the entrance and the original string afterward.
- Chromium fixtures and all demo baselines pass. A separate motion-enabled browser run asserts repeated 4px/4px segments mid-entrance and the exact final `4 4` attribute. Visually inspected `tmp/chart-612/dashes-300ms.png`: the revealed portion is dashed.
- Local algorithm checks cover partial segments, exact pattern boundaries, odd patterns and all-zero patterns. `go test ./components/chart/`, JS syntax and `git diff --check` passed.
- Static probes now request reduced motion; the dedicated screenshot probe explicitly retains animation.

### Task 6

- Replaced DotProps.Indices with Show(index, row), evaluated once per row into DotModel.Shown; nil omits the field. Browser dots skip false rows while retaining gap filtering.
- Added Go checks for predicate inputs, call count, selected rows and nil behavior. Chromium shows exactly two dots and `chart-line-dots` matches its baseline; previous fixtures still pass.
- `go test ./components/chart/`, JS syntax and `git diff --check` passed. Aligned DotProps/DotModel fields; generated Go came from the watcher.

### Task 7

- Added Hide on Line/Area/Bar and Hidden in the model. Hidden values leave Go domains, browser stacks/normalization and grouped-bar slots. Geometry placeholders retain series indexing; legends retain entries.
- Removed the hidden-key attribute protocol, per-container MutationObserver and line data-key attributes. Tooltip rows and active dots use the model flag.
- Go tests cover line domains and stacked bar/area domains with a much larger hidden series. Both engines show two lines, three legend entries, a visible-series axis maximum of 40 and six bar segments (two per row). Normalized stacked-area fixture also has only two curves and keeps 0..1.
- All fixture assertions and all six docs / 68 gallery baseline comparisons pass in both engines. `go test ./components/chart/`, JS syntax, empty legacy-attribute search and `git diff --check` passed.

## Planner review
