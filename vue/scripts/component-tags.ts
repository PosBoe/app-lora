// The component tag resolver, replicated.
//
// Component auto-import derives a tag from a file's path
// (directoryAsNamespace + collapseSamePrefixes), so moving a .vue file can
// rename the tag every template refers to. Nothing catches that: a template
// naming a tag that no longer resolves still type-checks, and the first report
// is a blank spot in the running UI.
//
// So a structural move has to predict its tag column before it lands. This
// module reproduces unplugin-vue-components' getNameFromFilePath for this
// config, verifies every result against the plugin's own resolver, and rejects
// two files that derive the same tag. Run it as a script to print the table:
//
//   node scripts/component-tags.ts > /tmp/tags.before
//   ... move files ...
//   node scripts/component-tags.ts | diff /tmp/tags.before -
//
// An empty diff is the property a tag-neutral refactor claims. A non-empty one
// is a bug in the move, not a detail.
//
// Two more modes serve a move that is still being planned:
//
//   --usages            Add an occurrence count between tag and path. The count
//                       is the rename budget: a tag used in 40 templates needs a
//                       much better reason to change than one used twice.
//
//   --plan <plan.tsv>   Check a planned layout before a single file moves. Rows
//                       are old_path<TAB>new_path<TAB>planned_tag, relative to
//                       vue/. Prints the tags that change, and fails when a
//                       planned tag is not what the layout produces or when the
//                       future namespace collides.
//
// Plan mode does not consult the plugin. The resolver cannot answer for files
// that do not exist yet, and a tree mid-move legitimately holds names that are
// not yet pinned, which would abort the run before the plan was read at all.
// Every default-mode run verifies deriveTag against the plugin, so capture the
// table first and the derivation behind a plan check is already proven.

