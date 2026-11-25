<script setup lang="ts">
// HomePage is the first page of an organization user. It holds an empty
// state until the device context adds the device list.
const { me } = useSession()
</script>

<template>
  <div class="mx-auto max-w-5xl p-6">
    <div class="app-card">
      <div class="app-card-header">
        {{ $t('home.title') }}
        <span v-if="me?.organization" class="ml-2 text-muted" data-test="organization-name">
          · {{ me.organization.name }}
        </span>
      </div>
      <p class="p-6 text-center text-muted">{{ $t('home.empty') }}</p>
    </div>
  </div>
</template>
