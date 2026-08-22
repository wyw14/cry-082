<script setup lang="ts">
import { ref } from 'vue'
import type { DeviceRow } from '../types/devices'
const rows = ref<DeviceRow[]>([{ id: 'device-demo-001', code: 'DUST-EAST-001', model: 'LocalSim-X1', point: '东门监测点', status: 'online', lastSeenAt: '刚刚' }])
</script>
<template><section><h1 class="page-title">设备治理</h1><p class="page-subtitle">生命周期、网络参数、校准与维护记录</p><div class="panel"><el-table :data="rows"><el-table-column prop="code" label="设备编号" /><el-table-column prop="model" label="型号" /><el-table-column prop="point" label="安装位置" /><el-table-column prop="status" label="状态" /><el-table-column prop="lastSeenAt" label="最近上报" /></el-table></div></section></template>
