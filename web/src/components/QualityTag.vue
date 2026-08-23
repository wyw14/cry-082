<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ quality: 'accepted' | 'suspect' | 'quarantined' }>()
const type = computed(() => props.quality === 'accepted' ? 'success' : props.quality === 'suspect' ? 'warning' : 'danger')
const label = computed(() => ({ accepted: '可信', suspect: '待复核', quarantined: '已隔离' })[props.quality])
</script>
<template><el-tag :type="type" effect="plain" size="small">{{ label }}</el-tag></template>
