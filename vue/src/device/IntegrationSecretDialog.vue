<script setup lang="ts">
import type { IntegrationKind } from './use-integrations'

// IntegrationSecretDialog shows the ingest URL and the secret of an
// integration one time, with the setup steps of its network server.
const props = defineProps<{
  kind: IntegrationKind
  url: string
  secret: string
}>()

const open = defineModel<boolean>({ required: true })

const { tm, rt } = useI18n()
const header = computed(() => `Bearer ${props.secret}`)
const steps = computed(() =>
  (tm(`integrations.steps.${props.kind}`) as unknown[]).map((s) => rt(s as never)),
)

const { copy, copied } = useClipboard({ legacy: true })
const copiedWhat = shallowRef('')

function copyValue(what: string, value: string) {
  copiedWhat.value = what
  void copy(value)
}
</script>

<template>
  <v-dialog v-model="open" max-width="640" persistent>
    <div class="app-card" data-test="integration-secret-dialog">
      <div class="app-card-header">{{ $t('integrations.secretTitle') }}</div>
      <div class="flex flex-col gap-4 p-4 text-body">
        <p class="text-warning">{{ $t('integrations.secretOnce') }}</p>
        <dl class="grid grid-cols-[9rem_1fr] items-center gap-x-3 gap-y-2">
          <dt class="text-muted">{{ $t('integrations.url') }}</dt>
          <dd class="flex min-w-0 items-center gap-2">
            <span class="truncate font-mono text-sm" data-test="integration-url">{{ url }}</span>
            <v-btn
              size="x-small"
              variant="text"
              :icon="copied && copiedWhat === 'url' ? 'mdi-check' : 'mdi-content-copy'"
              @click="copyValue('url', url)"
            />
          </dd>
          <dt class="text-muted">{{ $t('integrations.headerName') }}</dt>
          <dd class="font-mono text-sm">Authorization</dd>
          <dt class="text-muted">{{ $t('integrations.headerValue') }}</dt>
          <dd class="flex min-w-0 items-center gap-2">
            <span class="break-all font-mono text-sm text-strong" data-test="integration-secret">
              {{ header }}
            </span>
            <v-btn
              size="x-small"
              variant="text"
              :icon="copied && copiedWhat === 'secret' ? 'mdi-check' : 'mdi-content-copy'"
              data-test="copy-secret"
              @click="copyValue('secret', header)"
            />
          </dd>
        </dl>
        <div>
          <div class="mb-2 font-medium text-strong">{{ $t('integrations.stepsTitle') }}</div>
          <ol class="list-decimal space-y-1 pl-5" data-test="integration-steps">
            <li v-for="(step, i) in steps" :key="i">{{ step }}</li>
          </ol>
        </div>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="flat" color="primary" data-test="secret-close" @click="open = false">
          {{ $t('common.close') }}
        </v-btn>
      </div>
    </div>
  </v-dialog>
</template>
