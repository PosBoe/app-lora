<script setup lang="ts">
import { formatDateTime } from '@/lib/i18n'

import { displayDevEUI } from './use-devices'
import { daysLeft, type Unmatched } from './use-uplinks'

// UnmatchedWarning tells which DevEUIs send uplinks that no device of the
// organization takes, and offers to add each one (device:G9).
defineProps<{ unmatched: Unmatched[] }>()

const emit = defineEmits<{ add: [devEui: string] }>()

const { language } = useLanguage()
const open = shallowRef(false)
</script>

<template>
  <div
    v-if="unmatched.length"
    class="rounded-lg border border-subtle bg-surface p-4"
    data-test="unmatched-warning"
  >
    <div class="flex flex-wrap items-center justify-between gap-3">
      <span class="text-warning" data-test="unmatched-summary">
        <v-icon icon="mdi-alert-outline" size="small" class="mr-1" />
        {{ $t('unmatched.summary', { n: unmatched.length }) }}
      </span>
      <v-btn size="small" variant="text" data-test="unmatched-toggle" @click="open = !open">
        {{ open ? $t('unmatched.hide') : $t('unmatched.show') }}
      </v-btn>
    </div>
    <p class="mt-1 text-sm text-muted">{{ $t('unmatched.explain') }}</p>

    <table v-if="open" class="mt-3 w-full text-sm" data-test="unmatched-list">
      <thead class="text-left text-muted">
        <tr>
          <th class="py-1 font-normal">DevEUI</th>
          <th class="py-1 font-normal">{{ $t('unmatched.count') }}</th>
          <th class="py-1 font-normal">{{ $t('unmatched.last') }}</th>
          <th class="py-1 font-normal">{{ $t('unmatched.integration') }}</th>
          <th class="py-1 font-normal">{{ $t('unmatched.left') }}</th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="u in unmatched" :key="u.dev_eui" class="border-t border-subtle text-body">
          <td class="py-1 font-mono">{{ displayDevEUI(u.dev_eui) }}</td>
          <td class="py-1">{{ u.count }}</td>
          <td class="py-1">{{ formatDateTime(u.last_received_at, language) }}</td>
          <td class="py-1">{{ u.integration_name ?? '—' }}</td>
          <td class="py-1">{{ $t('unmatched.days', { n: daysLeft(u.deletes_at) }) }}</td>
          <td class="py-1 text-right">
            <v-btn
              size="x-small"
              variant="tonal"
              color="primary"
              :data-test="`unmatched-add-${u.dev_eui}`"
              @click="emit('add', u.dev_eui)"
            >
              {{ $t('devices.add') }}
            </v-btn>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
