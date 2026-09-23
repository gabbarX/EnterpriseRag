<template>
  <div class="vectorstore-settings">
    <div class="section-header">
      <h2>{{ 'Vector Database Engine' }}</h2>
      <p class="section-description">{{ 'Register and manage vector database instances for knowledge base search.' }}</p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-container">
      <t-loading size="small" />
    </div>

    <template v-else>
      <div class="settings-group">
        <h3 class="list-section-title">{{ 'Vector Databases' }}</h3>

        <div v-if="stores.length === 0 && !authStore.hasRole('admin')" class="empty-stores">
          <t-empty :description="'No vector databases configured. Click \u0022Add Database\u0022 to get started.'" />
        </div>
        <div v-else class="store-grid">
          <div
            v-for="store in [...envStores, ...userStores]"
            :key="store.id"
            class="store-card"
            :class="[
              `store-card--${store.engine_type}`,
              {
                'store-card--env': store.source === 'env',
                'store-card--clickable': isStoreCardClickable(store),
              },
            ]"
            :role="isStoreCardClickable(store) ? 'button' : undefined"
            :tabindex="isStoreCardClickable(store) ? 0 : undefined"
            @click="onStoreCardClick($event, store)"
            @keydown.enter="onStoreCardClick($event, store)"
          >
            <div class="store-card__main">
              <div
                class="store-card__badge"
                :class="badgeClass(store.engine_type)"
                :style="badgeStyle(store.engine_type)"
                :aria-label="store.engine_type"
              >
                <img
                  v-if="resolveLogo(store.engine_type)?.mode === 'color'"
                  :src="resolveLogo(store.engine_type)!.url"
                  :alt="store.engine_type"
                  class="store-card__badge-img"
                />
                <template v-else-if="!resolveLogo(store.engine_type)">{{ engineInitial(store.engine_type) }}</template>
              </div>
              <div class="store-card__body">
                <div class="store-card__header">
                  <h3 class="store-card__title" :title="store.name">{{ store.name }}</h3>
                  <span v-if="store.source === 'env'" class="store-card__pill">
                    {{ 'DEFAULT' }}
                  </span>
                  <!-- Test connection lives in the edit drawer footer, so the
                       menu no longer exposes it. Env-sourced stores are written
                       by .env and have no actions at all, so no dropdown. -->
                  <div
                    v-if="authStore.hasRole('admin') && storeActionsFor(store).length > 0"
                    class="store-card__actions"
                    @click.stop
                  >
                    <t-dropdown
                      :options="storeActionsFor(store)"
                      placement="bottom-right"
                      attach="body"
                      trigger="click"
                      @click="(action: any) => handleAction(action, store)"
                    >
                      <t-button variant="text" shape="square" size="small" class="store-card__more">
                        <t-icon name="ellipsis" />
                      </t-button>
                    </t-dropdown>
                  </div>
                </div>
                <div class="store-card__subtitle">
                  <span class="store-card__type">{{ store.engine_type }}</span>
                  <template v-if="getStoreEndpoint(store)">
                    <span class="store-card__sep">·</span>
                    <span class="store-card__endpoint" :title="getStoreEndpoint(store)">{{ getStoreEndpoint(store) }}</span>
                  </template>
                </div>
              </div>
            </div>
          </div>
          <button
            v-if="authStore.hasRole('admin')"
            type="button"
            class="store-card store-card--add"
            @click="openAddDialog"
          >
            <span class="store-card--add__icon" aria-hidden="true">
              <add-icon />
            </span>
            <span class="store-card--add__label">{{ 'Add Database' }}</span>
          </button>
        </div>
      </div>
    </template>

    <SettingDrawer
      v-model:visible="showDialog"
      :title="editingStore ? 'Edit Database' : 'Add Database'"
      :class="drawerClass"
      :confirm-loading="saving"
      @confirm="onDrawerConfirm"
      @cancel="showDialog = false"
    >
      <!-- Header icon. Per-engine colouring is injected by the non-scoped
           .vectorstore-drawer--{engine} rules at the bottom of this file. -->
      <template v-if="form.engine_type" #headerIcon>
        <img
          v-if="drawerLogo?.mode === 'color'"
          :src="drawerLogo.url"
          :alt="form.engine_type"
          class="header-icon__img"
        />
        <span
          v-else-if="drawerLogo?.mode === 'mono'"
          class="header-icon__mono"
          :style="drawerLogoStyle"
        />
        <span v-else class="header-icon__text">{{ engineInitial(form.engine_type) }}</span>
      </template>

      <template v-if="selectedType" #subtitle>
        <span>{{ selectedType.display_name || form.engine_type }}</span>
      </template>

      <template #footer-left>
        <t-button
          variant="outline"
          :loading="testing"
          :disabled="!canTestConnection"
          @click="onDrawerTest"
        >
          <template #icon>
            <t-icon
              v-if="!testing && lastTestOk === true"
              name="check-circle-filled"
              class="status-icon available"
            />
            <t-icon
              v-else-if="!testing && lastTestOk === false"
              name="close-circle-filled"
              class="status-icon unavailable"
            />
          </template>
          {{ testing ? 'Testing...' : 'Test Connection' }}
        </t-button>
      </template>

      <t-form ref="formRef" :data="form" :rules="formRules" label-align="top" class="store-form">
        <section v-if="editingStore" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Basic' }}</h4>

          <div class="inline-alert inline-alert--info">
            <t-icon name="info-circle-filled" class="inline-alert__icon" />
            <span class="inline-alert__text">{{ 'Engine type, connection, and index settings cannot be changed after creation.\nTo change these, delete and recreate.' }}</span>
          </div>

          <div class="form-item">
            <label class="form-label required">{{ 'Name' }}</label>
            <t-input v-model="form.name" :placeholder="'e.g., my-vector-store'" />
          </div>

          <div class="readonly-fields">
            <div class="readonly-row">
              <span class="readonly-label">{{ 'Engine Type' }}</span>
              <span class="readonly-value">{{ selectedType?.display_name || editingStore.engine_type }}</span>
            </div>
            <template v-if="selectedType">
              <template v-for="field in selectedType.connection_fields" :key="field.name">
                <div v-if="field.sensitive || form.connection_config[field.name]" class="readonly-row">
                  <span class="readonly-label">{{ fieldLabel(field.name) }}</span>
                  <span class="readonly-value">
                    {{ field.sensitive ? '********' : form.connection_config[field.name] }}
                  </span>
                </div>
              </template>
            </template>
            <template v-if="selectedType?.index_fields?.length">
              <template v-for="field in selectedType.index_fields" :key="field.name">
                <div v-if="form.index_config[field.name]" class="readonly-row">
                  <span class="readonly-label">{{ fieldLabel(field.name) }}</span>
                  <span class="readonly-value">{{ form.index_config[field.name] }}</span>
                </div>
              </template>
            </template>
          </div>
        </section>

        <template v-else>
          <section class="setting-drawer__section">
            <h4 class="setting-drawer__section-title">{{ 'Basic' }}</h4>

            <div class="form-item">
              <label class="form-label required">{{ 'Engine Type' }}</label>
              <t-select v-model="form.engine_type" @change="onEngineTypeChange">
                <t-option
                  v-for="st in storeTypes"
                  :key="st.type"
                  :value="st.type"
                  :label="st.display_name"
                />
              </t-select>
            </div>

            <div class="form-item">
              <label class="form-label required">{{ 'Name' }}</label>
              <t-input v-model="form.name" :placeholder="'e.g., my-vector-store'" />
            </div>
          </section>

          <section v-if="selectedType" class="setting-drawer__section">
            <h4 class="setting-drawer__section-title">{{ 'Connection Info' }}</h4>

            <div
              v-for="field in selectedType.connection_fields"
              :key="field.name"
              class="form-item"
            >
              <label
                class="form-label"
                :class="{ required: field.required }"
              >{{ fieldLabel(field.name) }}</label>

              <template v-if="field.type === 'boolean'">
                <div class="vision-toggle">
                  <t-switch v-model="form.connection_config[field.name]" />
                </div>
                <p
                  v-if="field.name === 'insecure_skip_verify' && form.connection_config[field.name]"
                  class="form-desc form-desc--warn"
                >
                  {{ 'Disabling TLS certificate verification exposes the connection to man-in-the-middle attacks. Use only for self-signed development clusters — never in production.' }}
                </p>
              </template>

              <t-input
                v-else-if="field.type === 'string' && field.sensitive"
                v-model="form.connection_config[field.name]"
                type="password"
                placeholder="********"
              >
                <template #prefix-icon><t-icon name="lock-on" /></template>
              </t-input>

              <t-input
                v-else-if="field.type === 'number'"
                v-model="connectionNumberTextProxy[field.name].value"
                type="number"
                :placeholder="field.default != null ? String(field.default) : ' '"
                class="number-input"
              />

              <t-input
                v-else
                v-model="form.connection_config[field.name]"
                :placeholder="field.default?.toString() || ''"
              />
            </div>
          </section>

          <section v-if="selectedType?.index_fields?.length" class="setting-drawer__section">
            <h4 class="setting-drawer__section-title">{{ 'Advanced Settings' }}</h4>

            <button
              type="button"
              class="advanced-toggle"
              @click="showAdvanced = !showAdvanced"
            >
              <t-icon :name="showAdvanced ? 'chevron-down' : 'chevron-right'" />
              <span>{{ showAdvanced ? 'Collapse' : 'Expand' }}</span>
            </button>

            <template v-if="showAdvanced">
              <div
                v-for="field in selectedType.index_fields"
                :key="field.name"
                class="form-item"
              >
                <label class="form-label">{{ fieldLabel(field.name) }}</label>

                <t-select
                  v-if="field.enum && field.enum.length"
                  v-model="form.index_config[field.name]"
                  :placeholder="field.default?.toString() || ''"
                >
                  <t-option v-for="opt in field.enum" :key="opt" :value="opt" :label="opt" />
                </t-select>

                <t-input
                  v-else-if="field.type === 'number'"
                  v-model="indexNumberTextProxy[field.name].value"
                  type="number"
                  :placeholder="field.default?.toString()"
                  :min="field.min ?? 1"
                  :max="field.max ?? (isReplicaField(field.name) ? 10 : 64)"
                  class="number-input"
                />

                <t-input
                  v-else
                  v-model="form.index_config[field.name]"
                  :placeholder="field.default?.toString() || ''"
                  :maxlength="128"
                />
              </div>
            </template>
          </section>
        </template>
      </t-form>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, type WritableComputedRef } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { AddIcon } from 'tdesign-icons-vue-next'
