<script setup lang="ts">
import { modelLabel } from './use-device-models'
import { displayDevEUI } from './use-devices'
import {
  commitImport,
  previewImport,
  TEMPLATE_URL,
  type ImportRow,
  type ImportStatus,
  type ImportTotals,
} from './use-device-import'

const { t, te } = useI18n()
const errorMessage = useErrorMessage()

const file = shallowRef<File | null>(null)
const rows = shallowRef<ImportRow[]>([])
const totals = shallowRef<ImportTotals>()
const result = shallowRef<ImportTotals>()
const busy = shallowRef(false)
const error = shallowRef('')
const shown = shallowRef<ImportStatus | 'all'>('all')

async function choose(picked: File | File[] | null) {
  const next = Array.isArray(picked) ? (picked[0] ?? null) : picked
  file.value = next
  rows.value = []
  totals.value = undefined
  result.value = undefined
  error.value = ''
  shown.value = 'all'
  if (!next) {
    return
  }
  busy.value = true
  try {
    const preview = await previewImport(next)
    rows.value = preview.rows
    totals.value = preview.totals
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busy.value = false
  }
}

async function runImport() {
  if (!file.value) {
    return
  }
  busy.value = true
  error.value = ''
  try {
    result.value = await commitImport(file.value)
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busy.value = false
  }
}

function restart() {
  void choose(null)
}

const visible = computed(() =>
  shown.value === 'all' ? rows.value : rows.value.filter((r) => r.status === shown.value),
)

const headers = computed(() => [
  { title: t('import.line'), key: 'line', width: 72 },
  { title: t('import.status'), key: 'status', sortable: false },
  { title: t('devices.devEui'), key: 'dev_eui', sortable: false },
  { title: t('devices.name'), key: 'name', sortable: false },
  { title: t('devices.model'), key: 'model', sortable: false },
  { title: t('devices.groups'), key: 'groups', sortable: false },
  { title: t('import.reason'), key: 'reason', sortable: false },
])

const statusColor: Record<ImportStatus, string> = {
  new: 'success',
  exists: 'info',
  error: 'error',
}

function reason(row: ImportRow): string {
  if (row.status === 'exists') {
    return t('import.existsReason')
  }
  if (row.status === 'error' && row.code) {
    const key = `errors.${row.code}`
    return te(key) ? t(key) : row.code
  }
  return ''
}
</script>

<template>
  <div class="mx-auto flex max-w-7xl flex-col gap-6 p-6">
    <router-link :to="{ name: 'Home' }" class="app-link text-sm">
      ← {{ $t('devices.title') }}
    </router-link>

    <div class="app-card">
      <div class="app-card-header">{{ $t('import.title') }}</div>
      <div class="flex flex-col gap-4 p-4">
        <p class="text-muted">
          {{ $t('import.intro') }}
          <a :href="TEMPLATE_URL" download class="app-link" data-test="import-template">
            {{ $t('import.template') }}
          </a>
        </p>
        <v-file-input
          :model-value="file"
          :label="$t('import.file')"
          accept=".csv,text/csv"
          prepend-icon=""
          prepend-inner-icon="mdi-file-delimited-outline"
          density="compact"
          variant="outlined"
          hide-details
          :disabled="busy"
          data-test="import-file"
          @update:model-value="choose"
        />
        <v-progress-linear v-if="busy" indeterminate color="primary" />
        <p v-if="error" class="text-error" data-test="import-error">{{ error }}</p>
      </div>
    </div>

    <div v-if="result" class="app-card" data-test="import-result">
      <div class="app-card-header">{{ $t('import.doneTitle') }}</div>
      <div class="flex flex-col gap-2 p-4 text-body">
        <p data-test="import-result-text">
          {{
            $t('import.done', {
              n: result.new,
              models: result.new_models,
              groups: result.new_groups,
            })
          }}
        </p>
        <p v-if="result.exists || result.error" class="text-muted">
          {{ $t('import.skipped', { exists: result.exists, error: result.error }) }}
        </p>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="restart">{{ $t('import.another') }}</v-btn>
        <v-btn :to="{ name: 'Home' }" variant="flat" color="primary">
          {{ $t('import.toList') }}
        </v-btn>
      </div>
    </div>

    <div v-else-if="totals" class="app-card" data-test="import-preview">
      <div class="app-card-header flex flex-wrap items-center justify-between gap-4">
        <span>{{ $t('import.previewTitle') }}</span>
        <v-btn
          variant="flat"
          color="primary"
          size="small"
          :disabled="!totals.new"
          :loading="busy"
          data-test="import-submit"
          @click="runImport"
        >
          {{ $t('import.submit', { n: totals.new }) }}
        </v-btn>
      </div>

      <div class="flex flex-wrap items-center gap-3 border-b border-subtle p-4">
        <span class="text-body" data-test="import-totals">
          {{
            $t('import.totals', {
              new: totals.new,
              exists: totals.exists,
              error: totals.error,
              models: totals.new_models,
              groups: totals.new_groups,
            })
          }}
        </span>
        <v-btn-toggle
          v-model="shown"
          mandatory
          density="compact"
          variant="text"
          divided
          class="ml-auto border border-subtle"
        >
          <v-btn value="all" size="small">{{ $t('import.all') }}</v-btn>
          <v-btn value="new" size="small">{{ $t('import.statuses.new') }}</v-btn>
          <v-btn value="exists" size="small">{{ $t('import.statuses.exists') }}</v-btn>
          <v-btn value="error" size="small" data-test="import-show-errors">
            {{ $t('import.statuses.error') }}
          </v-btn>
        </v-btn-toggle>
      </div>

      <v-data-table
        :headers="headers"
        :items="visible"
        item-value="line"
        density="compact"
        :items-per-page="100"
        data-test="import-rows"
      >
        <template #[`item.status`]="{ item }">
          <v-chip
            size="x-small"
            variant="tonal"
            :color="statusColor[item.status]"
            data-test="import-row-status"
          >
            {{ $t(`import.statuses.${item.status}`) }}
          </v-chip>
        </template>
        <template #[`item.dev_eui`]="{ item }">
          <span class="font-mono">
            {{ item.status === 'error' ? item.dev_eui : displayDevEUI(item.dev_eui) }}
          </span>
        </template>
        <template #[`item.model`]="{ item }">
          <span v-if="item.vendor && item.model">
            {{ modelLabel({ vendor: item.vendor, model: item.model }) }}
            <v-chip v-if="item.new_model" size="x-small" variant="tonal" color="primary">
              {{ $t('import.newBadge') }}
            </v-chip>
          </span>
        </template>
        <template #[`item.groups`]="{ item }">
          <div class="flex flex-wrap gap-1 py-1">
            <v-chip
              v-for="g in item.groups"
              :key="g"
              size="x-small"
              variant="tonal"
              :color="item.new_groups.includes(g) ? 'primary' : undefined"
            >
              {{ g }}
            </v-chip>
          </div>
        </template>
        <template #[`item.reason`]="{ item }">
          <span :class="item.status === 'error' ? 'text-error' : 'text-muted'">
            {{ reason(item) }}
          </span>
        </template>
      </v-data-table>
    </div>
  </div>
</template>
