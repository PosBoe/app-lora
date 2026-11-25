<script setup lang="ts">
import { LANGUAGES } from '@/lib/i18n'

const { language, setLanguage } = useLanguage()
</script>

<template>
  <v-btn-toggle
    :model-value="language"
    mandatory
    density="compact"
    variant="text"
    divided
    class="border border-subtle"
    data-test="language-switch"
    @update:model-value="setLanguage"
  >
    <v-btn
      v-for="lang in LANGUAGES"
      :key="lang"
      :value="lang"
      size="small"
      :data-test="`language-${lang}`"
    >
      {{ lang.toUpperCase() }}
    </v-btn>
  </v-btn-toggle>
</template>
