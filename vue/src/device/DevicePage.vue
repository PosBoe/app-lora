<script setup lang="ts">
import { formatDateTime } from '@/lib/i18n'

import DailyPulseChart from './DailyPulseChart.vue'
import DeviceDialog from './DeviceDialog.vue'
import PulseLineChart from './PulseLineChart.vue'
import {
  fetchAlarms,
  fetchPulses,
  formatDuration,
  PULSE_RANGES,
  type Alarm,
  type PulseData,
  type PulseRange,
} from './use-readings'
import { displayPayload, UPLINKS_PER_PAGE, useDeviceUplinks } from './use-uplinks'
import { modelLabel } from './use-device-models'
import {
  archiveDevice,
  displayDevEUI,
  fetchDevice,
  restoreDevice,
  type Device,
} from './use-devices'

const route = useRoute()
const { t } = useI18n()
const { language } = useLanguage()
const errorMessage = useErrorMessage()
const { copy, copied } = useClipboard({ legacy: true })

const device = shallowRef<Device>()
const loading = shallowRef(false)
const error = shallowRef<unknown>()

const id = computed(() => Number(route.params.id))

async function reload() {
  loading.value = true
  error.value = undefined
  try {
    device.value = await fetchDevice(id.value)
  } catch (err) {
    error.value = err
  } finally {
    loading.value = false
  }
}

watch(id, reload, { immediate: true })

const {
  uplinks,
  total: uplinkTotal,
  loading: uplinksLoading,
  error: uplinksError,
  load: loadUplinks,
} = useDeviceUplinks()
const uplinkPage = shallowRef(1)

watch([id, uplinkPage], ([deviceId, page]) => loadUplinks(deviceId, page), { immediate: true })

const uplinkHeaders = computed(() => [
  { title: t('uplinks.time'), key: 'received_at', sortable: false },
  { title: t('uplinks.fCnt'), key: 'f_cnt', sortable: false, align: 'end' as const },
  { title: t('uplinks.payload'), key: 'payload_hex', sortable: false },
])

const editOpen = shallowRef(false)
const busy = shallowRef(false)
const actionError = shallowRef('')

async function setArchived(archived: boolean) {
  busy.value = true
  actionError.value = ''
  try {
    if (archived) {
      await archiveDevice(id.value)
    } else {
      await restoreDevice(id.value)
    }
    await reload()
  } catch (err) {
    actionError.value = errorMessage(err)
  } finally {
    busy.value = false
  }
}

//------------------------------------------------------------------------------

const range = shallowRef<PulseRange>(30)
const pulses = shallowRef<PulseData>()
const pulsesError = shallowRef<unknown>()

async function loadPulses() {
  pulsesError.value = undefined
  try {
    pulses.value = await fetchPulses(id.value, range.value)
  } catch (err) {
    pulsesError.value = err
  }
}

const alarms = shallowRef<Alarm[]>([])
const alarmTotal = shallowRef(0)
const alarmPage = shallowRef(1)

async function loadAlarms() {
  try {
    const data = await fetchAlarms(id.value, alarmPage.value)
    alarms.value = data.alarms
    alarmTotal.value = data.total
  } catch {
    alarms.value = []
  }
}

// The readings load when the device has a decoder; they change with the
// range, the page, and the device.
watch(
  [() => device.value?.device_model.has_decoder, id, range],
  ([hasDecoder]) => hasDecoder && loadPulses(),
  { immediate: true },
)
watch(
  [() => device.value?.device_model.has_decoder, id, alarmPage],
  ([hasDecoder]) => hasDecoder && loadAlarms(),
  { immediate: true },
)

// dailyRows lists the days newest first for the table.
const dailyRows = computed(() => [...(pulses.value?.days ?? [])].reverse())

const alarmHeaders = computed(() => [
  { title: t('alarms.kind'), key: 'kind', sortable: false },
  { title: t('alarms.raised'), key: 'raised_at', sortable: false },
  { title: t('alarms.cleared'), key: 'cleared_at', sortable: false },
  { title: t('alarms.duration'), key: 'duration', sortable: false, align: 'end' as const },
])

const numbers = computed(() => new Intl.NumberFormat(language.value === 'ru' ? 'ru-MD' : 'ro-MD'))
</script>

