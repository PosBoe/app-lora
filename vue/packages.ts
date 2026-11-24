// The manifest of what Vite auto-imports. vite.config.ts and
// scripts/component-tags.ts both read it, so the tag table a script predicts is
// the one the build registers.
//
// Literal .ts extensions: the scripts run under node's type stripping.

// The directories whose exports are globals: the composables.
export const autoImportDirs: string[] = ['src/use']

// The directories whose .vue files are global components. A directory is a
// namespace: src/components/page/NotFound.vue is <PageNotFound>. A feature
// module is not listed: its components are imported by relative path.
export const componentDirs: string[] = ['src/components']

// Tags that match one of these are not registered.
export const excludeComponentNames: RegExp[] = []
