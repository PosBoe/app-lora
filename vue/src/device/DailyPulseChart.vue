<script setup lang="ts">
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  LinearScale,
  Tooltip,
  type ChartData,
  type ChartOptions,
} from 'chart.js'
import { Bar } from 'vue-chartjs'

import type { Language } from '@/lib/i18n'

import { useChartTokens } from './use-readings'

ChartJS.register(BarElement, CategoryScale, LinearScale, Tooltip)

// DailyPulseChart draws the pulses of each day as columns: one series, so
// the card title names it and there is no legend. A day with no delta (the
// first day of a device) has no column.
const props = defineProps<{
  days: { day: string; delta: number | null }[]
  language: Language
}>()

const tokens = useChartTokens()
const locale = computed(() => (props.language === 'ru' ? 'ru-MD' : 'ro-MD'))

function dayLabel(day: string): string {
  const [y, m, d] = day.split('-').map(Number)
  return new Intl.DateTimeFormat(locale.value, {
    day: '2-digit',
    month: '2-digit',
    timeZone: 'UTC',
  }).format(new Date(Date.UTC(y!, m! - 1, d!)))
}

const data = computed<ChartData<'bar', (number | null)[]>>(() => ({
  labels: props.days.map((d) => dayLabel(d.day)),
  datasets: [
    {
      data: props.days.map((d) => d.delta),
      backgroundColor: tokens.value.primary,
      hoverBackgroundColor: tokens.value.primary,
      maxBarThickness: 24,
      borderRadius: { topLeft: 4, topRight: 4 },
      borderSkipped: 'bottom',
      // The surface-colored border is the 2px gap between touching columns.
      borderWidth: { left: 1, right: 1 },
      borderColor: tokens.value.surface,
    },
  ],
}))

const options = computed<ChartOptions<'bar'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: false,
      callbacks: {
        label: (item) => new Intl.NumberFormat(locale.value).format(item.parsed.y as number),
      },
    },
  },
  scales: {
    x: {
      grid: { display: false },
      border: { color: tokens.value.subtle },
      ticks: { color: tokens.value.muted, maxTicksLimit: 10, autoSkip: true },
    },
    y: {
      beginAtZero: true,
      grid: { color: tokens.value.subtle },
      border: { display: false },
      ticks: {
        color: tokens.value.muted,
        maxTicksLimit: 5,
        callback: (v) => new Intl.NumberFormat(locale.value).format(v as number),
      },
    },
  },
}))
</script>

<template>
  <div class="h-56" data-test="daily-pulse-chart">
    <Bar :data="data" :options="options" />
  </div>
</template>
