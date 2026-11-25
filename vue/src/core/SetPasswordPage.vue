<script setup lang="ts">
const router = useRouter()
const session = useSession()
const errorMessage = useErrorMessage()
const { t } = useI18n()

const current = shallowRef('')
const next = shallowRef('')
const repeat = shallowRef('')
const submitting = shallowRef(false)
const error = shallowRef('')

const mismatch = computed(() => repeat.value !== '' && next.value !== repeat.value)

async function submit() {
  if (next.value !== repeat.value) {
    error.value = t('setPassword.mismatch')
    return
  }
  submitting.value = true
  error.value = ''
  try {
    await session.changePassword(current.value, next.value)
    await router.push('/')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-sm p-6 pt-16">
    <form class="app-card" data-test="set-password-form" @submit.prevent="submit">
      <div class="app-card-header">{{ $t('setPassword.title') }}</div>

      <div class="flex flex-col gap-4 p-4">
        <p class="text-muted">{{ $t('setPassword.intro') }}</p>
        <v-text-field
          v-model="current"
          :label="$t('setPassword.current')"
          type="password"
          autocomplete="current-password"
          density="compact"
          variant="outlined"
          hide-details
          data-test="current-input"
        />
        <v-text-field
          v-model="next"
          :label="$t('setPassword.next')"
          :hint="$t('setPassword.rule')"
          persistent-hint
          type="password"
          autocomplete="new-password"
          density="compact"
          variant="outlined"
          data-test="next-input"
        />
        <v-text-field
          v-model="repeat"
          :label="$t('setPassword.repeat')"
          :error-messages="mismatch ? $t('setPassword.mismatch') : undefined"
          type="password"
          autocomplete="new-password"
          density="compact"
          variant="outlined"
          data-test="repeat-input"
        />
        <p v-if="error" class="text-error" data-test="set-password-error">{{ error }}</p>
      </div>

      <div class="app-card-actions pt-0">
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          block
          :loading="submitting"
          :disabled="!current || !next || !repeat || mismatch"
          data-test="set-password-submit"
        >
          {{ $t('setPassword.submit') }}
        </v-btn>
      </div>
    </form>
  </div>
</template>
