<script setup lang="ts">
import {
  Chart as ChartJS,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
  Filler,
  type ChartData,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'vue-chartjs'

import { formatDateTime, type Language } from '@/lib/i18n'

import { useChartTokens } from './use-readings'

ChartJS.register(LinearScale, LineElement, PointElement, Tooltip, Filler)

// PulseLineChart draws the reported pulse count of a device over time: one
// series, so the card title names it and there is no legend.
const props = defineProps<{
  points: { at: string; pulse_count: number }[]
  language: Language
}>()

const tokens = useChartTokens()
const numbers = computed(() => new Intl.NumberFormat(props.language === 'ru' ? 'ru-MD' : 'ro-MD'))

const data = computed<ChartData<'line', { x: number; y: number }[]>>(() => ({
  datasets: [
    {
      data: props.points.map((p) => ({ x: new Date(p.at).getTime(), y: p.pulse_count })),
      borderColor: tokens.value.primary,
      backgroundColor: `${tokens.value.primary}1a`,
      borderWidth: 2,
      borderCapStyle: 'round',
      borderJoinStyle: 'round',
      fill: true,
      pointRadius: 0,
      pointHoverRadius: 5,
      pointHoverBorderWidth: 2,
      pointHoverBorderColor: tokens.value.surface,
      pointHoverBackgroundColor: tokens.value.primary,
      tension: 0,
    },
  ],
}))

const options = computed<ChartOptions<'line'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { mode: 'nearest', axis: 'x', intersect: false },
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: false,
      callbacks: {
        title: (items) =>
          formatDateTime(new Date(items[0]!.parsed.x as number).toISOString(), props.language),
        label: (item) => numbers.value.format(item.parsed.y as number),
      },
    },
  },
  scales: {
    x: {
      type: 'linear',
      grid: { display: false },
      border: { color: tokens.value.subtle },
      ticks: {
        color: tokens.value.muted,
        maxTicksLimit: 6,
        callback: (v) =>
          new Intl.DateTimeFormat(props.language === 'ru' ? 'ru-MD' : 'ro-MD', {
            timeZone: 'Europe/Chisinau',
            day: '2-digit',
            month: '2-digit',
          }).format(new Date(v as number)),
      },
    },
    y: {
      grid: { color: tokens.value.subtle },
      border: { display: false },
      ticks: {
        color: tokens.value.muted,
        maxTicksLimit: 5,
        callback: (v) => numbers.value.format(v as number),
      },
    },
  },
}))
</script>

<template>
  <div class="h-56" data-test="pulse-line-chart">
    <Line :data="data" :options="options" />
  </div>
</template>
