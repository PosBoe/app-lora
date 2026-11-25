<script setup lang="ts">
import DeviceModelDialog from './DeviceModelDialog.vue'
import { modelLabel, useDeviceModels, type DeviceModel } from './use-device-models'

const { t } = useI18n()
const errorMessage = useErrorMessage()
const { models, loading, error, reload, create, rename, remove } = useDeviceModels()

const headers = computed(() => [
  { title: t('models.vendor'), key: 'vendor' },
  { title: t('models.model'), key: 'model' },
  { title: t('models.devices'), key: 'device_count', align: 'end' as const },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
])

// editing is the model the dialog renames, or undefined to create one.
const editing = shallowRef<DeviceModel>()
const dialogOpen = shallowRef(false)
const saving = shallowRef(false)
const dialogError = shallowRef('')

function openDialog(m?: DeviceModel) {
  editing.value = m
  dialogError.value = ''
  dialogOpen.value = true
}

async function submit(vendor: string, model: string) {
  saving.value = true
  dialogError.value = ''
  try {
    if (editing.value) {
      await rename(editing.value.id, vendor, model)
    } else {
      await create(vendor, model)
    }
    dialogOpen.value = false
  } catch (err) {
    dialogError.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}

const deleting = shallowRef<DeviceModel>()
const deleteOpen = shallowRef(false)
const deleteBusy = shallowRef(false)
const deleteError = shallowRef('')

function askDelete(m: DeviceModel) {
  deleting.value = m
  deleteError.value = ''
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deleting.value) {
    return
  }
  deleteBusy.value = true
  deleteError.value = ''
  try {
    await remove(deleting.value.id)
    deleteOpen.value = false
  } catch (err) {
    deleteError.value = errorMessage(err)
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-4xl p-6">
    <div class="app-card">
      <div class="app-card-header flex items-center justify-between">
        <span>{{ $t('models.title') }}</span>
        <v-btn
          variant="flat"
          color="primary"
          size="small"
          prepend-icon="mdi-plus"
          data-test="model-add"
          @click="openDialog()"
        >
          {{ $t('models.add') }}
        </v-btn>
      </div>

      <div v-if="error" class="flex items-center gap-3 p-4">
        <span class="text-error">{{ errorMessage(error) }}</span>
        <v-btn size="small" variant="text" @click="reload">{{ $t('common.retry') }}</v-btn>
      </div>

      <v-data-table
        v-else
        :headers="headers"
        :items="models"
        :loading="loading"
        :no-data-text="$t('models.empty')"
        item-value="id"
        density="compact"
        :items-per-page="-1"
        hide-default-footer
        data-test="models-table"
      >
        <template #[`item.actions`]="{ item }">
          <v-btn
            size="small"
            variant="text"
            icon="mdi-pencil"
            :title="$t('models.rename')"
            @click="openDialog(item)"
          />
          <v-btn
            size="small"
            variant="text"
            icon="mdi-delete-outline"
            :title="item.device_count ? $t('models.inUse') : $t('models.delete')"
            :disabled="item.device_count > 0"
            :data-test="`model-delete-${item.id}`"
            @click="askDelete(item)"
          />
        </template>
      </v-data-table>
    </div>

    <DeviceModelDialog
      v-model="dialogOpen"
      :title="editing ? $t('models.renameTitle') : $t('models.createTitle')"
      :initial="editing"
      :loading="saving"
      :error="dialogError"
      @submit="submit"
    />

    <ConfirmDialog
      v-model="deleteOpen"
      :title="$t('models.delete')"
      :text="deleting ? $t('models.deleteConfirm', { name: modelLabel(deleting) }) : ''"
      :confirm-text="$t('models.delete')"
      :loading="deleteBusy"
      :error="deleteError"
      @confirm="confirmDelete"
    />
  </div>
</template>
