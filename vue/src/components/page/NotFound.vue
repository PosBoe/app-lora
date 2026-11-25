<script setup lang="ts">
defineOptions({ name: 'PageNotFound' })
</script>

<template>
  <div class="mx-auto max-w-3xl p-6">
    <div class="app-card p-6 text-center">
      <div class="text-xl font-medium text-strong">{{ $t('notFound.title') }}</div>
      <p class="mt-2 text-muted">{{ $t('notFound.text') }}</p>
      <router-link to="/" class="app-link mt-4 inline-block">{{ $t('notFound.home') }}</router-link>
    </div>
  </div>
</template>