import {
  listVectorStores,
  listVectorStoreTypes,
  createVectorStore,
  updateVectorStore,
  deleteVectorStore as deleteVectorStoreAPI,
  testVectorStoreRaw,
  type VectorStoreEntity,
  type VectorStoreTypeInfo,
} from '@/api/vector-store'
import { useAuthStore } from '@/stores/auth'
import { providerLogo } from './providerLogos'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'

const authStore = useAuthStore()

// ===== State =====
const stores = ref<VectorStoreEntity[]>([])
const storeTypes = ref<VectorStoreTypeInfo[]>([])
const loading = ref(false)
const showDialog = ref(false)
const editingStore = ref<VectorStoreEntity | null>(null)
const testing = ref(false)
const saving = ref(false)
const showAdvanced = ref(false)
const formRef = ref<any>()

const form = ref<{
  name: string
  engine_type: string
  connection_config: Record<string, any>
  index_config: Record<string, any>
}>({
  name: '',
  engine_type: '',
  connection_config: {},
  index_config: {},
})

// Tri-state hint icon next to the test button: null=neutral, true=just
// succeeded, false=just failed. Cleared when the user changes any
// connection-relevant field so a stale ✓/✗ doesn't follow a config the
// user is still editing.
const lastTestOk = ref<boolean | null>(null)

