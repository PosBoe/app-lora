<script setup lang="ts">
import DeviceGroupDialog from './DeviceGroupDialog.vue'
import { useDeviceGroups, type DeviceGroup } from './use-device-groups'

const { t } = useI18n()
const errorMessage = useErrorMessage()
const { groups, loading, error, reload, create, rename, remove } = useDeviceGroups()

const headers = computed(() => [
  { title: t('groups.name'), key: 'name' },
  { title: t('groups.devices'), key: 'device_count', align: 'end' as const },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
])

// editing is the group the dialog renames, or undefined to create one.
const editing = shallowRef<DeviceGroup>()
const dialogOpen = shallowRef(false)
const saving = shallowRef(false)
const dialogError = shallowRef('')

function openDialog(g?: DeviceGroup) {
  editing.value = g
  dialogError.value = ''
  dialogOpen.value = true
}

async function submit(name: string) {
  saving.value = true
  dialogError.value = ''
  try {
    if (editing.value) {
      await rename(editing.value.id, name)
    } else {
      await create(name)
    }
    dialogOpen.value = false
  } catch (err) {
    dialogError.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}

const deleting = shallowRef<DeviceGroup>()
const deleteOpen = shallowRef(false)
const deleteBusy = shallowRef(false)
const deleteError = shallowRef('')

function askDelete(g: DeviceGroup) {
  deleting.value = g
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
  <div class="mx-auto max-w-3xl p-6">
    <div class="app-card">
      <div class="app-card-header flex items-center justify-between">
        <span>{{ $t('groups.title') }}</span>
        <v-btn
          variant="flat"
          color="primary"
          size="small"
          prepend-icon="mdi-plus"
          data-test="group-add"
          @click="openDialog()"
        >
          {{ $t('groups.add') }}
        </v-btn>
      </div>

      <div v-if="error" class="flex items-center gap-3 p-4">
        <span class="text-error">{{ errorMessage(error) }}</span>
        <v-btn size="small" variant="text" @click="reload">{{ $t('common.retry') }}</v-btn>
      </div>

      <v-data-table
        v-else
        :headers="headers"
        :items="groups"
        :loading="loading"
        :no-data-text="$t('groups.empty')"
        item-value="id"
        density="compact"
        :items-per-page="-1"
        hide-default-footer
        data-test="groups-table"
      >
        <template #[`item.name`]="{ item }">
          <router-link
            :to="{ name: 'Home', query: { group: item.id } }"
            class="app-link"
            data-test="group-row-name"
          >
            {{ item.name }}
          </router-link>
        </template>
        <template #[`item.actions`]="{ item }">
          <v-btn
            size="small"
            variant="text"
            icon="mdi-pencil"
            :title="$t('groups.rename')"
            @click="openDialog(item)"
          />
          <v-btn
            size="small"
            variant="text"
            icon="mdi-delete-outline"
            :title="$t('groups.delete')"
            :data-test="`group-delete-${item.id}`"
            @click="askDelete(item)"
          />
        </template>
      </v-data-table>
    </div>

    <DeviceGroupDialog
      v-model="dialogOpen"
      :title="editing ? $t('groups.renameTitle') : $t('groups.createTitle')"
      :initial="editing?.name"
      :loading="saving"
      :error="dialogError"
      @submit="submit"
    />

    <ConfirmDialog
      v-model="deleteOpen"
      :title="$t('groups.delete')"
      :text="deleting ? $t('groups.deleteConfirm', { name: deleting.name }) : ''"
      :confirm-text="$t('groups.delete')"
      :loading="deleteBusy"
      :error="deleteError"
      @confirm="confirmDelete"
    />
  </div>
</template>
