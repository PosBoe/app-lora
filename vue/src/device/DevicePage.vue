<script setup lang="ts">
import DeviceDialog from './DeviceDialog.vue'
import { modelLabel } from './use-device-models'
import {
  archiveDevice,
  displayDevEUI,
  fetchDevice,
  restoreDevice,
  type Device,
} from './use-devices'

const route = useRoute()
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
          <span data-test="device-title">{{ device.name }}</span>
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
        <div class="app-card-header">{{ $t('devices.sections.pulses') }}</div>
        <p class="p-6 text-center text-muted">{{ $t('devices.sections.needsDecoder') }}</p>
      </div>

      <div class="app-card" data-test="section-alarms">
        <div class="app-card-header">{{ $t('devices.sections.alarms') }}</div>
        <p class="p-6 text-center text-muted">{{ $t('devices.sections.needsDecoder') }}</p>
      </div>

      <div class="app-card" data-test="section-uplinks">
        <div class="app-card-header">{{ $t('devices.sections.uplinks') }}</div>
        <p class="p-6 text-center text-muted">{{ $t('devices.sections.noUplinks') }}</p>
      </div>

      <DeviceDialog v-model="editOpen" :device="device" @saved="reload" />
    </template>
  </div>
</template>
