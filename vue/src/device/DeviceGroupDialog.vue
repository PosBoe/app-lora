<script setup lang="ts">
// DeviceGroupDialog asks for the name of a device group, to create one or to
// rename one. The parent runs the action on `submit`.
const props = defineProps<{
  title: string
  initial?: string
  loading?: boolean
  error?: string
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ submit: [name: string] }>()

const name = shallowRef('')

watch(
  open,
  (isOpen) => {
    if (isOpen) {
      name.value = props.initial ?? ''
    }
  },
  { immediate: true },
)

function submit() {
  if (!props.loading) {
    emit('submit', name.value)
  }
}
</script>

<template>
  <v-dialog v-model="open" max-width="420">
    <form class="app-card" data-test="group-dialog" @submit.prevent="submit">
      <div class="app-card-header">{{ title }}</div>
      <div class="flex flex-col gap-4 p-4">
        <v-text-field
          v-model="name"
          :label="$t('groups.name')"
          :placeholder="$t('groups.nameExample')"
          autofocus
          density="compact"
          variant="outlined"
          hide-details
          data-test="group-name-input"
        />
        <p v-if="error" class="text-error" data-test="group-error">{{ error }}</p>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="open = false">{{ $t('common.cancel') }}</v-btn>
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          :loading="loading"
          :disabled="!name.trim()"
          data-test="group-submit"
        >
          {{ $t('common.save') }}
        </v-btn>
      </div>
    </form>
  </v-dialog>
</template>
