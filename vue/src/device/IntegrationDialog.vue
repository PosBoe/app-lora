<script setup lang="ts">
import { INTEGRATION_KINDS, type IntegrationKind } from './use-integrations'

// IntegrationDialog asks for the name and, for a new integration, the
// network server kind. The parent runs the action on `submit`.
const props = defineProps<{
  title: string
  // initialName is set to rename an integration; the kind then stays.
  initialName?: string
  loading?: boolean
  error?: string
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ submit: [name: string, kind: IntegrationKind] }>()

const { t } = useI18n()
const name = shallowRef('')
const kind = shallowRef<IntegrationKind>('chirpstack')

const kindItems = computed(() =>
  INTEGRATION_KINDS.map((k) => ({ title: t(`integrations.kinds.${k}`), value: k })),
)

watch(
  open,
  (isOpen) => {
    if (isOpen) {
      name.value = props.initialName ?? ''
      kind.value = 'chirpstack'
    }
  },
  { immediate: true },
)

function submit() {
  if (!props.loading) {
    emit('submit', name.value, kind.value)
  }
}
</script>

<template>
  <v-dialog v-model="open" max-width="440">
    <form class="app-card" data-test="integration-dialog" @submit.prevent="submit">
      <div class="app-card-header">{{ title }}</div>
      <div class="flex flex-col gap-4 p-4">
        <v-text-field
          v-model="name"
          :label="$t('integrations.name')"
          :placeholder="$t('integrations.nameExample')"
          autofocus
          density="compact"
          variant="outlined"
          hide-details
          data-test="integration-name-input"
        />
        <v-select
          v-if="initialName === undefined"
          v-model="kind"
          :items="kindItems"
          :label="$t('integrations.kind')"
          density="compact"
          variant="outlined"
          hide-details
          data-test="integration-kind-select"
        />
        <p v-if="error" class="text-error" data-test="integration-error">{{ error }}</p>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="open = false">{{ $t('common.cancel') }}</v-btn>
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          :loading="loading"
          :disabled="!name.trim()"
          data-test="integration-submit"
        >
          {{ $t('common.save') }}
        </v-btn>
      </div>
    </form>
  </v-dialog>
</template>
