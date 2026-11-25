<script setup lang="ts">
import DeviceGroupDialog from './DeviceGroupDialog.vue'
import { useDeviceGroups } from './use-device-groups'

// DeviceGroupPicker selects the groups of a device, and creates a new group
// in place.
const selected = defineModel<number[]>({ required: true })

const { t } = useI18n()
const errorMessage = useErrorMessage()
const { groups, loading, create } = useDeviceGroups()

const items = computed(() => groups.value.map((g) => ({ title: g.name, value: g.id })))

const createOpen = shallowRef(false)
const creating = shallowRef(false)
const createError = shallowRef('')

function openCreate() {
  createError.value = ''
  createOpen.value = true
}

async function submitCreate(name: string) {
  creating.value = true
  createError.value = ''
  try {
    const created = await create(name)
    selected.value = [...selected.value, created.id]
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
    <v-autocomplete
      v-model="selected"
      :items="items"
      :label="t('devices.groups')"
      :loading="loading"
      :no-data-text="t('groups.empty')"
      multiple
      chips
      closable-chips
      density="compact"
      variant="outlined"
      hide-details
      class="flex-1 min-w-0"
      data-test="group-select"
    />
    <v-btn
      variant="tonal"
      height="40"
      prepend-icon="mdi-plus"
      data-test="group-new"
      @click="openCreate"
    >
      {{ t('groups.add') }}
    </v-btn>

    <DeviceGroupDialog
      v-model="createOpen"
      :title="t('groups.createTitle')"
      :loading="creating"
      :error="createError"
      @submit="submitCreate"
    />
  </div>
</template>
