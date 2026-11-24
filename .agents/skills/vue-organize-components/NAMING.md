# Naming and taxonomy principles

> The worked examples below come from the reference repository this project copied its
> conventions from. Read them as the shape of a case, not as paths in this tree.

Grounded in the official Vue style guide's component-naming rules (vuejs.org/style-guide), adapted to this repo's folder-as-namespace auto-import. Judge every choice by how the **full resolved tag** reads in a template — that is the only surface callers ever see; paths are just the encoding.

The existing component tree is the thing being redesigned: treat it as evidence of roles and usage, **not as a naming authority**. Do not justify a name by "that's how a neighboring component is named"; justify it by the rules below.

Every repo example here was a real path in the reference repository, checked against `node scripts/component-tags.ts` at the time of writing. A handful of cases are *constructed* — a layout the repo does not have, kept because it shows a trap — and each says so and states the derivation the resolver confirmed. Re-check any example you are about to lean on: the tree moves, and a stale example is worse than none.

## The idiomatic rules (official style guide)

1. **Multi-word names.** Component names are always multi-word (except a root `App.vue`) to avoid conflicts with current and future single-word HTML elements. Three standing violations, all resolving to a bare word today: `Btn` (`components/btn/Btn.vue`), `EChart` (`components/e-chart/EChart.vue`) and `Default` (`apps/main/default.vue`). The first two are the awkward kind — a fully-collapsing family root whose folder is itself one word, so the fix is a better *concept* name, not a longer filename.

