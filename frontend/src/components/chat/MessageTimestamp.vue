<template>
  <time v-if="label" class="conversation-time" :datetime="datetime">{{ label }}</time>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getConversationTimestampModel } from '@/utils/messageTimestamp'

const props = defineProps<{
  value?: unknown
}>()


const model = computed(() => getConversationTimestampModel(props.value))
const datetime = computed(() => model.value?.datetime ?? '')
const label = computed(() => {
  const next = model.value
  if (!next) return ''
  if (next.kind === 'today') return `Today ${next.time}`
  if (next.kind === 'yesterday') return `Yesterday ${next.time}`
  if (next.kind === 'thisYear') {
    return `${next.month}/${next.day} ${next.time}`
  }
  return `${next.month}/${next.day}/${next.year} ${next.time}`
})
</script>

<style scoped lang="less">
.conversation-time {
  display: block;
  width: 100%;
  padding: 4px 0 8px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-variant-numeric: tabular-nums;
  line-height: 20px;
  text-align: center;
  user-select: none;
}
</style>
