<script setup lang="ts">
// DeviceModelDialog asks for the vendor and the model name of a device model,
// to create one or to rename one. The parent runs the action on `submit`.
const props = defineProps<{
  title: string
  initial?: { vendor: string; model: string }
  loading?: boolean
  error?: string
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ submit: [vendor: string, model: string] }>()

const vendor = shallowRef('')
const model = shallowRef('')

watch(
  open,
  (isOpen) => {
    if (isOpen) {
      vendor.value = props.initial?.vendor ?? ''
      model.value = props.initial?.model ?? ''
    }
  },
  { immediate: true },
)

function submit() {
  if (!props.loading) {
    emit('submit', vendor.value, model.value)
  }
}
</script>

<template>
  <v-dialog v-model="open" max-width="420">
    <form class="app-card" data-test="model-dialog" @submit.prevent="submit">
      <div class="app-card-header">{{ title }}</div>
      <div class="flex flex-col gap-4 p-4">
        <v-text-field
          v-model="vendor"
          :label="$t('models.vendor')"
          :placeholder="$t('models.vendorExample')"
          autofocus
          density="compact"
          variant="outlined"
          hide-details
          data-test="model-vendor-input"
        />
        <v-text-field
          v-model="model"
          :label="$t('models.model')"
          :placeholder="$t('models.modelExample')"
          density="compact"
          variant="outlined"
          hide-details
          data-test="model-name-input"
        />
        <p v-if="error" class="text-error" data-test="model-error">{{ error }}</p>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="open = false">{{ $t('common.cancel') }}</v-btn>
        <v-btn
          type="submit"
          variant="flat"
          color="primary"
          :loading="loading"
          :disabled="!vendor.trim() || !model.trim()"
          data-test="model-submit"
        >
          {{ $t('common.save') }}
        </v-btn>
      </div>
    </form>
  </v-dialog>
</template>
