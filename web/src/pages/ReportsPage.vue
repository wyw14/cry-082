<script setup lang="ts">
import { ref } from 'vue'
const format = ref<'csv' | 'json'>('csv')
</script>
<template><section><h1 class="page-title">日报与监管导出</h1><p class="page-subtitle">按配置时区汇总，导出文件通过授权接口获取</p><div class="panel"><el-form label-position="top"><el-form-item label="导出格式"><el-segmented v-model="format" :options="['csv','json']" /></el-form-item><el-button type="primary">生成监管导出</el-button></el-form></div></section></template>
