import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getDashboard } from '../services/monitoring'
import type { MonitoringDashboard } from '../types/telemetry'

export const useMonitoringStore = defineStore('monitoring', () => {
  const dashboard = ref<MonitoringDashboard | null>(null)
  const loading = ref(false)
  const error = ref('')
  const environmentalCount = computed(() => dashboard.value?.OpenEnvironmentalAlerts ?? 0)
  const offlineCount = computed(() => dashboard.value?.OpenOfflineAlerts ?? 0)
  async function refresh(siteId = 'site-demo') {
    loading.value = true
    error.value = ''
    try {
      dashboard.value = await getDashboard(siteId)
    } catch (reason) {
      error.value = reason instanceof Error ? reason.message : '监控数据加载失败'
    } finally {
      loading.value = false
    }
  }
  return { dashboard, loading, error, environmentalCount, offlineCount, refresh }
})
