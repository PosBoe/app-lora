<script setup lang="ts">
import { formatDateTime } from '@/lib/i18n'

import OrganizationCreateDialog from './OrganizationCreateDialog.vue'
import TemporaryPasswordDialog from './TemporaryPasswordDialog.vue'
import { useOrganizations, type Member, type Organization } from './use-organizations'

const { t } = useI18n()
const { language } = useLanguage()
const errorMessage = useErrorMessage()
const { organizations, loading, error, reload, create, resetPassword, setActive } =
  useOrganizations()

const headers = computed(() => [
  { title: t('organizations.name'), key: 'name' },
  { title: t('organizations.status'), key: 'status', sortable: false },
  { title: t('organizations.login'), key: 'login', sortable: false },
  { title: t('organizations.profile'), key: 'profile', sortable: false },
  { title: t('organizations.lastLogin'), key: 'last_login', sortable: false },
  { title: t('organizations.devices'), key: 'device_count', align: 'end' as const },
  { title: t('organizations.created'), key: 'created_at' },
  { title: t('organizations.actions'), key: 'actions', sortable: false, align: 'end' as const },
])

// firstUser is the user of an organization. The MVP has one user for each.
function firstUser(org: Organization): Member | undefined {
  return org.users[0]
}

function profileText(m: Member | undefined): string {
  if (!m) {
    return ''
  }
  const name = [m.first_name, m.last_name].filter(Boolean).join(' ')
  return [name, m.email, m.phone].filter(Boolean).join(' · ')
}

function time(iso: string | null): string {
  return iso ? formatDateTime(iso, language.value) : t('organizations.never')
}

//------------------------------------------------------------------------------

const createOpen = shallowRef(false)
const creating = shallowRef(false)
const createError = shallowRef('')

const shownOpen = shallowRef(false)
const shownLogin = shallowRef('')
const shownPassword = shallowRef('')

function showPassword(login: string, password: string) {
  shownLogin.value = login
  shownPassword.value = password
  shownOpen.value = true
}

function openCreate() {
  createError.value = ''
  createOpen.value = true
}

async function submitCreate(name: string, login: string) {
  creating.value = true
  createError.value = ''
  try {
    const created = await create(name, login)
    createOpen.value = false
    showPassword(created.organization.users[0]?.login ?? login, created.temporary_password)
  } catch (err) {
    createError.value = errorMessage(err)
  } finally {
    creating.value = false
  }
}

//------------------------------------------------------------------------------

type Pending =
  | { kind: 'reset'; org: Organization; member: Member }
  | { kind: 'deactivate' | 'reactivate'; org: Organization }

const pending = shallowRef<Pending>()
const confirmOpen = shallowRef(false)
const confirming = shallowRef(false)
const confirmError = shallowRef('')

const confirmText = computed(() => {
  const p = pending.value
  if (!p) {
    return ''
  }
  if (p.kind === 'reset') {
    return t('organizations.resetConfirm', { login: p.member.login })
  }
  const key = p.kind === 'deactivate' ? 'deactivateConfirm' : 'reactivateConfirm'
  return t(`organizations.${key}`, { name: p.org.name })
})

const confirmTitle = computed(() => {
  const kind = pending.value?.kind
  if (kind === 'reset') {
    return t('organizations.resetPassword')
  }
  return kind === 'deactivate' ? t('organizations.deactivate') : t('organizations.reactivate')
})

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
    if (p.kind === 'reset') {
      const password = await resetPassword(p.member.id)
      confirmOpen.value = false
      showPassword(p.member.login, password)
    } else {
      await setActive(p.org.id, p.kind === 'reactivate')
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
  <div class="mx-auto max-w-7xl p-6">
    <div class="app-card">
      <div class="app-card-header flex items-center justify-between">
        <span>{{ $t('organizations.title') }}</span>
        <v-btn
          variant="flat"
          color="primary"
          size="small"
          prepend-icon="mdi-plus"
          data-test="organization-add"
          @click="openCreate"
        >
          {{ $t('organizations.add') }}
        </v-btn>
      </div>

      <div v-if="error" class="flex items-center gap-3 p-4">
        <span class="text-error" data-test="organizations-error">{{ errorMessage(error) }}</span>
        <v-btn size="small" variant="text" @click="reload">{{ $t('common.retry') }}</v-btn>
      </div>

      <v-data-table
        v-else
        :headers="headers"
        :items="organizations"
        :loading="loading"
        :no-data-text="$t('organizations.empty')"
        item-value="id"
        density="compact"
        :items-per-page="-1"
        hide-default-footer
        data-test="organizations-table"
      >
        <template #[`item.name`]="{ item }">
          <span class="text-strong" data-test="organization-row-name">{{ item.name }}</span>
        </template>

        <template #[`item.status`]="{ item }">
          <v-chip
            size="small"
            variant="tonal"
            :color="item.deactivated_at ? 'warning' : 'success'"
            data-test="organization-status"
          >
            {{ item.deactivated_at ? $t('organizations.deactivated') : $t('organizations.active') }}
          </v-chip>
        </template>

        <template #[`item.login`]="{ item }">
          <span class="font-mono">{{ firstUser(item)?.login }}</span>
        </template>

        <template #[`item.profile`]="{ item }">
          <span v-if="profileText(firstUser(item))" class="text-body">
            {{ profileText(firstUser(item)) }}
          </span>
          <span v-else class="text-faint" data-test="profile-missing">
            {{ $t('organizations.profileMissing') }}
          </span>
        </template>

        <template #[`item.last_login`]="{ item }">
          <span class="text-muted">{{ time(firstUser(item)?.last_login_at ?? null) }}</span>
        </template>

        <template #[`item.created_at`]="{ item }">
          <span class="text-muted">{{ time(item.created_at) }}</span>
        </template>

        <template #[`item.actions`]="{ item }">
          <v-menu location="bottom end">
            <template #activator="{ props: menu }">
              <v-btn
                v-bind="menu"
                icon="mdi-dots-vertical"
                size="small"
                variant="text"
                :data-test="`organization-actions-${item.id}`"
              />
            </template>
            <v-list density="compact">
              <v-list-item
                v-if="firstUser(item)"
                prepend-icon="mdi-lock-reset"
                :title="$t('organizations.resetPassword')"
                data-test="action-reset"
                @click="ask({ kind: 'reset', org: item, member: firstUser(item)! })"
              />
              <v-list-item
                v-if="item.deactivated_at"
                prepend-icon="mdi-play-circle-outline"
                :title="$t('organizations.reactivate')"
                data-test="action-reactivate"
                @click="ask({ kind: 'reactivate', org: item })"
              />
              <v-list-item
                v-else
                prepend-icon="mdi-pause-circle-outline"
                :title="$t('organizations.deactivate')"
                data-test="action-deactivate"
                @click="ask({ kind: 'deactivate', org: item })"
              />
            </v-list>
          </v-menu>
        </template>
      </v-data-table>
    </div>

    <OrganizationCreateDialog
      v-model="createOpen"
      :loading="creating"
      :error="createError"
      @submit="submitCreate"
    />

    <ConfirmDialog
      v-model="confirmOpen"
      :title="confirmTitle"
      :text="confirmText"
      :confirm-text="$t('common.confirm')"
      :loading="confirming"
      :error="confirmError"
      @confirm="confirm"
    />

    <TemporaryPasswordDialog v-model="shownOpen" :login="shownLogin" :password="shownPassword" />
  </div>
</template>