watch(
  () => [form.value.engine_type, form.value.connection_config],
  () => { lastTestOk.value = null },
  { deep: true },
)

// ===== Computed =====
const envStores = computed(() => stores.value.filter(s => s.source === 'env'))
const userStores = computed(() => stores.value.filter(s => s.source === 'user'))
const selectedType = computed(() => storeTypes.value.find(st => st.type === form.value.engine_type))

const drawerLogo = computed(() => {
  if (!form.value.engine_type) return null
  return providerLogo('vectorstore', form.value.engine_type)
})

const drawerLogoStyle = computed((): Record<string, string> => {
  const logo = drawerLogo.value
  if (!logo || logo.mode !== 'mono') return {}
  return { '--logo-url': `url("${logo.url}")` }
})

// per-engine class on drawer for non-scoped header-icon coloring rules.
const drawerClass = computed(() => {
  return form.value.engine_type
    ? `vectorstore-drawer vectorstore-drawer--${form.value.engine_type}`
    : 'vectorstore-drawer'
})

// Whether test connection is clickable. In create mode every required connection
// field must be filled; in edit mode the engine and the connection config are
// immutable, so there is nothing left to test.
const canTestConnection = computed(() => {
  if (editingStore.value) return false
  const st = selectedType.value
  if (!st) return false
  for (const f of st.connection_fields) {
    if (!f.required) continue
    const v = form.value.connection_config[f.name]
    if (v == null || v === '' || (typeof v === 'string' && v.trim() === '')) return false
  }
  return true
})

