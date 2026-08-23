import { request } from './api'
import type { EnvironmentalMetric, MonitoringDashboard, TrendSample } from '../types/telemetry'

export const getDashboard = (siteId: string) =>
  request<MonitoringDashboard>(`/api/v1/sites/${siteId}/dashboard`)

export const getTrend = (
  siteId: string,
  pointId: string,
  metric: EnvironmentalMetric,
  start: string,
  end: string,
) => {
  const query = new URLSearchParams({
    point_id: pointId,
    metric,
    start,
    end,
    bucket_seconds: '300',
  })
  return request<TrendSample[]>(`/api/v1/sites/${siteId}/trends?${query.toString()}`)
}
