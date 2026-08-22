<script setup lang="ts">
import type { LatestMetric } from '../../types/telemetry'
import QualityTag from '../../components/QualityTag.vue'
defineProps<{ metrics: LatestMetric[] }>()
const names: Record<string, string> = { pm2_5: 'PM2.5', pm10: 'PM10', noise: '噪声', temperature: '温度', humidity: '湿度', wind_speed: '风速' }
</script>
<template><div class="metric-grid"><div v-for="item in metrics" :key="`${item.PointID}:${item.Metric}`" class="metric"><div class="metric-label">{{ names[item.Metric] ?? item.Metric }} · {{ item.PointID }}</div><div class="metric-value">{{ item.Value }}<small>{{ item.Unit }}</small></div><QualityTag :quality="item.Quality" /></div></div></template>
