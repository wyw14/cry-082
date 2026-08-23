import { computed, type Ref } from 'vue'

export function useTimezone(value: Ref<string | Date>, timezone: Ref<string>) {
  return computed(() => new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'medium', timeZone: timezone.value }).format(new Date(value.value)))
}
