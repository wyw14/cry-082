<script setup lang="ts">
import { Bell, DataAnalysis, Monitor, Setting } from '@element-plus/icons-vue'
</script>

<template>
  <el-container class="shell">
    <el-aside width="216px" class="sidebar">
      <div class="brand"><span class="brand-mark">尘</span><strong>工地环境监管</strong></div>
      <el-menu router :default-active="$route.path" background-color="#17211d" text-color="#cad4cf" active-text-color="#ffffff">
        <el-menu-item index="/"><el-icon><Monitor /></el-icon><span>实时监控</span></el-menu-item>
        <el-menu-item index="/alerts"><el-icon><Bell /></el-icon><span>告警处置</span></el-menu-item>
        <el-menu-item index="/reports"><el-icon><DataAnalysis /></el-icon><span>日报与导出</span></el-menu-item>
        <el-menu-item index="/devices"><el-icon><Setting /></el-icon><span>设备治理</span></el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar"><div><strong>滨江综合施工现场</strong><span> Asia/Shanghai</span></div><el-tag type="success" effect="plain">在线</el-tag></el-header>
      <el-main><router-view /></el-main>
    </el-container>
  </el-container>
</template>
