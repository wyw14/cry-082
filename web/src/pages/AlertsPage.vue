<script setup lang="ts">
import { ref } from 'vue'
import type { AlertRow } from '../types/incidents'
const rows = ref<AlertRow[]>([{ id: 'alert-demo', kind: 'environmental-exceedance', status: 'acknowledged', point: '东门监测点', startedAt: '2026-08-23 15:32', assignee: '值班主管' }, { id: 'offline-demo', kind: 'device-offline', status: 'open', point: '材料堆场', startedAt: '2026-08-23 15:41', assignee: '-' }])
</script>
<template><section><h1 class="page-title">告警处置</h1><p class="page-subtitle">环境超标与设备离线分别流转</p><div class="panel"><el-table :data="rows"><el-table-column prop="id" label="编号" /><el-table-column prop="kind" label="类型" /><el-table-column prop="point" label="监测点" /><el-table-column prop="status" label="状态" /><el-table-column prop="assignee" label="负责人" /></el-table></div></section></template>
