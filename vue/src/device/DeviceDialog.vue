<script setup lang="ts">
import { apiErrorCode } from '@/lib/api'

import DeviceModelPicker from './DeviceModelPicker.vue'
import {
  createDevice,
  displayDevEUI,
  findArchived,
  METER_TYPES,
  restoreDevice,
  updateDevice,
  type Device,
  type MeterType,
} from './use-devices'

// DeviceDialog adds a device, or edits `device` when it is set. It emits
// `saved` with the device that the user saved or restored.
const props = defineProps<{ device?: Device }>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ saved: [device: Device | { id: number }] }>()

const { t } = useI18n()
const errorMessage = useErrorMessage()

const devEui = shallowRef('')
const name = shallowRef('')
const modelId = shallowRef<number | null>(null)
const meterType = shallowRef<MeterType | null>(null)
const address = shallowRef('')
const note = shallowRef('')

const saving = shallowRef(false)
const error = shallowRef('')
// archivedDevEUI is set when the DevEUI belongs to an archived device, which
// the user can restore in place of adding a new one.
const archivedDevEUI = shallowRef('')

const meterItems = computed(() =>
  METER_TYPES.map((v) => ({ title: t(`meterTypes.${v}`), value: v })),
)

watch(
  open,
  (isOpen) => {
    if (!isOpen) {
      return
    }
    const d = props.device
    devEui.value = d ? displayDevEUI(d.dev_eui) : ''
    name.value = d?.name ?? ''
    modelId.value = d?.device_model.id ?? null
    meterType.value = d?.meter_type ?? null
    address.value = d?.address ?? ''
    note.value = d?.note ?? ''
    error.value = ''
    archivedDevEUI.value = ''
  },
  { immediate: true },
)

async function submit() {
  if (saving.value || modelId.value === null) {
    return
  }
  saving.value = true
  error.value = ''
  archivedDevEUI.value = ''
  const input = {
    dev_eui: devEui.value,
    name: name.value,
    device_model_id: modelId.value,
    meter_type: meterType.value ?? ('' as const),
    address: address.value,
    note: note.value,
  }
  try {
    const saved = props.device
      ? await updateDevice(props.device.id, input)
      : await createDevice(input)
    open.value = false
    emit('saved', saved)
  } catch (err) {
    error.value = errorMessage(err)
    if (apiErrorCode(err) === 'dev_eui_archived') {
      archivedDevEUI.value = devEui.value
    }
  } finally {
    saving.value = false
  }
}

async function restoreArchived() {
  saving.value = true
  try {
    const archived = await findArchived(archivedDevEUI.value)
    if (!archived) {
      error.value = t('errors.not_found')
      return
    }
    await restoreDevice(archived.id)
    open.value = false
    emit('saved', archived)
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <v-dialog v-model="open" max-width="520">
    <form class="app-card" data-test="device-dialog" @submit.prevent="submit">
      <div class="app-card-header">
        {{ device ? $t('devices.editTitle') : $t('devices.addTitle') }}
      </div>
      <div class="flex flex-col gap-4 p-4">
        <v-text-field
          v-model="devEui"
          :label="$t('devices.devEui')"
          :hint="$t('devices.devEuiHint')"
          persistent-hint
          autofocus
          autocomplete="off"
          class="font-mono"
          density="compact"
          variant="outlined"
          data-test="dev-eui-input"
        />
        <v-text-field
          v-model="name"
          :label="$t('devices.name')"
          density="compact"
          variant="outlined"
          hide-details
          data-test="device-name-input"
        />
        <DeviceModelPicker v-model="modelId" />
        <v-select
          v-model="meterType"
          :items="meterItems"
          :label="$t('devices.meterType')"
          clearable
          density="compact"
          variant="outlined"
          hide-details
          data-test="meter-type-select"
        />
        <v-text-field
          v-model="address"
          :label="$t('devices.address')"
          density="compact"
          variant="outlined"
          hide-details
          data-test="address-input"
        />
        <v-textarea
          v-model="note"
          :label="$t('devices.note')"
          rows="2"
          auto-grow
          density="compact"
          variant="outlined"
          hide-details
          data-test="note-input"
        />
        <div v-if="error" class="flex items-center gap-3">
          <p class="flex-1 min-w-0 text-error" data-test="device-error">{{ error }}</p>
          <v-btn
            v-if="archivedDevEUI"
            size="small"
            variant="tonal"
            color="primary"
            :loading="saving"
            data-test="device-restore-archived"
            @click="restoreArchived"
          >
            {{ $t('devices.restore') }}
          </v-btn>
        </div>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="open = false">{{ $t('common.cancel') }}</v-btn>
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          :loading="saving"
          :disabled="!devEui.trim() || !name.trim() || modelId === null"
          data-test="device-submit"
        >
          {{ $t('common.save') }}
        </v-btn>
      </div>
    </form>
  </v-dialog>
</template>
