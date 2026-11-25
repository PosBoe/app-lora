<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const session = useSession()
const errorMessage = useErrorMessage()

const login = shallowRef('')
const password = shallowRef('')
const submitting = shallowRef(false)
const error = shallowRef('')

// safeRedirect keeps a redirect inside the app: a path that starts with one
// slash.
function safeRedirect(value: unknown): string {
  if (typeof value === 'string' && value.startsWith('/') && !value.startsWith('//')) {
    return value
  }
  return '/'
}

async function submit() {
  submitting.value = true
  error.value = ''
  try {
    await session.login(login.value, password.value)
    await router.push(safeRedirect(route.query.redirect))
  } catch (err) {
    error.value = errorMessage(err)
    password.value = ''
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-sm p-6 pt-16">
    <form class="app-card" data-test="login-form" @submit.prevent="submit">
      <div class="app-card-header">{{ $t('login.title') }}</div>

      <div class="flex flex-col gap-4 p-4">
        <v-text-field
          v-model="login"
          :label="$t('login.login')"
          autocomplete="username"
          autofocus
          density="compact"
          variant="outlined"
          hide-details
          data-test="login-input"
        />
        <v-text-field
          v-model="password"
          :label="$t('login.password')"
          type="password"
          autocomplete="current-password"
          density="compact"
          variant="outlined"
          hide-details
          data-test="password-input"
        />
        <p v-if="error" class="text-error" data-test="login-error">{{ error }}</p>
      </div>

      <div class="app-card-actions pt-0">
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          block
          :loading="submitting"
          :disabled="!login || !password"
          data-test="login-submit"
        >
          {{ $t('login.submit') }}
        </v-btn>
      </div>
    </form>
  </div>
</template>
