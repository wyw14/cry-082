<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useMonitoringStore } from '../stores/monitoring'
import MetricStrip from '../features/monitoring/MetricStrip.vue'
import type { LatestMetric } from '../types/telemetry'
const store = useMonitoringStore()
const sampleMetrics: LatestMetric[] = [
  { PointID: 'point-east-gate', Metric: 'pm2_5', Value: 34, Unit: 'ug/m3', SampledAt: new Date().toISOString(), Quality: 'accepted' },
  { PointID: 'point-east-gate', Metric: 'pm10', Value: 82, Unit: 'ug/m3', SampledAt: new Date().toISOString(), Quality: 'accepted' },
  { PointID: 'point-east-gate', Metric: 'noise', Value: 67, Unit: 'dB', SampledAt: new Date().toISOString(), Quality: 'suspect' },
  { PointID: 'point-east-gate', Metric: 'wind_speed', Value: 2.8, Unit: 'm/s', SampledAt: new Date().toISOString(), Quality: 'accepted' },
]
const metrics = computed<LatestMetric[]>(() => store.dashboard?.Latest ?? sampleMetrics)
onMounted(() => store.refresh())
</script>
<template><section><h1 class="page-title">实时监控</h1><p class="page-subtitle">观测质量、区域趋势与处置状态</p><el-alert v-if="store.error" :title="store.error" type="warning" show-icon /><MetricStrip :metrics="metrics" /><div class="content-grid"><div class="panel"><h3>区域指标对比</h3><div class="bar-row"><span>东门施工区</span><div class="bar-track"><div class="bar-fill" style="width:62%" /></div><strong>82</strong></div><div class="bar-row"><span>材料堆场</span><div class="bar-track"><div class="bar-fill" style="width:38%" /></div><strong>51</strong></div><div class="bar-row"><span>生活区</span><div class="bar-track"><div class="bar-fill" style="width:21%" /></div><strong>29</strong></div></div><div class="panel"><h3>待处置</h3><el-statistic title="环境超标告警" :value="store.environmentalCount" /><el-divider /><el-statistic title="设备离线告警" :value="store.offlineCount" /></div></div></section></template>
