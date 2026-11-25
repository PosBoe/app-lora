<script setup lang="ts">
import DeviceModelDialog from './DeviceModelDialog.vue'
import { modelLabel, useDeviceModels } from './use-device-models'

// DeviceModelPicker selects a device model of the organization, and creates a
// new one in place when the list has none that fits.
const selected = defineModel<number | null>({ required: true })

const { t } = useI18n()
const errorMessage = useErrorMessage()
const { models, loading, create } = useDeviceModels()

const items = computed(() => models.value.map((m) => ({ title: modelLabel(m), value: m.id })))

const createOpen = shallowRef(false)
const creating = shallowRef(false)
const createError = shallowRef('')

function openCreate() {
  createError.value = ''
  createOpen.value = true
}

async function submitCreate(vendor: string, model: string) {
  creating.value = true
  createError.value = ''
  try {
    const created = await create(vendor, model)
    selected.value = created.id
    createOpen.value = false
  } catch (err) {
    createError.value = errorMessage(err)
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="flex items-start gap-2">
    <v-select
      v-model="selected"
      :items="items"
      :label="t('devices.model')"
      :loading="loading"
      :no-data-text="t('models.empty')"
      density="compact"
      variant="outlined"
      hide-details
      class="flex-1 min-w-0"
      data-test="model-select"
    />
    <v-btn
      variant="tonal"
      height="40"
      prepend-icon="mdi-plus"
      data-test="model-new"
      @click="openCreate"
    >
      {{ t('models.add') }}
    </v-btn>

    <DeviceModelDialog
      v-model="createOpen"
      :title="t('models.createTitle')"
      :loading="creating"
      :error="createError"
      @submit="submitCreate"
    />
  </div>
</template>