import { globSync, readFileSync } from 'node:fs'
import { readFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { basename, dirname, extname, join, relative, resolve } from 'node:path'

import Components from 'unplugin-vue-components/vite'
import { pascalCase } from 'unplugin-vue-components'

import { ROOT } from './root.ts'

// Literal .ts extension: node's type stripping resolves the specifier as
// written, with no .js -> .ts rewrite.
import { componentDirs, excludeComponentNames as excludeNames } from '../packages.ts'

export { componentDirs }

// TagTable is the resolved component namespace, plus everything that went wrong
// building it. Both failure lists are returned rather than thrown so a caller can
// report all of them at once instead of one per fix cycle.
export interface TagTable {
  // tag -> src-relative defining file, as the resolver actually resolves it.
  map: Map<string, string>
  // Files whose derived tag the plugin's resolver does not know, i.e. this
  // replication has drifted from the plugin.
  unresolved: string[]
  // Tags two or more files derive. The resolver keeps one and drops the rest,
  // warning mid-build but still exiting 0.
  collisions: Map<string, string[]>
  // Components whose runtime name is not their tag — see nameProblem.
  misnamed: string[]
}

// declaredName returns the name an SFC states for itself, via
// `defineOptions({ name })` in <script setup> or `name` in a defineComponent
// options object, or null when it states none.
export function declaredName(source: string): string | null {
  const setup = source.match(/defineOptions\(\s*\{[^)]*?\bname\s*:\s*'([^']+)'/s)
  if (setup) return setup[1]
  const options = source.match(/defineComponent\(\s*\{\s*name\s*:\s*'([^']+)'/s)
  return options ? options[1] : null
}

// nameProblem reports why a component answers to something other than its tag,
// or null when it answers to the right thing.
//
// A component has two names and only one of them comes from its path. The *tag*
// is derived from the directory (`monitor/Body.vue` -> <MonitorBody>), while the
// runtime name — what Vue devtools shows, what a VTU stub key matches, what
// findComponent({ name }) looks for — is whatever the SFC declares, and failing
// that the bare filename the compiler puts in `__name`. So a folder move keeps
// every template working and silently renames the component underneath: stubs
// stop matching and quietly stub nothing, and the test still passes.
//
// The rule is that the two agree. Where the filename already is the tag nothing
// needs saying; anywhere else the file states its tag with defineOptions.
export function nameProblem(tag: string, relFile: string, source: string): string | null {
  const declared = declaredName(source)
  const effective = declared ?? basename(relFile, '.vue')
  if (effective === tag) return null

  return declared
    ? `${relFile} declares the name <${declared}> but resolves as <${tag}>`
    : `${relFile} resolves as <${tag}> but answers to <${effective}>, its filename` +
        ` — add defineOptions({ name: '${tag}' })`
}

// deriveTag reimplements unplugin-vue-components' getNameFromFilePath for this
// config (directoryAsNamespace + collapseSamePrefixes, no prefix, no resolvers).
// Every result is verified against the plugin's own resolver in buildTagTable.
//
// The collapse pass is the part that surprises people: a directory prefix is
// stripped from the filename that repeats it, so incident/IncidentsTableRow.vue
// becomes <IncidentSTableRow> — the shared "Incident" goes, and the leftover "s"
// is pascal-cased into the next word. Name the directory for the exact prefix
// its files share (incidents/ for IncidentsTable*) and the tag stays readable.
export function deriveTag(absFile: string, resolvedDirs: string[]): string {
  const dir = dirname(absFile)
  let stripped = ''
  for (const d of resolvedDirs) {
    if (dir.startsWith(d)) {
      stripped = dir.slice(d.length)
      break
    }
  }

  const folders = stripped
    .slice(1)
    .split('/')
    .filter(Boolean)
    .map((f) => f.replace(/[^a-z0-9-]/gi, ''))
  let filename = basename(absFile, extname(absFile))
  if (filename.toLowerCase() === 'index') filename = ''

  if (folders.length) {
    let namespaced = [...folders, filename]
    const collapsed = []
    for (const part of namespaced) {
      let cumulative = ''
      let didCollapse = false
      const pascal = pascalCase(part)
      for (const parent of [...collapsed].reverse()) {
        cumulative = `${parent}${cumulative}`
        if (pascal.startsWith(cumulative)) {
          collapsed.push(pascal.slice(cumulative.length))
          didCollapse = true
          break
        }
      }
      if (!didCollapse) collapsed.push(pascal)
    }
    namespaced = collapsed
    filename = namespaced.filter(Boolean).join('-')
  }
  return pascalCase(filename)
}

// scanComponentFiles returns every scanned .vue file as a root-relative path.
export function scanComponentFiles(root: string): string[] {
  return globSync(
    componentDirs.map((d) => `${d}/**/*.vue`),
    { cwd: root },
  )
}

// buildTagTable resolves every scanned .vue file to its tag.
export async function buildTagTable(root: string): Promise<TagTable> {
  const src = resolve(root, 'src')
  const dts = join(tmpdir(), `lora-component-tags-${process.pid}.d.ts`)
  // unplugin keeps the raw per-bundler hook containers on the plugin object
  // alongside the flattened vite hooks. `.vite` is real at runtime but absent
  // from the public types, hence the cast.
  const plugin = Components({
    dirs: componentDirs,
    directoryAsNamespace: true,
    collapseSamePrefixes: true,
    excludeNames,
    // The plugin only runs its glob when dts is set, so point it at a throwaway
    // path rather than disabling it.
    dts,
  }) as ReturnType<typeof Components> & {
    vite: { configResolved: (this: unknown, config: unknown) => void | Promise<void> }
  }
  await plugin.vite.configResolved.call(plugin, { root, build: {}, command: 'build' })

  const resolvedDirs = componentDirs.map((d) => resolve(root, d))
  const files = scanComponentFiles(root).map((f) => resolve(root, f))

  const map = new Map<string, string>()
  const unresolved: string[] = []
  const misnamed: string[] = []
  // Every file that derived a given tag, so a collision names both sides.
  const derivedBy = new Map<string, string[]>()

  for (const file of files) {
    const tag = deriveTag(file, resolvedDirs)
    if (excludeNames.some((re) => re.test(tag))) continue

    const rel = relative(src, file)
    derivedBy.set(tag, [...(derivedBy.get(tag) ?? []), rel])

    const problem = nameProblem(tag, rel, await readFile(file, 'utf8'))
    if (problem) misnamed.push(problem)

    const info = await plugin.api.findComponent(tag)
    if (!info?.from) {
      unresolved.push(
        `${relative(src, file)} -> derived tag <${tag}>, which the resolver does not know`,
      )
      continue
    }
    // A tag resolving to a file other than the one that derived it is a
    // collision, not a derivation bug; record what the resolver actually picked
    // and let the collision report name the losers.
    map.set(tag, relative(src, resolve(root, info.from)))
  }
  await rm(dts, { force: true })

  const collisions = new Map([...derivedBy].filter(([, files]) => files.length > 1))
  return { map, unresolved, collisions, misnamed }
}

// reportProblems prints the table's failures and returns whether any were found.
// Both are fatal: a drifted derivation makes the boundary checker miss real
// violations, and a collision means one of the two components is unreachable.
export function reportProblems(table: TagTable, log: (s: string) => void): boolean {
  let failed = false

  if (table.unresolved.length) {
    failed = true
    log('component tags: derivation drifted from the resolver:\n')
    for (const u of table.unresolved) log(`  ${u}`)
    log('\nThis module mirrors unplugin-vue-components; update deriveTag to match.\n')
  }

  if (table.misnamed.length) {
    failed = true
    log(`component tags: ${table.misnamed.length} component(s) whose name is not their tag:\n`)
    for (const m of table.misnamed) log(`  ${m}`)
    log('')
    log('The tag comes from the path; the runtime name comes from the file. When they')
    log('disagree, every template keeps working while VTU stubs and findComponent({ name })')
    log('silently match nothing — so the test that stubs a child passes without stubbing it.\n')
  }

  if (table.collisions.size) {
    failed = true
    log(`component tags: ${table.collisions.size} tag(s) derived by more than one file:\n`)
    for (const [tag, files] of [...table.collisions].sort()) {
      log(`  <${tag}>`)
      for (const f of files.sort()) log(`    src/${f}`)
    }
    log('\nThe resolver keeps one and drops the rest. It prints a warning mid-build')
    log('and exits 0, so the dropped component fails only where it is rendered.')
    log('Rename a file or its directory so each component derives its own tag.\n')
  }

  return failed
}

// --- usage counting ---

// countUsages tokenizes every source file under src/ once and returns
// token -> occurrence count. A kebab-case template tag (<my-tag ...>) is counted
// under its PascalCase form, so both spellings of one component accumulate
// together.
//
// It overcounts on purpose: an unrelated identifier that happens to equal a tag
// counts too. The number is a rename budget, not an inventory — read the call
// sites for what a component is actually for.
export function countUsages(root: string): Map<string, number> {
  const counts = new Map<string, number>()
  const bump = (token: string) => counts.set(token, (counts.get(token) ?? 0) + 1)

  const files = globSync(['src/**/*.vue', 'src/**/*.ts', 'src/**/*.tsx', 'src/**/*.js'], {
    cwd: root,
  }).filter((f) => !f.endsWith('.d.ts'))

  for (const file of files) {
    const code = readFileSync(resolve(root, file), 'utf8')
    for (const [token] of code.matchAll(/[A-Za-z_$][\w$]*/g)) bump(token)
    for (const [, kebab] of code.matchAll(/<\/?([a-z][\w]*(?:-[\w]+)+)(?=[\s/>])/g)) {
      bump(pascalCase(kebab))
    }
  }
  return counts
}

// --- plan checking ---

// PlanRow is one component the plan moves. Paths are relative to vue/.
interface PlanRow {
  oldPath: string
  newPath: string
  tag: string
}

// readPlan parses the TSV. Blank lines and # comments are skipped, and columns
// past the third are ignored so a plan may carry notes of its own.
function readPlan(file: string): PlanRow[] {
  return readFileSync(file, 'utf8')
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#'))
    .map((line, i) => {
      const [oldPath, newPath, tag] = line.split('\t').map((c) => c.trim())
      if (!oldPath || !newPath || !tag) {
        console.error(`plan row ${i + 1}: need old_path<TAB>new_path<TAB>planned_tag: ${line}`)
        process.exit(2)
      }
      return { oldPath, newPath, tag }
    })
}

// checkPlan verifies a plan against the layout it would produce: current files,
// minus every old path, plus every new one. It reports the tags that change —
// that list is what tells a caller which usage sites need rewriting.
function checkPlan(root: string, planFile: string): void {
  const resolvedDirs = componentDirs.map((d) => resolve(root, d))
  const underRoot = (rel: string) =>
    resolvedDirs.some((d) => resolve(root, rel).startsWith(d + '/'))
  const tagOf = (rel: string) => deriveTag(resolve(root, rel), resolvedDirs)

  const rows = readPlan(planFile)
  const current = new Set(scanComponentFiles(root))
  const future = new Set(current)
  let failed = false

  for (const row of rows) {
    if (!current.has(row.oldPath)) {
      console.error(`plan error: old path does not exist: ${row.oldPath}`)
      failed = true
    }
    future.delete(row.oldPath)
  }
  for (const row of rows) {
    if (future.has(row.newPath)) {
      console.error(`plan error: new path already occupied: ${row.newPath}`)
      failed = true
    }
    future.add(row.newPath)
  }

  // Every planned tag must be the one its new path actually derives.
  const renames: string[] = []
  for (const row of rows) {
    if (!underRoot(row.newPath)) {
      console.error(`plan error: new path is under no component root: ${row.newPath}`)
      failed = true
      continue
    }
    const got = tagOf(row.newPath)
    if (excludeNames.some((re) => re.test(got))) {
      console.error(`plan error: ${row.newPath} resolves to the excluded name <${got}>`)
      failed = true
      continue
    }
    if (got !== row.tag) {
      console.error(`tag mismatch: ${row.newPath} resolves to <${got}>, plan says <${row.tag}>`)
      failed = true
      continue
    }
    if (current.has(row.oldPath)) {
      const oldTag = tagOf(row.oldPath)
      if (oldTag !== row.tag) renames.push(`${oldTag} -> ${row.tag}`)
    }
  }

  // The future namespace must be collision-free, across every root at once — a
  // planned directory can collide with a component the plan never touches.
  const derivedBy = new Map<string, string[]>()
  for (const rel of future) {
    if (!underRoot(rel)) continue
    const tag = tagOf(rel)
    if (excludeNames.some((re) => re.test(tag))) continue
    derivedBy.set(tag, [...(derivedBy.get(tag) ?? []), rel])
  }
  for (const [tag, files] of [...derivedBy].sort()) {
    if (files.length > 1) {
      console.error(`future collision: <${tag}> <- ${files.sort().join(', ')}`)
      failed = true
    }
  }

  console.log(`${rows.length} moves, ${renames.length} renames`)
  for (const r of renames.sort()) console.log(`rename: ${r}`)
  if (failed) process.exit(1)
  console.log('plan ok')
}

// --- CLI ---

interface Args {
  plan: string | null
  usages: boolean
}

function parseArgs(argv: string[]): Args {
  const args: Args = { plan: null, usages: false }
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--usages') {
      args.usages = true
    } else if (argv[i] === '--plan') {
      args.plan = argv[++i] ?? ''
      if (!args.plan) {
        console.error('--plan needs a plan file')
        process.exit(2)
      }
    } else {
      console.error(`unknown argument: ${argv[i]}`)
      console.error('usage: component-tags.ts [--usages] [--plan <plan.tsv>]')
      process.exit(2)
    }
  }
  return args
}

// Printed as `tag<TAB>path` so the output diffs cleanly against a table captured
// before a move. Sorted by tag, because glob order is not stable across machines.
async function main(argv: string[]): Promise<void> {
  const root = ROOT
  const args = parseArgs(argv)

  if (args.plan) {
    checkPlan(root, args.plan)
    return
  }

  const table = await buildTagTable(root)
  if (reportProblems(table, (s) => console.error(s))) {
    process.exit(2)
  }

  const usages = args.usages ? countUsages(root) : null
  for (const [tag, file] of [...table.map].sort(([a], [b]) => a.localeCompare(b))) {
    if (usages) console.log(`${tag}\t${usages.get(tag) ?? 0}\t${file}`)
    else console.log(`${tag}\t${file}`)
  }
}

// Importable without running: check-boundaries.ts builds the same table.
if (process.argv[1] && resolve(process.argv[1]) === import.meta.filename) {
  await main(process.argv.slice(2))
}
