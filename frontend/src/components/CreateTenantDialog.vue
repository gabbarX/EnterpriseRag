<template>
  <t-dialog :visible="visible" width="480px" :on-confirm="handleSubmit" :on-close="handleClose"
    :confirm-btn="{ content: 'Create', loading: submitting, theme: 'primary' }"
    :cancel-btn="{ content: 'Cancel' }" :close-on-overlay-click="!submitting"
    :close-on-esc-keydown="!submitting" @update:visible="onVisibleUpdate">
    <template #header>
      <span class="create-tenant-dialog-header">
        <t-icon name="system-sum" size="20px" class="create-tenant-dialog-header-icon" aria-hidden="true" />
        <span class="create-tenant-dialog-header-title">{{ 'Create new workspace' }}</span>
      </span>
    </template>

    <p class="create-tenant-tip">{{ 'A workspace has its own knowledge bases and members. You will become the owner of the new workspace.' }}</p>

    <t-form ref="formRef" :data="form" :rules="formRules" label-align="top" class="create-tenant-form" @submit.prevent>
      <t-form-item :label="'Workspace name'" name="name">
        <t-input v-model="form.name" :placeholder="'e.g. My new project'" :maxlength="128" autofocus
          @enter="handleSubmit" />
      </t-form-item>
      <t-form-item :label="'Description (optional)'" name="description">
        <t-textarea v-model="form.description" :placeholder="'Briefly describe what this workspace is for'"
          :maxlength="512" :autosize="{ minRows: 3, maxRows: 5 }" />
      </t-form-item>
    </t-form>
  </t-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { MessagePlugin, type FormInstanceFunctions, type FormRule } from 'tdesign-vue-next'
import { createTenant, type TenantInfo } from '@/api/tenant'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'created', tenant: TenantInfo): void
}>()


const formRef = ref<FormInstanceFunctions | null>(null)
const submitting = ref(false)

const form = reactive({
  name: '',
  description: '',
})

const formRules: Record<string, FormRule[]> = {
  name: [
    {
      validator: (val: string) => (val ?? '').trim().length > 0,
      message: 'Please enter a workspace name',
      trigger: 'blur',
    },
  ],
}

watch(
  () => props.visible,
  (open) => {
    if (open) {
      form.name = ''
      form.description = ''
      requestAnimationFrame(() => formRef.value?.clearValidate?.())
    }
  },
)

const onVisibleUpdate = (next: boolean) => {
  if (!next && submitting.value) return
  emit('update:visible', next)
}

const handleClose = () => {
  if (submitting.value) return
  emit('update:visible', false)
}

const handleSubmit = async () => {
  if (submitting.value) return
  const validateResult = await formRef.value?.validate?.()
  if (validateResult !== true) return

  submitting.value = true
  try {
    const response = await createTenant({
      name: form.name.trim(),
      description: form.description.trim() || undefined,
    })
    if (!response.success || !response.data) {
      MessagePlugin.error(response.message || 'Failed to create workspace')
      return
    }
    MessagePlugin.success('Workspace created successfully')
    emit('created', response.data)
    emit('update:visible', false)
  } catch (error: any) {
    console.error('Failed to create tenant:', error)
    MessagePlugin.error(error?.message || 'Failed to create workspace')
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="less" scoped>
.create-tenant-dialog-header {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.create-tenant-dialog-header-icon {
  flex-shrink: 0;
  color: var(--td-brand-color);
}

.create-tenant-dialog-header-title {
  font: inherit;
}

.create-tenant-tip {
  margin: 0 0 16px;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.create-tenant-form {
  :deep(.t-form__item):last-child {
    margin-bottom: 0;
  }
}
</style>