// Per-store dropdown options. Env-sourced stores are written by .env and can be
// neither edited nor deleted, and test connection now lives in the drawer footer
// — so no action is left for them and the dropdown is dropped entirely.
const storeActionsFor = (store: VectorStoreEntity) => {
  if (store.source === 'env') return []
  return [
    { content: 'Edit', value: 'edit' },
    { content: 'Delete', value: 'delete', theme: 'error' as const },
  ]
}

const formRules = computed(() => {
  const rules: Record<string, any[]> = {
    name: [{ required: true, message: 'Name is required' }],
  }
  if (!editingStore.value) {
    rules.engine_type = [{ required: true, message: 'Engine type is required' }]
    if (selectedType.value) {
      for (const field of selectedType.value.connection_fields) {
        if (field.required) {
          rules[`connection_config.${field.name}`] = [
            { required: true, message: `${fieldLabel(field.name)} is required` },
          ]
        }
      }
      // Index name/collection string fields: pattern validation (optional — empty is allowed)
      for (const field of (selectedType.value.index_fields || [])) {
        if (field.type === 'string') {
          rules[`index_config.${field.name}`] = [
            {
              validator: (val: string) => !val || indexNamePattern.test(val),
              message: 'Must start with a letter. Only letters, numbers, underscore, and hyphen allowed (max 128)',
              trigger: 'blur',
            },
          ]
        }
      }
    }
  }
  return rules
})

// Index/collection name pattern: must start with letter, alphanumeric + _ + - only, max 128
const indexNamePattern = /^[a-zA-Z][a-zA-Z0-9_-]{0,127}$/

// ===== Methods =====
const fieldLabel = (name: string): string => {
  const key = `vectorStoreSettings.fields.${name}`
  const translated = key
  // If i18n key not found, vue-i18n returns the key itself — fall back to field name
  return translated === key ? name : translated
}

// Distinguish replica fields (max 10) from shard fields (max 64) for input bounds
const replicaFieldNames = ['number_of_replicas', 'replication_factor', 'replica_number']
const isReplicaField = (name: string): boolean => replicaFieldNames.includes(name)