2. **General → specific word order.** Names start with the most general words and end with descriptive modifiers: `SearchButtonClear`, not `ClearSearchButton`; `SettingsCheckboxTerms`, not `TermsCheckbox` (the guide's own examples). The guide accepts that this reads less like natural English, because it groups related components in editors and reveals relationships. In this repo the payoff is doubled: **the leading general words become the folder.** `alerting/notif-channel-form/Slack.vue` → `NotifChannelFormSlack` is the pattern in full — domain, then kind noun, then the modifier that distinguishes it from its siblings.

3. **Tightly coupled children carry the parent's name as prefix.** The guide's `TodoList` / `TodoListItem` / `TodoListItemButton`. The repo's real instance is `alerting/alert-timeline/`, which carries the same three depths of name in one flat folder:

   | File | Tag |
   |---|---|
   | `AlertTimeline.vue` | `AlertTimeline` (family root — the filename repeats the whole folder and collapses away) |
   | `UserEvent.vue` | `AlertTimelineUserEvent` |
   | `UserEventAssignees.vue` | `AlertTimelineUserEventAssignees` |

   Note the grandchild stays a *filename*, not a sub-folder. That is rule 9.

4. **Base components share one consistent prefix.** Generic presentational primitives — only HTML elements, other base components, or third-party UI, and no global state — get a common prefix. The guide offers `Base`, `App`, or `V`; `V` is off-limits here because Vuetify owns it. Pick one and apply it uniformly, and never leave a primitive single-word: `Btn` is exactly this gap, a base primitive with no base prefix.

5. **Single-instance components take `The`.** `TheNavbar`, `TheFooter` — components with exactly one active instance per app. Apply only where it genuinely communicates "there is only one"; it is not worth a mass rename on its own.

6. **Full words over abbreviations** — with this repo's documented exception: `attr` is preferred over `attribute` in identifiers (root `AGENTS.md`), and component names are identifiers. `alerting/notif-channel-form/` shows the same latitude for `notif`, which the backend uses too.

## Taxonomy discipline

7. **Folders are the general words, so they are domain concepts — never component kinds, and never broad umbrellas.** Under general→specific order the kind noun (`Button`, `Select`, `Dialog`) is a *trailing* word; a folder named after it inverts the rule and produces backwards tags (`TableAlerts`, `IconStatus`). Folder names must be nouns from the app's domain language — but the **narrowest coherent coupling**, not a broad domain concept. `alert-timeline/`, `alert-priority/` and `repeat-interval/` name tightly-coupled feature units with a naturally bounded membership. A catch-all `alert/` or `monitor/` that swallows every component sharing that prefix is the antipattern this skill exists to prevent: it has no size limit, it mixes unrelated roles, and it grows until someone reaches for a sub-folder. Generic primitives stay flat under the base prefix instead.

   **Gather the evidence mechanically, then judge it.** List the call sites that render each member, one `rg -l '<Tag\b'` per member. Read the result against one question: are the members views of **one subject**, or surfaces of **different subjects that share an owner's name**? The second is the umbrella. `iam/org/` held a settings form, a projects table and a members table — the org's settings, the org's projects and the org's people are three subjects, rendered by three different views, sharing only the possessive "org". It read tidy, and its name was a real domain noun, and it was still an umbrella.

   Scattered call sites are strong evidence, not proof, so do not stop at counting them. `iam/user-invites/` is rendered from two unrelated views — `OrgInvites` renders its table, `UserInvite` renders its form — and is a true unit, because a list of invites and the form that creates one are two views of one subject. A folder can equally have a single entry view and still be an umbrella, when that view is a settings page that stacks unrelated sections. The subject count is the reading; the call sites are how you see it.

8. **Flat is the default; a folder must earn itself.** The bar is *tight coupling*, not a shared prefix: a parent and its dedicated children, or a feature unit whose members change together. Folder only at **≥ 3 tightly-coupled members**, never a one-file folder, and never a folder broad enough to keep accumulating members. Two exceptions take **≥ 2**: merging into a folder that has itself passed the tests below, and a **parent with its dedicated child** — a table and its row, a form and its item — where the child is rendered by that parent alone and by nothing else. That pair is closed by construction, so the growth test has no question left to ask. When the choice is close, stay flat.

   The one-file rule had live violations to learn from: `alerting/alert-policy/Form.vue`, `alerting/facet-filter/Chips.vue` and `alerting/incident-filters/Assignee.vue` each bought a namespace segment for a single file. The tag was no worse flat, and the folder was an invitation to add a second unrelated member. All three are gone: the first two are the flat `AlertPolicyForm` and `FacetChips`, and the third was removed with its filters.

   Name an earned folder for the whole shared prefix, not the bare entity: `transpipe/pipe-ops-table/`, holding `PipeOpsTable.vue` and `Row.vue`, gives `PipeOpsTable` and `PipeOpsTableRow`. Under a `pipe-op/` folder the same root file derives **`PipeOpSTable`** — the resolver strips the `PipeOp` prefix and stranded the `s` (confirmed by `--plan`; rule 12).

   **A folder already in the tree is a candidate, not a given.** Everything above governs the folders the scope *has* exactly as it governs the ones a candidate proposes, and a folder that fails is dissolved. This is the rule the skill is most likely to break, because the status quo writes no plan row and so is never asked to justify itself — and because the ≥ 2 merge exception reads as an invitation to feed an umbrella one more member. Dissolving is usually cheap: a folder whose members already resolve to `OrgForm`, `OrgProjectsTable` and `OrgUsersTable` flattens to files of exactly those names, with **no tag change and no template edit at all**. Price the dissolve before assuming it is expensive.

9. **One namespace level; avoid sub-folders.** The style guide recommends *against* grouping components into directories at all, and prefers a flat `components/` with the general words spelled into each filename. It grants one exception — "very large apps (e.g. 100+ components)" — and gives three costs to weigh: slower navigation through nested directories, name conflicts between same-named files in different folders, and refactors that find-and-replace cannot finish. This repo holds well over 100 components across the scan roots `packages.ts` lists, so it sits inside the exception, and `directoryAsNamespace` answers the second cost outright by making the folder part of the tag. That is the whole licence for foldering here, and it does not extend one level deeper.

   So: prefer a single folder over the filename, and do not nest. `apps/main/nav/panel/` is a two-segment namespace — the panels `NavPanelDashboards`, `NavPanelExplore`, `NavPanelOverview`, `NavPanelProjects`, `NavPanelSettings` and `NavPanelSetup`, plus the `NavPanelRows` they share. It works because the panels are one closed set, and it is still the ceiling rather than a target. `tracing/span-card/exception/` and `tracing/span-card/breadcrumbs/` are the other two-segment namespaces, and they hold to the same ceiling: the sibling `nav/user-notification-list/` keeps its parent-child pair one level deep instead (`UserNotificationList.vue` + `Item.vue` → `NavUserNotificationList` + `NavUserNotificationListItem`). If a folder grows enough to *tempt* a sub-folder, that is the tell that it was an umbrella: split it into specific siblings, or flatten it — never deepen it.

10. **Keep good names; spend renames deliberately.** A rename must be justified by a concrete rule violation or a consistency gain, weighted by usage count — the rename budget from the inventory. "Slightly nicer" does not justify touching 40 templates. The conservative baseline candidate exists to keep this honest.

11. **Suffix families get reordered or left alone.** A suffix family such as `TextValue` / `NumberValue` / `BytesValue` violates general→specific order, and namespacing is prefix-based, so it cannot be foldered as written: `value/TextValue.vue` derives `ValueTextValue`. The repo already took the idiomatic fix — `components/value/` holds `Text.vue`, `Number.vue`, `Bytes.vue`, `Any.vue`, `Bar.vue`, `Percent.vue` and yields `ValueText`, `ValueNumber`, `ValueBytes`. Reordering is the fix; leaving the family flat is the pragmatic alternative when the rename cost outweighs the gain. Choose consciously and say which in the plan rationale — never bend the layout to preserve a backwards name.

    Not every trailing word is a suffix family. `components/date/Value.vue` → `DateValue` is already general→specific: `Date` is the domain and `Value` is the modifier. Check which word is the general one before reaching for a reorder.

12. **Design for the resolver, not against it.** The layout must encode the planned tags cleanly under the collapse rules (REFERENCE.md). If achieving a tag requires a filename that repeats part of the folder, the tag is misdesigned — pick a different namespace or role name. Never settle this by eye; `--plan` answers it in a second.

## Number, mode words, and forms

Rules 1–12 fix the word *order*. These settle the question that comes up on almost every form: is it `TeamUsersAddForm`, `TeamUserAddForm`, `TeamUserForm`, `TeamUserFormAdd`, or a `New`/`Edit` pair?

Say where each rule's authority comes from, because it differs. **Rule 14 is the style guide's**, applied literally. **Rules 13 and 15 are house decisions** on questions the guide does not address — they are settled, but do not cite them as idiom.

A warning that applies to all three: the repo is **not** the authority here, and on rule 14 the repo is largely wrong. Count nothing; check the guide, then check the tree to see how much drift you are standing in.

13. **Grammatical number follows what the component handles — many or one.** A view over a collection takes the plural; a view over a single record takes the singular. *House decision: the style guide is silent on number, so this rests on ordinary English agreement and nothing more.* The tree already reads this way — most `*Table` tags are plural (`AlertsTable`, `MonitorsTable`, `TeamUsersTable`, `OrgProjectsTable`, `UserInvitesTable`), while every single-record surface is singular (`AlertCard`, `IncidentCard`, `MetricMonitorForm`, `TeamForm`, `AnnotationForm`) — but that is corroboration, not the reason.

    Two things are not violations. A **mass noun** has no plural to take: `TimeseriesTable`, `ForecastSeriesTable`, `ChartLegendTable`. And a **folder segment** takes the number of the folder's own subject, which is why `pipe-ops-table/` is plural (it holds a table of many) and `alert-priority/` is not (one attribute).

    Genuine drift, which reads as a rule and is not one: `SavedViewTable` and `WidgetTable` both list many rows under a singular name. Do not copy them. `AlertEventTable` was a third until its module came into scope and it became `AlertEventsTable`.

    The practical consequence: a form that acts on **one** record is singular even when its sibling table is plural. `TeamUsersTable` and `TeamUserAdd` are both correct, and they still sort together.

14. **A mode word is a descriptive modifier, so it goes last — after the kind noun.** `<Subject><Kind><Mode>`. This is rule 2 applied without an exception, and the style guide's own examples are all of exactly this shape, verbs included:

    ```
    SearchButtonClear.vue      SearchButtonRun.vue
    SearchInputQuery.vue       SearchInputExcludeGlob.vue
    SettingsCheckboxTerms.vue  SettingsCheckboxLaunchOnStartup.vue
    ```

    `Clear` and `Run` are verbs, and the guide still puts them after `Button`. So the mode word is not special: `Add`, `New`, `Edit` and `Create` are modifiers like any other, and a name that fronts one is a rule 2 violation, not a separate convention.

    **The tree disagrees with this, and the tree is wrong.** The folder families already conform, because the folder puts the kind noun in front for free — `notif-channel-form/Slack.vue` → `NotifChannelFormSlack`, `pipe-op-form/RenameAttr.vue` → `PipeOpFormRenameAttr`, `widget-form/Chart.vue` → `WidgetFormChart`. Standalone components are where the drift sits.

    **Re-derive the list; do not trust the one below.** It is a snapshot of 2026-09-13, and the modules are being fixed one at a time. Take `node scripts/component-tags.ts` and read every tag that holds a mode word ahead of its kind noun. At the snapshot these stand:

    | Module | Tag | Reads |
    |---|---|---|
    | `transpipe` | `PipeOpEditDialog` | `PipeOpDialogEdit` |
    | `transpipe` | `PipeOpNewMenu` | `PipeOpMenuNew` |
    | `auth` | `MfaConfirmCard` | `MfaCardConfirm` |
    | `tracing` | `SavedViewNewDialog` | `SavedViewDialogNew` |
    | `tracing` | `ExtendTraceRetentionMenu` | `TraceRetentionMenuExtend` |
    | `license` | `LicenseExtendForm` | `LicenseFormExtend` |
    | `components` | `LinkedIssueCreateDialog` | `LinkedIssueDialogCreate` |
    | `components` | `FacetAddDialog` | `FacetDialogAdd` |

    Others were fixed as their modules came into scope, among them `IncidentDialogCreate`, `MonitorMenuNew`, `MonitorMenuNewFromQuery`, `MonitorYamlDialogEdit`, `ProjectTokensDialogCreate`, `FilterBuilderMenuAdd`, `WidgetMenuNew` and `DashboardMenuNew`. Some left the list another way: a component was removed, or its rename took the mode word out. That is the pace to expect, and it is why a stale list here is worse than none.

    Some cases want a naming decision rather than a reorder, so they are not on the list: `dashboard/add-query/`, where the *folder* fronts the verb and two of its tags are named in `packages.ts`; and `ExportData` and `ArchiveBtn`, which front a verb with no subject at all, so the fix is a subject.

    Fix drift when its module is in scope; do not cite it as precedent, and do not treat its number as a vote.

    A mode word earns its place only when the subject alone would be ambiguous. **Most components need none: of the tags that end in `Form`, only `LicenseExtendForm` carries a mode word** — that is the entry above, which is drift and not a reading of this rule. Reach for one when a sibling acts on the same subject in a different mode — which is the case the guide's `SearchButtonClear` / `SearchButtonRun` pair is about — or when the action is the whole point and no entity is being created or edited.

15. **One form serves create and edit; split it only when the flows genuinely differ.** The default name is the bare `<Subject>Form`. *House decision: the guide does not address this.* `iam/TeamForm.vue` is the pattern — one component, one set of fields, and a button reading "Create team" or "Save" depending on `team.id` — and the tree holds no `*FormNew` / `*FormEdit` pair anywhere.

    When a split is genuinely warranted, it is a component decision that the name then reports, and rule 14 places the mode word: `TeamFormNew` and `TeamFormEdit`, never `TeamNewForm`.

    **A control that attaches an existing record to a collection is not an entity form.** It creates nothing and edits nothing; it writes a membership. That is the case where a mode word is load-bearing, because the bare subject name would promise a form the component does not provide.

    Resolving the question in full:

    | Candidate | Verdict |
    |---|---|
    | `TeamUsersAddForm` | Wrong number (13) — it adds one user, not many — and fronts the mode word (14). |
    | `TeamUserAddForm` | Right number, but fronts the mode word ahead of the kind noun (2, 14). |
    | `TeamAddUserForm` | Fronts the verb ahead of the subject too. Worst of the set. |
    | `TeamUserForm` | Promises create/edit of a team user (15); the component only attaches an existing one. |
    | `TeamUserFormNew` + `TeamUserFormEdit` | Splits what the house keeps whole, and invents an edit mode that does not exist (15). |
    | **`TeamUserFormAdd`** | **Correct.** Singular subject (13), kind noun before the modifier (2, 14) — the shape of `SearchButtonClear`, and of `NotifChannelFormSlack` already in the tree. |

    The tree took `TeamUserFormAdd`, and later renamed it `TeamUserAdd` (`ae322228a2`). That change made the control a picker and a verb in the card header, not a form, so no kind noun was left to name. The table still answers the question for a control that is a form.

## Worked examples: good vs bad folders

Real cases from this repo. The contrast is always *tight bounded coupling* vs *broad prefix umbrella*.

**Good — a tight, bounded coupling earns a folder:**

| Folder | Tags | Why it works |
|---|---|---|
| `alerting/alert-priority/` — `Icon.vue` `Menu.vue` `MenuBase.vue` `StatusLabel.vue` | `AlertPriorityIcon` `AlertPriorityMenu` `AlertPriorityMenuBase` `AlertPriorityStatusLabel` | Ways to view or set *one* attribute; `MenuBase` is what the menu builds on, and `StatusLabel` is the tooltip body of the icon and the menu. Closed set — the attribute bounds it. |
| `alerting/alert-timeline/` — `AlertTimeline.vue` `AutoEvent.vue` `UserEvent.vue` `UserEventAssignees.vue` | `AlertTimeline` … `AlertTimelineUserEventAssignees` | A feature unit: the timeline, its two event kinds, and one event's detail. Bounded by the number of event kinds. |
| `alerting/repeat-interval/` — `Fixed.vue` `Linear.vue` `Exponential.vue` `Panel.vue` | `RepeatIntervalFixed` … `RepeatIntervalPanel` | A closed set of interval strategies plus their host panel. |
| `alerting/notif-channel-form/` — `Base.vue`, one file per channel (`Slack.vue`, `Email.vue`, `Pagerduty.vue`, …), `MenuNew.vue` | `NotifChannelFormBase` `NotifChannelFormSlack` … `NotifChannelFormMenuNew` | Parent (`Base`) plus one sibling per channel — the canonical parent/children family. Large, yet bounded: the channel list sets the membership, not the domain word. `MenuNew` is the one member that is not a channel, and it is the menu that picks among them. |
| `transpipe/pipe-op-form/` — `RenameAttr.vue` `Drop.vue` `Sample.vue` … | `PipeOpFormRenameAttr` `PipeOpFormDrop` … | One form per `PipeOpType` member, all dispatched by one `<component :is>`. Bounded by the enum: a new form means a new op type. |
| `transpipe/pipe-ops-table/` — `PipeOpsTable.vue` `Row.vue` | `PipeOpsTable` `PipeOpsTableRow` | The rule-8 pair exception: the row is rendered by that table alone. Folder named for the full prefix, so the tags survive the collapse. |

**Bad — a broad prefix umbrella is not a family.** The first three rows were in the tree until 2026-08-15, when `7dc16c589d`, `aae91c4548` and `042df5c9d5` dissolved them. They stay here because that is the point: an umbrella is not a mistake someone is about to make, it is the shape a folder drifts into while nobody re-judges it.

| Folder | Why it fails | Do instead |
|---|---|---|
| `alerting/monitor/` (dissolved) — `Actions` `DescriptionSection` `FormPanes` `NameSection` `NewMenu` `NotificationsPanel` `Picker` `StatusAvatar` `Thresholds` `TrendPanel` `TypeIcon` `YamlEditDialog` `YamlForm` | Growth test. A picker, a type icon, a YAML form and a trend panel are independent roles sharing one domain word. Nothing bounded the membership, and the pressure for `monitor/form/` and `monitor/yaml/` was already there. | Flatten most of it; carve out only the real couplings. Done: most members are flat in `alerting/` (`MonitorActions`, `MonitorPicker`, `MonitorTypeIcon`, `MonitorYamlForm`), and the form parts are `alerting/monitor-form/` (`MonitorFormName`, `MonitorFormPanes`, …). |
| `alerting/alert/` (dissolved) — `ActivityStateMark` `Icons` `Metadata` `Selection` `Sparkline` `StatusMark` `TrendChart` | Same failure. A chart, a selection toolbar and two marks share only the word "alert" — and "alert" is this module's central noun, so every future alert component landed here by default. | Flat, beside the earned `alert-priority/`, `alert-timeline/` and `alert-card/`. Done: `AlertSelection`, `AlertSparkline`, `AlertTrendChart`, `AlertMetadata` and `AlertIcons` are flat in `alerting/`. |
| `iam/org/` (dissolved) — `Form` `ProjectsTable` `UsersTable` | Subject test: the org's settings, its projects and its people are three subjects, rendered by three views (`OrgShow`; `OrgList`+`OrgShow`; `OrgMembers`), sharing a possessive. An org is the product's top container, so the folder can only grow. | Dissolve. The tags are already `OrgForm`, `OrgProjectsTable`, `OrgUsersTable`, so flattening is **tag-neutral and edits no template**. Done: `iam/OrgForm.vue`, `iam/OrgProjectsTable.vue`, `iam/OrgUsersTable.vue`. |
| `transpipe/pipe-op/` — would swallow the flat `PipeOpEditDialog` `PipeOpNewMenu` `PipeOpScopePicker` | The same failure at three files: a dialog, a menu and a picker are independent roles sharing a domain word. It would also spring the stranded-`s` trap on the table root (rule 8). | Leave them flat, beside the earned `pipe-op-form/` and `pipe-ops-table/`. |
| `table/` or `icon/` grouping every `*Table` / `*Icon` | Component-kind folder (rule 7): inverts general→specific, giving backwards tags `TableAlerts`, `IconStatus`. | Domain-noun folders only; primitives flat under a base prefix. |

**Bad — names that violate the rules, and their fixes:**

| Bad name | Rule | Fix |
|---|---|---|
| `components/btn/Btn.vue` → `Btn` | 1 (single-word), 4 (unprefixed primitive) | Give the base primitive the codebase's chosen base prefix: `AppBtn` / `BaseBtn`. Renaming the file alone cannot fix it — the folder is one word too. |
| `apps/main/default.vue` → `Default` | 1 (single-word) | Say what it is: `DefaultLayout`. |
| `PriorityMenu` beside `alert-priority/Menu.vue` → `AlertPriorityMenu` | 2 (general word missing) | Two priority menus, one of which does not say whose priority. Fold it into `alert-priority/` if it is the same subject, or qualify it. The tree qualified it: `alerting/IncidentPriorityMenu.vue` → `IncidentPriorityMenu`. |
| *Constructed:* `empty-state/State.vue` → `EmptyStateState` | 12 (filename *partially* repeats the folder, so the collapse leaves both) | Use a non-overlapping remainder, as the real `NoData.vue` does, or the fully-collapsing root `EmptyState.vue`. Derivation confirmed by `--plan`. |
| *Constructed:* `value/TextValue.vue` → `ValueTextValue` | 11 (suffix family) | Reorder to `value/Text.vue` → `ValueText`, which is what the repo actually does. |

The through-line: **when a grouping is close, stay flat.** A slightly longer flat list is cheaper to live with than a bucket you have to split later.

## Scoring rubric (design it twice)

Score each candidate taxonomy 1–5 per axis; justify the winner in one paragraph:

- **Idiomatic fit** — conformance to rules 1–6; count remaining violations.
- **Call-site readability** — do the tags read well *given* general→specific order? Judge the tags, not the tree.
- **Folder coherence** — two tests, both must pass, and both run over **every folder in the candidate's tree, inherited ones included**. A candidate that keeps an umbrella it did not create scores as though it had just created one. *Deletion test*: if this folder vanished, would its members scatter to unrelated places (bad grouping), or clearly belong together somewhere (real concept)? *Growth test*: is the membership naturally bounded, or is this a domain umbrella that any future component of that domain would land in? Answer it from the render sites and the subject count of rule 7, never from the folder's name. An unbounded folder scores low even when its current members look tidy — it is a sub-folder waiting to happen.
- **Rename cost** — total usage sites touched; every point spent needs a rationale.
- **Resolver fit** — no collisions, no collapse traps, no awkward encodings. `--plan` is the arbiter.
