<script setup lang="ts">
// ConfirmDialog asks the user to confirm one action. The parent opens it with
// v-model, runs the action on `confirm`, and closes it when the action ends.
defineProps<{
  title: string
  text: string
  confirmText: string
  loading?: boolean
  error?: string
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ confirm: [] }>()
</script>

<template>
  <v-dialog v-model="open" max-width="440">
    <div class="app-card" data-test="confirm-dialog">
      <div class="app-card-header">{{ title }}</div>
      <div class="p-4 text-body">
        <p>{{ text }}</p>
        <p v-if="error" class="mt-3 text-error" data-test="confirm-error">{{ error }}</p>
      </div>
      <div class="app-card-actions pt-0">
        <v-btn variant="text" @click="open = false">{{ $t('common.cancel') }}</v-btn>
        <v-btn
          variant="flat"
          color="primary"
          :loading="loading"
          data-test="confirm-submit"
          @click="emit('confirm')"
        >
          {{ confirmText }}
        </v-btn>
      </div>
    </div>
  </v-dialog>
</template>