const getStoreEndpoint = (store: VectorStoreEntity): string => {
  const cc = store.connection_config || {}
  return cc.addr || cc.host || ''
}

// Badge initial. engine_type is always ASCII, so charAt is safe.
const engineInitial = (engineType: string): string => {
  return (engineType || '?').charAt(0).toUpperCase()
}

// When an engine ships a logo asset, pass its SVG URL to CSS (::before renders it
// with mask-image) and switch the card back to a neutral background; with no logo
// the per-engine brand-colour monogram is kept. Colour-mode logos need no mask
// tinting, so no URL is reported for them.
const resolveLogo = (engineType: string) => providerLogo('vectorstore', engineType)

const badgeClass = (engineType: string) => {
  const m = resolveLogo(engineType)?.mode
  return {
    'store-card__badge--logo': !!m,
    'store-card__badge--color': m === 'color',
    'store-card__badge--mono': m === 'mono',
  }
}

const badgeStyle = (engineType: string): Record<string, string> => {
  const logo = resolveLogo(engineType)
  return logo?.mode === 'mono' ? { '--logo-url': `url("${logo.url}")` } : {}
}

const onEngineTypeChange = () => {
  form.value.connection_config = {}
  form.value.index_config = {}
  showAdvanced.value = false
  // Drop cached number-text proxies so a switch to a different engine
  // doesn't keep stale entries pointing at the old field set.
  for (const k of Object.keys(connectionNumberText)) delete connectionNumberText[k]
  for (const k of Object.keys(indexNumberText)) delete indexNumberText[k]
}

// ---- Number-input text proxies (lazy per field name) ----
// With type=number, v-model coerces an empty string to 0 / NaN, which produces
// the irritating "user clears the field, 0 is pushed straight back" behaviour.
// A WritableComputedRef wraps that: reads render the number as a string, and a
// write of an empty string deletes the field (so the placeholder shows again)
// while a non-empty one is converted to an int. Proxies are created per field
// name on demand and cached, so the computed is not rebuilt on every render.
const connectionNumberText: Record<string, WritableComputedRef<string>> = {}
const indexNumberText: Record<string, WritableComputedRef<string>> = {}

function ensureNumberProxy(
  bag: Record<string, WritableComputedRef<string>>,
  store: Record<string, any>,
  key: string,
): WritableComputedRef<string> {
  if (bag[key]) return bag[key]
  bag[key] = computed<string>({
    get: () => {
      const v = store[key]
      return v == null || v === '' ? '' : String(v)
    },
    set: (raw: string) => {
      const s = String(raw ?? '').trim()
      if (!s) {
        delete store[key]
        return
      }
      const n = Number(s)
      store[key] = Number.isFinite(n) ? n : s
    },
  })
  return bag[key]
}

// Vue templates can't call ensureNumberProxy on every render without the
// keys multiplying — wrap in a Proxy so `connectionNumberText[name].value`
// from the template lazily creates the proxy on first read.
const connectionNumberTextProxy = new Proxy(connectionNumberText, {
  get: (target, name: string) => ensureNumberProxy(target, form.value.connection_config, name),
})
const indexNumberTextProxy = new Proxy(indexNumberText, {
  get: (target, name: string) => ensureNumberProxy(target, form.value.index_config, name),
})

const loadStores = async () => {
  try {
    const response = await listVectorStores()
    if (response.data && Array.isArray(response.data)) {
      stores.value = response.data
    }
  } catch (error) {
    console.error('Failed to load vector stores:', error)
  }
}

const loadStoreTypes = async () => {
  try {
    storeTypes.value = await listVectorStoreTypes()
  } catch (error) {
    console.error('Failed to load vector store types:', error)
  }
}

const openAddDialog = () => {
  editingStore.value = null
  showAdvanced.value = false
  form.value = {
    name: '',
    engine_type: storeTypes.value[0]?.type || '',
    connection_config: {},
    index_config: {},
  }
  lastTestOk.value = null
  showDialog.value = true
}