<template>
  <div class="mx-auto flex max-w-5xl flex-col gap-6 p-6">
    <router-link :to="{ name: 'Home' }" class="app-link text-sm" data-test="back-to-devices">
      ← {{ $t('devices.title') }}
    </router-link>

    <v-progress-linear v-if="loading && !device" indeterminate color="primary" />
    <p v-else-if="error" class="text-error" data-test="device-load-error">
      {{ errorMessage(error) }}
    </p>

    <template v-else-if="device">
      <div
        v-if="device.archived_at"
        class="app-card flex items-center justify-between gap-4 p-4"
        data-test="archived-banner"
      >
        <span class="text-warning">{{ $t('devices.archivedBanner') }}</span>
        <v-btn
          size="small"
          variant="tonal"
          color="primary"
          :loading="busy"
          data-test="device-restore"
          @click="setArchived(false)"
        >
          {{ $t('devices.restore') }}
        </v-btn>
      </div>

      <div class="app-card">
        <div class="app-card-header flex items-center justify-between gap-4">
          <span class="flex flex-wrap items-center gap-2">
            <span data-test="device-title">{{ device.name }}</span>
            <v-chip
              v-for="kind in device.active_alarms"
              :key="kind"
              size="x-small"
              color="error"
              variant="flat"
              prepend-icon="mdi-alert"
              data-test="device-active-alarm"
            >
              {{ $t(`alarmKinds.${kind}`) }}
            </v-chip>
          </span>
          <div class="flex gap-2">
            <v-btn size="small" variant="text" prepend-icon="mdi-pencil" @click="editOpen = true">
              {{ $t('devices.edit') }}
            </v-btn>
            <v-btn
              v-if="!device.archived_at"
              size="small"
              variant="text"
              prepend-icon="mdi-archive-outline"
              :loading="busy"
              data-test="device-archive"
              @click="setArchived(true)"
            >
              {{ $t('devices.archive') }}
            </v-btn>
          </div>
        </div>
        <dl class="grid grid-cols-[10rem_1fr] gap-x-4 gap-y-2 p-4 text-body">
          <dt class="text-muted">{{ $t('devices.devEui') }}</dt>
          <dd class="flex items-center gap-2">
            <span class="font-mono" data-test="device-dev-eui">{{
              displayDevEUI(device.dev_eui)
            }}</span>
            <v-btn
              size="x-small"
              variant="text"
              :icon="copied ? 'mdi-check' : 'mdi-content-copy'"
              @click="copy(displayDevEUI(device.dev_eui))"
            />
          </dd>
          <dt class="text-muted">{{ $t('devices.lastUplink') }}</dt>
          <dd data-test="device-last-uplink">
            {{ device.last_uplink_at ? formatDateTime(device.last_uplink_at, language) : '—' }}
          </dd>
          <dt class="text-muted">{{ $t('devices.model') }}</dt>
          <dd data-test="device-model">{{ modelLabel(device.device_model) }}</dd>
          <dt class="text-muted">{{ $t('devices.meterType') }}</dt>
          <dd>{{ device.meter_type ? $t(`meterTypes.${device.meter_type}`) : '—' }}</dd>
          <dt class="text-muted">{{ $t('devices.address') }}</dt>
          <dd>{{ device.address ?? '—' }}</dd>
          <dt class="text-muted">{{ $t('devices.groups') }}</dt>
          <dd class="flex flex-wrap gap-1" data-test="device-groups">
            <v-chip
              v-for="g in device.groups"
              :key="g.id"
              size="small"
              variant="tonal"
              :to="{ name: 'Home', query: { group: g.id } }"
            >
              {{ g.name }}
            </v-chip>
            <span v-if="!device.groups.length">—</span>
          </dd>
          <dt class="text-muted">{{ $t('devices.note') }}</dt>
          <dd class="whitespace-pre-line">{{ device.note ?? '—' }}</dd>
        </dl>
        <p v-if="actionError" class="px-4 pb-4 text-error">{{ actionError }}</p>
      </div>

      <div class="app-card" data-test="section-pulses">
        <div class="app-card-header flex flex-wrap items-center justify-between gap-3">
          <span>{{ $t('devices.sections.pulses') }}</span>
          <v-btn-toggle
            v-if="device.device_model.has_decoder"
            v-model="range"
            mandatory
            density="compact"
            variant="text"
            divided
            class="border border-subtle"
            data-test="pulse-range"
          >
            <v-btn v-for="r in PULSE_RANGES" :key="r" :value="r" size="small">
              {{ $t('pulses.range', { n: r }) }}
            </v-btn>
          </v-btn-toggle>
        </div>
        <p
          v-if="!device.device_model.has_decoder"
          class="p-6 text-center text-muted"
          data-test="pulses-no-decoder"
        >
          {{ $t('devices.sections.needsDecoder') }}
        </p>
        <p v-else-if="pulsesError" class="p-4 text-error">{{ errorMessage(pulsesError) }}</p>
        <p
          v-else-if="pulses && !pulses.points.length"
          class="p-6 text-center text-muted"
          data-test="pulses-empty"
        >
          {{ $t('pulses.empty') }}
        </p>
        <div v-else-if="pulses" class="flex flex-col gap-6 p-4">
          <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <div>
              <div class="mb-2 text-sm text-muted">{{ $t('pulses.reported') }}</div>
              <PulseLineChart :points="pulses.points" :language="language" />
            </div>
            <div>
              <div class="mb-2 text-sm text-muted">{{ $t('pulses.perDay') }}</div>
              <DailyPulseChart :days="pulses.days" :language="language" />
            </div>
          </div>
          <table class="w-full text-sm" data-test="daily-table">
            <thead class="text-left text-muted">
              <tr>
                <th class="py-1 font-normal">{{ $t('pulses.day') }}</th>
                <th class="py-1 text-right font-normal">{{ $t('pulses.last') }}</th>
                <th class="py-1 text-right font-normal">{{ $t('pulses.delta') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="d in dailyRows" :key="d.day" class="border-t border-subtle text-body">
                <td class="py-1">{{ d.day }}</td>
                <td class="py-1 text-right font-mono">{{ numbers.format(d.last) }}</td>
                <td class="py-1 text-right font-mono" data-test="daily-delta">
                  {{ d.delta === null ? '—' : numbers.format(d.delta) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="app-card" data-test="section-alarms">
        <div class="app-card-header">
          {{ $t('devices.sections.alarms') }}
          <span v-if="alarmTotal" class="ml-1 text-muted">({{ alarmTotal }})</span>
        </div>
        <p v-if="!device.device_model.has_decoder" class="p-6 text-center text-muted">
          {{ $t('devices.sections.needsDecoder') }}
        </p>
        <p v-else-if="!alarmTotal" class="p-6 text-center text-muted" data-test="alarms-empty">
          {{ $t('alarms.empty') }}
        </p>
        <v-data-table-server
          v-else
          v-model:page="alarmPage"
          :headers="alarmHeaders"
          :items="alarms"
          :items-length="alarmTotal"
          :items-per-page="UPLINKS_PER_PAGE"
          :items-per-page-options="[UPLINKS_PER_PAGE]"
          item-value="id"
          density="compact"
          data-test="alarms-table"
        >
          <template #[`item.kind`]="{ item }">
            <span class="text-body" data-test="alarm-kind">{{
              $t(`alarmKinds.${item.kind}`)
            }}</span>
          </template>
          <template #[`item.raised_at`]="{ item }">
            {{ formatDateTime(item.raised_at, language) }}
          </template>
          <template #[`item.cleared_at`]="{ item }">
            <span v-if="item.cleared_at">{{ formatDateTime(item.cleared_at, language) }}</span>
            <v-chip v-else size="x-small" color="error" variant="flat" data-test="alarm-active">
              {{ $t('alarms.active') }}
            </v-chip>
          </template>
          <template #[`item.duration`]="{ item }">
            <span class="text-muted">{{ formatDuration(item.raised_at, item.cleared_at) }}</span>
          </template>
        </v-data-table-server>
      </div>

      <div class="app-card" data-test="section-uplinks">
        <div class="app-card-header">
          {{ $t('devices.sections.uplinks') }}
          <span v-if="uplinkTotal" class="ml-1 text-muted">({{ uplinkTotal }})</span>
        </div>
        <p v-if="uplinksError" class="p-4 text-error">{{ errorMessage(uplinksError) }}</p>
        <p
          v-else-if="!uplinksLoading && uplinkTotal === 0"
          class="p-6 text-center text-muted"
          data-test="uplinks-empty"
        >
          {{ $t('devices.sections.noUplinks') }}
        </p>
        <v-data-table-server
          v-else
          v-model:page="uplinkPage"
          :headers="uplinkHeaders"
          :items="uplinks"
          :items-length="uplinkTotal"
          :loading="uplinksLoading"
          :items-per-page="UPLINKS_PER_PAGE"
          :items-per-page-options="[UPLINKS_PER_PAGE]"
          item-value="id"
          density="compact"
          data-test="uplinks-table"
        >
          <template #[`item.received_at`]="{ item }">
            <span class="text-body">{{ formatDateTime(item.received_at, language) }}</span>
          </template>
          <template #[`item.payload_hex`]="{ item }">
            <span class="font-mono text-body" data-test="uplink-payload">
              {{ displayPayload(item.payload_hex) }}
            </span>
          </template>
        </v-data-table-server>
      </div>

      <DeviceDialog v-model="editOpen" :device="device" @saved="reload" />
    </template>
  </div>
</template>
