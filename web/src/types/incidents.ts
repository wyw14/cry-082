export type AlertKind = 'environmental-exceedance' | 'device-offline' | 'device-drift'
export type AlertStatus = 'open' | 'acknowledged' | 'dispatched' | 'recovering' | 'recovered' | 'closed'

export interface AlertRow {
  id: string
  kind: AlertKind
  status: AlertStatus
  point: string
  startedAt: string
  assignee: string
}