const isStoreCardClickable = (store: VectorStoreEntity) =>
  authStore.hasRole('admin') && store.source !== 'env'

const onStoreCardClick = (event: Event, store: VectorStoreEntity) => {
  if (!isStoreCardClickable(store)) return
  if (event.type === 'keydown') {
    const ke = event as KeyboardEvent
    if (ke.key !== 'Enter' && ke.key !== ' ') return
    ke.preventDefault()
  }
  const target = event.target as HTMLElement | null
  if (target?.closest('.store-card__actions')) return
  editStore(store)
}

const editStore = (store: VectorStoreEntity) => {
  if (store.source === 'env') {
    return
  }
  editingStore.value = store
  showAdvanced.value = false
  form.value = {
    name: store.name,
    engine_type: store.engine_type,
    connection_config: { ...store.connection_config },
    index_config: { ...store.index_config },
  }
  lastTestOk.value = null
  showDialog.value = true
}

const onDrawerConfirm = async () => {
  const result = await formRef.value?.validate()
  if (result !== true && result !== undefined) {
    const firstError =
      typeof result === 'object'
        ? Object.values(result).map((errs: any) => Array.isArray(errs) ? errs[0]?.message : '').find(Boolean)
        : ''
    MessagePlugin.warning(firstError || ('An error occurred. Please try again.' as string))
    return
  }

  saving.value = true
  try {
    if (editingStore.value) {
      await updateVectorStore(editingStore.value.id!, { name: form.value.name.trim() })
      MessagePlugin.success('Vector database updated')
    } else {
      const data: Partial<VectorStoreEntity> = {
        name: form.value.name.trim(),
        engine_type: form.value.engine_type,
        connection_config: { ...form.value.connection_config },
        index_config: showAdvanced.value ? { ...form.value.index_config } : {},
      }
      await createVectorStore(data)
      MessagePlugin.success('Vector database created')
    }
    showDialog.value = false
    await loadStores()
  } catch (error: any) {
    const msg = error?.message || 'An error occurred. Please try again.'
    if (msg.toLowerCase().includes('already exists') || msg.toLowerCase().includes('duplicate')) {
      MessagePlugin.error('A vector database with this name already exists')
    } else {
      MessagePlugin.error(msg)
    }
  } finally {
    saving.value = false
  }
}

const handleAction = (action: { value: string }, store: VectorStoreEntity) => {
  if (action.value === 'edit') {
    editStore(store)
  } else if (action.value === 'delete') {
    confirmDelete(store)
  }
}

const confirmDelete = (store: VectorStoreEntity) => {
  const dialog = DialogPlugin.confirm({
    header: 'Are you sure you want to delete this vector database?',
    confirmBtn: { content: 'Delete', theme: 'danger' },
    cancelBtn: 'Cancel',
    theme: 'danger',
    onConfirm: async () => {
      try {
        await deleteVectorStoreAPI(store.id!)
        MessagePlugin.success('Vector database deleted')
        await loadStores()
      } catch (error: any) {
        MessagePlugin.error(error?.message || 'An error occurred. Please try again.')
      }
      dialog.destroy()
    },
  })
}

// Test connection, triggered inside the drawer. The button is disabled in edit
// mode, so only the create path (current form data against /test/raw) runs here.
const onDrawerTest = async () => {
  if (editingStore.value) return
  testing.value = true
  try {
    const data = {
      engine_type: form.value.engine_type,
      connection_config: { ...form.value.connection_config },
    }
    const res = await testVectorStoreRaw(data)
    lastTestOk.value = !!res.success
    if (res.success) {
      MessagePlugin.success('Connection test succeeded')
    } else {
      MessagePlugin.error(res.error || 'Connection test failed')
    }
  } catch (error: any) {
    lastTestOk.value = false
    MessagePlugin.error(error?.message || 'Connection test failed')
  } finally {
    testing.value = false
  }
}

