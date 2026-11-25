<script setup lang="ts">
// TemporaryPasswordDialog shows a login and its temporary password one time,
// so the admin can copy them and send them to the user.
defineProps<{
  login: string
  password: string
}>()

const open = defineModel<boolean>({ required: true })

const { copy, copied } = useClipboard({ legacy: true })
</script>

<template>
  <v-dialog v-model="open" max-width="440" persistent>
    <div class="app-card" data-test="temporary-password-dialog">
      <div class="app-card-header">{{ $t('temporaryPassword.title') }}</div>
      <div class="flex flex-col gap-3 p-4 text-body">
        <p class="text-muted">{{ $t('temporaryPassword.intro') }}</p>
        <dl class="grid grid-cols-[6rem_1fr] items-center gap-y-2">
          <dt class="text-muted">{{ $t('temporaryPassword.login') }}</dt>
          <dd class="font-mono" data-test="temporary-login">{{ login }}</dd>
          <dt class="text-muted">{{ $t('temporaryPassword.password') }}</dt>
          <dd class="flex items-center gap-2">
            <span class="font-mono text-strong" data-test="temporary-password">{{ password }}</span>
            <v-btn
              size="small"
              variant="text"
              :prepend-icon="copied ? 'mdi-check' : 'mdi-content-copy'"
              data-test="copy-password"
              @click="copy(`${login}\n${password}`)"
            >
              {{ copied ? $t('common.copied') : $t('common.copy') }}
            </v-btn>
          </dd>
        </dl>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="flat" color="primary" data-test="temporary-close" @click="open = false">
          {{ $t('common.close') }}
        </v-btn>
      </div>
    </div>
  </v-dialog>
</template>
