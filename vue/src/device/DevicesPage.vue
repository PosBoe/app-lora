<script setup lang="ts">
import type { LocationQuery } from 'vue-router'

import DeviceDialog from './DeviceDialog.vue'
import { modelLabel, useDeviceModels } from './use-device-models'
import {
  archiveDevices,
  displayDevEUI,
  METER_TYPES,
  useDeviceList,
  type Device,
  type DeviceFilter,
  type MeterType,
} from './use-devices'

const PER_PAGE = 50

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const errorMessage = useErrorMessage()
const { devices, total, loading, error, load } = useDeviceList()
const { models } = useDeviceModels()

// The filter lives in the URL, so the back button and a shared link keep it.
function filterOf(query: LocationQuery): DeviceFilter {
  const text = (key: string) => (typeof query[key] === 'string' ? (query[key] as string) : '')
  const modelId = Number(text('model'))
  const page = Number(text('page'))
  const meterType = text('type')
  return {
    q: text('q'),
    modelId: modelId > 0 ? modelId : null,
    meterType: (METER_TYPES as string[]).includes(meterType) ? (meterType as MeterType) : null,
    archived: text('archived') === '1',
    page: page > 1 ? page : 1,
    perPage: PER_PAGE,
  }
}

const filter = computed(() => filterOf(route.query))

// setFilter writes a change to the URL. A change other than the page goes
// back to the first page.
function setFilter(change: Partial<DeviceFilter>) {
  const next = { ...filter.value, ...change }
  if (!('page' in change)) {
    next.page = 1
  }
  void router.replace({
    query: {
      q: next.q || undefined,
      model: next.modelId ?? undefined,
      type: next.meterType ?? undefined,
      archived: next.archived ? '1' : undefined,
      page: next.page > 1 ? next.page : undefined,
    },
  })
}

const search = shallowRef(filter.value.q)
const applySearch = useDebounceFn((q: string) => setFilter({ q }), 300)
watch(search, (q) => applySearch(q.trim()))

watch(filter, (f) => load(f), { immediate: true })

const selected = ref<number[]>([])
watch(filter, () => {
  selected.value = []
})

const headers = computed(() => [
  { title: t('devices.name'), key: 'name', sortable: false },
  { title: t('devices.devEui'), key: 'dev_eui', sortable: false },
  { title: t('devices.model'), key: 'model', sortable: false },
  { title: t('devices.meterType'), key: 'meter_type', sortable: false },
  { title: t('devices.address'), key: 'address', sortable: false },
  { title: t('devices.lastUplink'), key: 'last_uplink', sortable: false },
])

const modelItems = computed(() => models.value.map((m) => ({ title: modelLabel(m), value: m.id })))
const meterItems = computed(() =>
  METER_TYPES.map((v) => ({ title: t(`meterTypes.${v}`), value: v })),
)

const filtered = computed(
  () =>
    !!(filter.value.q || filter.value.modelId || filter.value.meterType || filter.value.archived),
)

// rowProps makes a whole row open its device page. Vuetify 4 has no
// click:row event. A click on the select checkbox only selects.
function rowProps({ item }: { item: Device }) {
  return {
    class: 'cursor-pointer',
    onClick: (e: MouseEvent) => {
      if ((e.target as HTMLElement | null)?.closest('.v-selection-control')) {
        return
      }
      void router.push({ name: 'Device', params: { id: item.id } })
    },
  }
}

//------------------------------------------------------------------------------

const addOpen = shallowRef(false)

function onSaved(device: { id: number }) {
  void router.push({ name: 'Device', params: { id: device.id } })
}

const archiveOpen = shallowRef(false)
const archiving = shallowRef(false)
const archiveError = shallowRef('')

function askArchive() {
  archiveError.value = ''
  archiveOpen.value = true
}