// ===== Init =====
onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadStoreTypes(), loadStores()])
  } finally {
    loading.value = false
  }
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/provider-card.less';

@import (reference) '@/components/css/settings-section.less';

.vectorstore-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.loading-container {
  display: flex;
  justify-content: center;
  padding: 48px 0;
}

.settings-group {
  display: flex;
  flex-direction: column;
}

.list-section-title {
  font-size: var(--app-text-xl);
  font-weight: 600;
  color: var(--td-text-color-primary);
  margin: 0 0 16px 0;
}

.store-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;

  .store-card--add {
    width: 100%;
    height: 100%;
  }
}

.store-card {
  .provider-card();
  flex-direction: column;

  &--env {
    background: var(--td-bg-color-secondarycontainer);
  }

  &--clickable {
    .provider-card-interactive();


  }

  &--env:not(.store-card--clickable):hover {
    border-color: var(--td-component-stroke);
    box-shadow: none;
  }

  &--add {
    .provider-card-add();

    &:hover,
    &:focus-visible {
      box-shadow: none;
    }



    &__icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 32px;
      height: 32px;
      border-radius: var(--app-radius-md);
      background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
      color: var(--td-brand-color);
      font-size: var(--app-text-2xl);
    }

    &__label {
      font-size: var(--app-text-md);
      font-weight: 500;
      line-height: 1.4;
    }
  }
}

.store-card__actions {
  flex-shrink: 0;
}

