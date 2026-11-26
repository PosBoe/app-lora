<script setup lang="ts">
import { useAdminModels, type AdminModel } from './use-admin-models'

const { t } = useI18n()
const errorMessage = useErrorMessage()
const { models, decoders, loading, error, reload, setDecoder } = useAdminModels()

const headers = computed(() => [
  { title: t('adminModels.organization'), key: 'organization_name' },
  { title: t('models.vendor'), key: 'vendor' },
  { title: t('models.model'), key: 'model' },
  { title: t('models.devices'), key: 'device_count', align: 'end' as const },
  { title: t('adminModels.decoder'), key: 'decoder', sortable: false, width: 280 },
])

const decoderItems = computed(() => decoders.value.map((d) => ({ title: d.name, value: d.id })))

const saving = shallowRef<number>()
const saveError = shallowRef('')

async function choose(m: AdminModel, decoder: string | null) {
  saving.value = m.id
  saveError.value = ''
  try {
    await setDecoder(m.id, decoder)
  } catch (err) {
    saveError.value = errorMessage(err)
  } finally {
    saving.value = undefined
  }
}

// While uplinks wait for the decode worker, the page looks again after a
// few seconds.
const decoding = computed(() => models.value.some((m) => m.pending_uplinks > 0))
useIntervalFn(() => decoding.value && reload(), 3000)
</script>

<template>
  <div class="mx-auto max-w-7xl p-6">
    <div class="app-card">
      <div class="app-card-header">{{ $t('adminModels.title') }}</div>
      <p class="px-4 pt-3 text-muted">{{ $t('adminModels.intro') }}</p>
      <p v-if="saveError" class="px-4 pt-2 text-error" data-test="admin-model-error">
        {{ saveError }}
      </p>

      <div v-if="error" class="flex items-center gap-3 p-4">
        <span class="text-error">{{ errorMessage(error) }}</span>
        <v-btn size="small" variant="text" @click="reload">{{ $t('common.retry') }}</v-btn>
      </div>

      <v-data-table
        v-else
        :headers="headers"
        :items="models"
        :loading="loading && !models.length"
        :no-data-text="$t('adminModels.empty')"
        item-value="id"
        density="compact"
        :items-per-page="100"
        data-test="admin-models-table"
      >
        <template #[`item.decoder`]="{ item }">
          <div class="flex items-center gap-2 py-1">
            <v-select
              :model-value="item.decoder"
              :items="decoderItems"
              :placeholder="$t('adminModels.noDecoder')"
              :loading="saving === item.id"
              clearable
              density="compact"
              variant="outlined"
              hide-details
              class="flex-1 min-w-0"
              :data-test="`decoder-select-${item.id}`"
              @update:model-value="(v: string | null) => choose(item, v)"
            />
            <span
              v-if="item.pending_uplinks"
              class="whitespace-nowrap text-sm text-muted"
              :data-test="`decoding-${item.id}`"
            >
              {{ $t('adminModels.decoding', { n: item.pending_uplinks }) }}
            </span>
          </div>
        </template>
      </v-data-table>
    </div>
  </div>
</template>
