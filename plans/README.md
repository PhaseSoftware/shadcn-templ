# Plans

One file per feature, bug or refactor: `plans/<topic>.md`. Two roles. The Planner writes the plan and reviews the result. The Executor builds it. Any model can play either role. The plan file header says who is who right now.

Kick-off is one sentence: "Read plans/README.md. You are Executor for plans/<topic>.md." or "You are Planner for <topic>. Write plans/<topic>.md."

## File layout

- **Planner** and **Executor**: who plays which role right now
- **Reviewer**: optional. Defaults to the Planner. Name one only for a risky task that deserves a second pair of eyes.
- **Status**: planning, ready, in progress, review, done
- **Context**: why, in a few sentences, plus the facts the plan builds on
- **Decisions**: fixed. Reopen only with a note in the log.
- **Tasks**: numbered checklist. Each task has a "done when" line and the checks to run.
- **Executor log**: appended by the Executor after each task. What was done, what deviated, open questions.
- **Planner review**: appended by the Planner. Verdict per task, new or changed tasks.

## Rules

- The Executor edits only checkboxes in Tasks and the Executor log.
- The Planner edits everything else.
- The Planner reviews. A named Reviewer reads only the diff of the task it is named for and writes into the Planner review section.
- One commit per task. The message starts with the topic and task number: `a11y-600 3: accordion controls only the open panel`.
- A task is done only when its "done when" line is true and its checks pass.
- Questions go in the Executor log. The Executor continues with the next task that does not depend on the answer.
- Status done means the file stays as history. Never delete a plan.
- Every rule in `AGENTS.md` still applies: never run `templ generate` or rebuild `*.min.js`, the watchers do that. Components stay 1:1 with shadcn and Base UI, nothing is added that the upstream primitive does not have.
- Commits read as the project owner's own: no attribution trailers, no em-dashes, short or no body.
