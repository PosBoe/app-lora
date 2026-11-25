<script setup lang="ts">
// OrganizationCreateDialog asks for the name of a new organization and the
// login of its user. The parent runs the create on `submit`.
const props = defineProps<{
  loading?: boolean
  error?: string
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ submit: [name: string, login: string] }>()

const name = shallowRef('')
const login = shallowRef('')

watch(open, (isOpen) => {
  if (isOpen) {
    name.value = ''
    login.value = ''
  }
})

function submit() {
  if (!props.loading) {
    emit('submit', name.value, login.value)
  }
}
</script>

<template>
  <v-dialog v-model="open" max-width="440">
    <form class="app-card" data-test="organization-create-dialog" @submit.prevent="submit">
      <div class="app-card-header">{{ $t('organizations.createTitle') }}</div>
      <div class="flex flex-col gap-4 p-4">
        <v-text-field
          v-model="name"
          :label="$t('organizations.name')"
          autofocus
          density="compact"
          variant="outlined"
          hide-details
          data-test="organization-name-input"
        />
        <v-text-field
          v-model="login"
          :label="$t('organizations.login')"
          :hint="$t('organizations.loginHint')"
          persistent-hint
          autocomplete="off"
          density="compact"
          variant="outlined"
          data-test="organization-login-input"
        />
        <p v-if="error" class="text-error" data-test="organization-create-error">{{ error }}</p>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="open = false">{{ $t('common.cancel') }}</v-btn>
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          :loading="loading"
          :disabled="!name.trim() || !login.trim()"
          data-test="organization-create-submit"
        >
          {{ $t('organizations.createSubmit') }}
        </v-btn>
      </div>
    </form>
  </v-dialog>
</template>
