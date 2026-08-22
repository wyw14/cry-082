export type ObservationQuality = 'accepted' | 'suspect' | 'quarantined'

export type EnvironmentalMetric =
  | 'pm2_5'
  | 'pm10'
  | 'noise'
  | 'temperature'
  | 'humidity'
  | 'wind_speed'

export interface LatestMetric {
  PointID: string
  Metric: EnvironmentalMetric
  Value: number
  Unit: string
  SampledAt: string
  Quality: ObservationQuality
}

export interface MonitoringDashboard {
  SiteID: string
  GeneratedAt: string
  Latest: LatestMetric[]
  OpenEnvironmentalAlerts: number
  OpenOfflineAlerts: number
}

export interface TrendSample {
  bucketStart: string
  minimum: number
  maximum: number
  average: number
  acceptedSamples: number
  suspectSamples: number
}
