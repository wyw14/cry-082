import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMonitoringStore } from '../stores/monitoring'

vi.mock('../services/monitoring', () => ({ getDashboard: vi.fn(async () => ({ SiteID: 's1', GeneratedAt: '2026-08-23T08:00:00Z', Latest: [], OpenEnvironmentalAlerts: 2, OpenOfflineAlerts: 1 })) }))

describe('monitoring store', () => { beforeEach(() => setActivePinia(createPinia())); it('keeps environmental and offline alert totals separate', async () => { const store = useMonitoringStore(); await store.refresh('s1'); expect(store.environmentalCount).toBe(2); expect(store.offlineCount).toBe(1) }) })
