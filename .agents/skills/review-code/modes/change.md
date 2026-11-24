# Change mode

The scope is a change: the difference between a **base** and a **head** revision. A PR or a branch resolves to its merge base and its tip (`git merge-base <target> <branch>`). Compare against the merge base, never against the current tip of the target branch — commits that landed there after the branch point are not part of the change.

`git diff <base> <head>` shows the change. `git show <base>:<path>` shows a file before it.

## What to read

The model covers the change and everything it reaches, not the whole package:

- every changed file in full at the head, and each changed hunk at the base;
- every symbol the change adds, removes, or changes the behavior of, and every consumer of that symbol at the head — found by grep, not assumed from the diff;
- the documents of each changed directory and its parents (core step 1), including the documents the change itself edits;
- the tests that exercise the changed paths, at both revisions.

A change that edits a rule in one place reaches every mirror of that rule. Read the mirrors the territory files name even when the diff does not touch them.

## What counts

A finding must be **introduced or exposed by the change**:

- *Introduced*: the head behaves wrongly where the base did not.
- *Exposed*: a defect that was already there, and that the change now reaches — a new caller of a broken function, a new input that meets an old missing check, a removed guard that hid it.

A defect that was already there and that the change does not reach is out of scope. If it is serious, name it in one line under **Found, not in scope** after the findings; do not rank it.

Each finding carries a **causal trace**: the path at the base, the path at the head, and the difference that produces the wrong behavior. Cite a code reference at each revision (`<path>:<line>` at the head, and at the base). A finding with no base-side reference is an introduced file — say so.

A rule the change edits in one mirror but not in its siblings is a finding of this change: the change made them disagree.

A guard the change adds to some paths of a state but not to the others is a finding of this change too. The diff shows the paths the author thought of. List the paths from the state itself (core step 1), and check the paths the diff does not touch. Rank such a path as a finding, not under **Found, not in scope**, even when it behaves as it did at the base. The change made a new rule, and the path now breaks it. In the causal trace, cite the new guard at the head and the unguarded path at both revisions.

## Evidence rules

- A check that fails on the head does not prove a regression. Run or read the same check at the base before you call it one.
- A failed base build or fixture setup proves nothing about behavior.
- A scoped green run is not a full-suite pass. Say which packages or specs ran.
- Two reviewers that agree do not prove causality. Only the trace does.
- A test the change adds that passes is not proof that the behavior is right. Read what it asserts.

## Changes to the review itself

When the change edits this skill, the review policy under `.forgejo/review/`, or the review engine under `.forgejo/actions/ai-review/`, audit it like any other code. The review that runs on the change uses the base copy of this skill and the policy, not the head copy.