.store-card__main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.store-card__badge {
  .provider-card-badge();
  .provider-card-badge-color(#0052d9);
}

.store-card__badge-img {
  .provider-card-badge-img();
}

.store-card--qdrant .store-card__badge {
  .provider-card-badge-color(#e12626);
}
.store-card--milvus .store-card__badge {
  .provider-card-badge-color(#0089ff);
}
.store-card--weaviate .store-card__badge {
  .provider-card-badge-color(#07a050);
}
.store-card--elasticsearch .store-card__badge,
.store-card--elasticfaiss .store-card__badge {
  .provider-card-badge-color(#d97706);
}
.store-card--postgres .store-card__badge {
  .provider-card-badge-color(#0052d9);
}
.store-card--opensearch .store-card__badge {
  .provider-card-badge-color(#6235bb);
}
.store-card--infinity .store-card__badge {
  .provider-card-badge-color(#6235bb);
}
.store-card--sqlite .store-card__badge {
  .provider-card-badge-color(#464646);
}

.store-card__body {
  .provider-card-body();
}

.store-card__header {
  .provider-card-header();
}

.store-card__title {
  .provider-card-title();
}

.store-card__pill {
  flex-shrink: 0;
  padding: 1px 6px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 16px;
  border-radius: 3px;
  color: var(--td-warning-color-7);
  background: var(--td-warning-color-1);
}

.store-card__more {
  .provider-card-more();
}

.store-card:hover .store-card__more,
.store-card:focus-within .store-card__more,
.store-card__actions:focus-within .store-card__more {
  opacity: 1;
}

.store-card__subtitle {
  .provider-card-subtitle();
}

.store-card__type {
  font-weight: 500;
}

.store-card__sep {
  color: var(--td-text-color-placeholder);
}

.store-card__endpoint {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: var(--app-text-xs);
  color: var(--td-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}


.empty-stores {
  padding: 64px 0;
  text-align: center;

  :deep(.t-empty__description) {
    font-size: var(--app-text-base);
    color: var(--td-text-color-placeholder);
    margin-bottom: 16px;
  }
}

.form-item {
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);

  &--inline { margin: 0; }

  &--warn { color: var(--td-error-color); }
}

:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea) {
  width: 100%;
  font-size: var(--app-text-md);
}

// Hide TDesign's default form-item wrapper — we render .form-item / .form-label
:deep(.t-form) .t-form-item {
  display: none;
}

.vision-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

.inline-alert {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
  flex-wrap: wrap;

  white-space: pre-line;

  &__icon {
    font-size: var(--app-text-lg);
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
  }

  &__text {
    flex: 1 1 auto;
    min-width: 0;
  }

  &--info {
    color: var(--td-text-color-primary);

    .inline-alert__icon { color: var(--td-brand-color); }
  }
}

.readonly-fields {
  padding: 10px 12px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--app-radius-md);
}

.readonly-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 4px 0;
  font-size: var(--app-text-sm);
  line-height: 1.4;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child { border-bottom: none; }
}

.readonly-label {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xs);
  white-space: nowrap;
  min-width: 80px;
}

.readonly-value {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-sm);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  word-break: break-all;
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  background: transparent;
  border: none;
  font-family: inherit;
  cursor: pointer;
  user-select: none;
  align-self: flex-start;

  &:hover { color: var(--td-brand-color); }

  .t-icon { font-size: var(--app-text-base); }
}

.number-input {
  :deep(input::-webkit-outer-spin-button),
  :deep(input::-webkit-inner-spin-button) {
    -webkit-appearance: none;
    appearance: none;
    margin: 0;
  }

  :deep(input[type="number"]) {
    -moz-appearance: textfield;
    appearance: textfield;
  }
}

.header-icon__img {
  width: 24px;
  height: 24px;
  object-fit: contain;
  display: block;
}

.header-icon__mono {
  display: inline-block;
  width: 22px;
  height: 22px;
  background-color: currentColor;
  -webkit-mask-image: var(--logo-url);
  -webkit-mask-position: center;
  -webkit-mask-repeat: no-repeat;
  -webkit-mask-size: contain;
  mask-image: var(--logo-url);
  mask-position: center;
  mask-repeat: no-repeat;
  mask-size: contain;
}

.header-icon__text {
  font-size: var(--app-text-lg);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.status-icon {
  font-size: var(--app-text-xl);
  flex-shrink: 0;

  &.available { color: var(--td-brand-color); }
  &.unavailable { color: var(--td-error-color); }
}
</style>

<!--
  Non-scoped block: per-engine header-icon coloring + color-logo background
  tweak. Same pattern as Storage/Parser/WebSearch drawers — these rules
  must be global so they reach the t-drawer panel even if its scoped
  data-attribute is dropped in some builds. Each rule mirrors the matching
  .store-card--{engine} .store-card__badge from the scoped block above so
  list-card → drawer hand-off stays visually continuous.
-->
<style lang="less">
.vectorstore-drawer .setting-drawer__header-icon:has(.header-icon__img) {
  background: var(--td-bg-color-container);
  box-shadow: inset 0 0 0 1px var(--td-component-stroke);
}

.vectorstore-drawer--qdrant .setting-drawer__header-icon {
  background: rgba(225, 38, 38, 0.12);
  color: #E12626;
}
.vectorstore-drawer--milvus .setting-drawer__header-icon {
  background: rgba(0, 137, 255, 0.12);
  color: #0089FF;
}
.vectorstore-drawer--weaviate .setting-drawer__header-icon {
  background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: #07A050;
}
.vectorstore-drawer--elasticsearch .setting-drawer__header-icon,
.vectorstore-drawer--elasticfaiss .setting-drawer__header-icon {
  background: rgba(255, 153, 0, 0.12);
  color: #D97706;
}
.vectorstore-drawer--postgres .setting-drawer__header-icon {
  background: rgba(0, 82, 217, 0.1);
  color: #0052D9;
}
.vectorstore-drawer--opensearch .setting-drawer__header-icon {
  background: rgba(98, 53, 187, 0.12);
  color: #6235BB;
}
.vectorstore-drawer--infinity .setting-drawer__header-icon {
  background: rgba(98, 53, 187, 0.12);
  color: #6235BB;
}
.vectorstore-drawer--sqlite .setting-drawer__header-icon {
  background: rgba(70, 70, 70, 0.1);
  color: #464646;
}
</style>
