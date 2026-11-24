<script setup lang="ts">
const { health, loading, error, reload } = useHealth()
</script>

<template>
  <div class="mx-auto max-w-3xl p-6">
    <div class="app-card">
      <div class="app-card-header">System health</div>

      <div class="p-4">
        <v-progress-linear v-if="loading && !health" indeterminate color="primary" />

        <dl v-else-if="health" class="grid grid-cols-[8rem_1fr] gap-y-2 text-body">
          <dt class="text-muted">Status</dt>
          <dd data-test="status" :class="health.status === 'ok' ? 'text-success' : 'text-warning'">
            {{ health.status }}
          </dd>
          <dt class="text-muted">Database</dt>
          <dd data-test="db" :class="health.db === 'ok' ? 'text-success' : 'text-error'">
            {{ health.db }}
          </dd>
          <dt class="text-muted">Version</dt>
          <dd>{{ health.version }}</dd>
        </dl>

        <p v-else data-test="error" class="text-error">{{ error }}</p>
      </div>

      <div class="app-card-actions pt-0">
        <v-btn variant="flat" color="primary" :loading="loading" @click="reload">Refresh</v-btn>
      </div>
    </div>
  </div>
</template>