async function confirmArchive() {
  archiving.value = true
  archiveError.value = ''
  try {
    await archiveDevices(selected.value)
    archiveOpen.value = false
    selected.value = []
    await load(filter.value)
  } catch (err) {
    archiveError.value = errorMessage(err)
  } finally {
    archiving.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-7xl p-6">
    <div class="app-card">
      <div class="app-card-header flex items-center justify-between gap-4">
        <span>
          {{ $t('devices.title') }}
          <span class="ml-1 text-muted" data-test="devices-total">({{ total }})</span>
        </span>
        <v-btn
          variant="flat"
          color="primary"
          size="small"
          prepend-icon="mdi-plus"
          data-test="device-add"
          @click="addOpen = true"
        >
          {{ $t('devices.add') }}
        </v-btn>
      </div>

      <div class="flex flex-wrap items-center gap-3 border-b border-subtle p-4">
        <v-text-field
          v-model="search"
          :placeholder="$t('devices.search')"
          prepend-inner-icon="mdi-magnify"
          clearable
          density="compact"
          variant="outlined"
          hide-details
          class="min-w-64 flex-1"
          data-test="device-search"
          @click:clear="search = ''"
        />
        <v-select
          :model-value="filter.modelId"
          :items="modelItems"
          :placeholder="$t('devices.allModels')"
          clearable
          density="compact"
          variant="outlined"
          hide-details
          class="w-56 flex-none"
          data-test="filter-model"
          @update:model-value="(v: number | null) => setFilter({ modelId: v })"
        />
        <v-select
          :model-value="filter.meterType"
          :items="meterItems"
          :placeholder="$t('devices.allTypes')"
          clearable
          density="compact"
          variant="outlined"
          hide-details
          class="w-40 flex-none"
          data-test="filter-type"
          @update:model-value="(v: MeterType | null) => setFilter({ meterType: v })"
        />
        <v-checkbox
          :model-value="filter.archived"
          :label="$t('devices.archivedOnly')"
          density="compact"
          hide-details
          class="flex-none"
          data-test="filter-archived"
          @update:model-value="(v: boolean | null) => setFilter({ archived: !!v })"
        />
        <v-btn
          v-if="selected.length && !filter.archived"
          variant="tonal"
          size="small"
          prepend-icon="mdi-archive-outline"
          data-test="archive-selected"
          @click="askArchive"
        >
          {{ $t('devices.archiveSelected', { n: selected.length }) }}
        </v-btn>
      </div>

      <div v-if="error" class="flex items-center gap-3 p-4">
        <span class="text-error" data-test="devices-error">{{ errorMessage(error) }}</span>
        <v-btn size="small" variant="text" @click="load(filter)">{{ $t('common.retry') }}</v-btn>
      </div>

      <div
        v-else-if="!loading && total === 0 && !filtered"
        class="flex flex-col items-center gap-3 p-10 text-center"
        data-test="devices-empty"
      >
        <p class="text-muted">{{ $t('devices.empty') }}</p>
        <v-btn variant="flat" color="primary" prepend-icon="mdi-plus" @click="addOpen = true">
          {{ $t('devices.add') }}
        </v-btn>
      </div>

      <v-data-table-server
        v-else
        v-model="selected"
        :headers="headers"
        :items="devices"
        :items-length="total"
        :loading="loading"
        :page="filter.page"
        :items-per-page="PER_PAGE"
        :items-per-page-options="[PER_PAGE]"
        :no-data-text="$t('devices.noMatch')"
        :show-select="!filter.archived"
        item-value="id"
        density="compact"
        :row-props="rowProps"
        hover
        data-test="devices-table"
        @update:page="(p: number) => setFilter({ page: p })"
      >
        <template #[`item.name`]="{ item }">
          <span class="text-strong" data-test="device-row-name">{{ item.name }}</span>
        </template>
        <template #[`item.dev_eui`]="{ item }">
          <span class="font-mono text-body">{{ displayDevEUI(item.dev_eui) }}</span>
        </template>
        <template #[`item.model`]="{ item }">
          <span class="text-body">{{ modelLabel(item.device_model) }}</span>
        </template>
        <template #[`item.meter_type`]="{ item }">
          <span v-if="item.meter_type" class="text-body">
            {{ $t(`meterTypes.${item.meter_type}`) }}
          </span>
        </template>
        <template #[`item.address`]="{ item }">
          <span class="text-muted">{{ item.address }}</span>
        </template>
        <template #[`item.last_uplink`]>
          <span class="text-faint">—</span>
        </template>
      </v-data-table-server>
    </div>

    <DeviceDialog v-model="addOpen" @saved="onSaved" />

    <ConfirmDialog
      v-model="archiveOpen"
      :title="$t('devices.archive')"
      :text="$t('devices.archiveConfirm', { n: selected.length })"
      :confirm-text="$t('devices.archive')"
      :loading="archiving"
      :error="archiveError"
      @confirm="confirmArchive"
    />
  </div>
</template>
