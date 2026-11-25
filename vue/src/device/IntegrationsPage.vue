<script setup lang="ts">
import { formatDateTime } from '@/lib/i18n'

import IntegrationDialog from './IntegrationDialog.vue'
import IntegrationSecretDialog from './IntegrationSecretDialog.vue'
import { useIntegrations, type Integration, type IntegrationKind } from './use-integrations'

const { t, te } = useI18n()
const { language } = useLanguage()
const errorMessage = useErrorMessage()
const { integrations, loading, error, reload, create, rename, regenerate, remove } =
  useIntegrations()

function time(iso: string): string {
  return formatDateTime(iso, language.value)
}

// lastErrorText names the last error of an integration in the UI language.
function lastErrorText(i: Integration): string {
  if (!i.last_error) {
    return ''
  }
  const key = `errors.${i.last_error.code}`
  return te(key) ? t(key) : i.last_error.message
}

//------------------------------------------------------------------------------

// editing is the integration the dialog renames, or undefined to create one.
const editing = shallowRef<Integration>()
const dialogOpen = shallowRef(false)
const saving = shallowRef(false)
const dialogError = shallowRef('')

function openDialog(i?: Integration) {
  editing.value = i
  dialogError.value = ''
  dialogOpen.value = true
}

const secretOpen = shallowRef(false)
const secretKind = shallowRef<IntegrationKind>('chirpstack')
const secretURL = shallowRef('')
const secret = shallowRef('')

function showSecret(i: Integration, value: string) {
  secretKind.value = i.kind
  secretURL.value = i.ingest_url
  secret.value = value
  secretOpen.value = true
}

async function submit(name: string, kind: IntegrationKind) {
  saving.value = true
  dialogError.value = ''
  try {
    if (editing.value) {
      await rename(editing.value.id, name)
      dialogOpen.value = false
    } else {
      const created = await create(name, kind)
      dialogOpen.value = false
      showSecret(created.integration, created.secret)
    }
  } catch (err) {
    dialogError.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}

//------------------------------------------------------------------------------

type Pending = { kind: 'regenerate' | 'delete'; integration: Integration }

const pending = shallowRef<Pending>()
const confirmOpen = shallowRef(false)
const confirming = shallowRef(false)
const confirmError = shallowRef('')

function ask(next: Pending) {
  pending.value = next
  confirmError.value = ''
  confirmOpen.value = true
}

async function confirm() {
  const p = pending.value
  if (!p) {
    return
  }
  confirming.value = true
  confirmError.value = ''
  try {
    if (p.kind === 'regenerate') {
      const value = await regenerate(p.integration.id)
      confirmOpen.value = false
      showSecret(p.integration, value)
    } else {
      await remove(p.integration.id)
      confirmOpen.value = false
    }
  } catch (err) {
    confirmError.value = errorMessage(err)
  } finally {
    confirming.value = false
  }
}
</script>

<template>
  <div class="mx-auto flex max-w-5xl flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <div>
        <div class="text-lg font-medium text-strong">{{ $t('integrations.title') }}</div>
        <p class="text-muted">{{ $t('integrations.intro') }}</p>
      </div>
      <v-btn
        variant="flat"
        color="primary"
        size="small"
        prepend-icon="mdi-plus"
        data-test="integration-add"
        @click="openDialog()"
      >
        {{ $t('integrations.add') }}
      </v-btn>
    </div>

    <v-progress-linear v-if="loading && !integrations.length" indeterminate color="primary" />

    <div v-if="error" class="app-card flex items-center gap-3 p-4">
      <span class="text-error">{{ errorMessage(error) }}</span>
      <v-btn size="small" variant="text" @click="reload">{{ $t('common.retry') }}</v-btn>
    </div>

    <div
      v-else-if="!loading && !integrations.length"
      class="app-card p-10 text-center text-muted"
      data-test="integrations-empty"
    >
      {{ $t('integrations.empty') }}
    </div>

    <div v-for="i in integrations" :key="i.id" class="app-card" :data-test="`integration-${i.id}`">
      <div class="app-card-header flex items-center justify-between gap-4">
        <span class="flex items-center gap-2">
          <span data-test="integration-title">{{ i.name }}</span>
          <v-chip size="x-small" variant="tonal">{{ $t(`integrations.kinds.${i.kind}`) }}</v-chip>
        </span>
        <v-menu location="bottom end">
          <template #activator="{ props: menu }">
            <v-btn v-bind="menu" icon="mdi-dots-vertical" size="small" variant="text" />
          </template>
          <v-list density="compact">
            <v-list-item
              prepend-icon="mdi-pencil"
              :title="$t('integrations.rename')"
              @click="openDialog(i)"
            />
            <v-list-item
              prepend-icon="mdi-key-change"
              :title="$t('integrations.regenerate')"
              data-test="integration-regenerate"
              @click="ask({ kind: 'regenerate', integration: i })"
            />
            <v-list-item
              prepend-icon="mdi-delete-outline"
              :title="$t('integrations.delete')"
              data-test="integration-delete"
              @click="ask({ kind: 'delete', integration: i })"
            />
          </v-list>
        </v-menu>
      </div>
      <dl class="grid grid-cols-[10rem_1fr] gap-x-4 gap-y-2 p-4 text-body">
        <dt class="text-muted">{{ $t('integrations.url') }}</dt>
        <dd class="truncate font-mono text-sm">{{ i.ingest_url }}</dd>
        <dt class="text-muted">{{ $t('integrations.lastUplink') }}</dt>
        <dd data-test="integration-last-uplink">
          <span v-if="i.last_uplink_at">{{ time(i.last_uplink_at) }}</span>
          <span v-else class="text-warning">{{ $t('integrations.noUplinkYet') }}</span>
        </dd>
        <dt class="text-muted">{{ $t('integrations.recent') }}</dt>
        <dd data-test="integration-recent">{{ i.uplinks_24h }}</dd>
        <template v-if="i.last_error">
          <dt class="text-muted">{{ $t('integrations.lastError') }}</dt>
          <dd class="text-error" data-test="integration-last-error">
            {{ time(i.last_error.at) }} — {{ lastErrorText(i) }}
          </dd>
        </template>
      </dl>
    </div>

    <IntegrationDialog
      v-model="dialogOpen"
      :title="editing ? $t('integrations.renameTitle') : $t('integrations.createTitle')"
      :initial-name="editing?.name"
      :loading="saving"
      :error="dialogError"
      @submit="submit"
    />

    <IntegrationSecretDialog
      v-model="secretOpen"
      :kind="secretKind"
      :url="secretURL"
      :secret="secret"
    />

    <ConfirmDialog
      v-model="confirmOpen"
      :title="
        pending?.kind === 'delete' ? $t('integrations.delete') : $t('integrations.regenerate')
      "
      :text="
        pending
          ? $t(
              pending.kind === 'delete'
                ? 'integrations.deleteConfirm'
                : 'integrations.regenerateConfirm',
              { name: pending.integration.name },
            )
          : ''
      "
      :confirm-text="$t('common.confirm')"
      :loading="confirming"
      :error="confirmError"
      @confirm="confirm"
    />
  </div>
</template>
