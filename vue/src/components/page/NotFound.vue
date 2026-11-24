<script setup lang="ts">
defineOptions({ name: 'PageNotFound' })
</script>

<template>
  <div class="mx-auto max-w-3xl p-6">
    <div class="app-card p-6 text-center">
      <div class="text-xl font-medium text-strong">Page not found</div>
      <p class="mt-2 text-muted">The page you opened does not exist.</p>
      <router-link to="/" class="app-link mt-4 inline-block">Go to the home page</router-link>
    </div>
  </div>
</template>
