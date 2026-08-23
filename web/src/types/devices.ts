export type DeviceStatus = 'registered' | 'online' | 'offline' | 'maintenance' | 'replaced' | 'retired'

export interface DeviceRow {
  id: string
  code: string
  model: string
  point: string
  status: DeviceStatus
  lastSeenAt: string
}
