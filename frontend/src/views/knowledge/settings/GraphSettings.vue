<template>
  <div class="graph-settings" :class="{ 'graph-settings--embedded': embedded }">
    <div v-if="!embedded" class="section-header">
      <h2>{{ 'Knowledge Graph Configuration' }}</h2>
      <p class="section-description">{{ 'Configure entity-relationship extraction to automatically build a knowledge graph from text (note: this is different from the \u0022page-link graph\u0022 inside the Wiki — that one shows references between Wiki pages, while this one is an LLM-extracted entity-relationship graph)' }}</p>

      <t-alert
        v-if="!isGraphDatabaseEnabled"
        theme="warning"
        style="margin-top: 16px;"
      >
        <template #message>
          <div>{{ 'Knowledge graph database is not enabled, entity-relationship extraction will not be available' }}</div>
          <t-link class="graph-guide-link" theme="primary" @click="handleOpenGraphGuide">
            {{ 'How to enable knowledge graph?' }}
          </t-link>
        </template>
      </t-alert>
    </div>
    <t-alert
      v-else-if="!isGraphDatabaseEnabled"
      theme="warning"
      class="embedded-graph-alert"
    >
      <template #message>
        <div>{{ 'Knowledge graph database is not enabled, entity-relationship extraction will not be available' }}</div>
      </template>
    </t-alert>

    <div v-if="isGraphDatabaseEnabled" class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Enable Entity-Relationship Extraction' }}</label>
          <p class="desc">{{ 'Automatically extract entities and relationships from text when enabled' }}</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localGraphExtract.enabled"
            @change="handleEnabledChange"
          />
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ 'Additional Extraction Instructions' }}</label>
          <p class="desc">{{ 'Add domain scope, entity filtering, or attribute guidance while the system retains the structured output contract' }}</p>
        </div>
        <div class="setting-control full-width">
          <t-textarea
            v-model="localGraphExtract.customInstructions"
            :placeholder="'For example: focus on contract parties, amounts, performance dates, and liabilities…'"
            :maxlength="4000"
            :autosize="{ minRows: 3, maxRows: 8 }"
            @change="handleConfigChange"
          />
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ 'Relationship Types' }}</label>
          <p class="desc">{{ 'Define relationship type tags to extract, separated by commas' }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="tags-control-group">
            <t-button
              v-if="canRunGraphExtract"
              theme="default"
              size="medium"
              :disabled="!modelStatus.llm.available"
              :loading="tagFabring"
              @click="handleFabriTag"
              class="gen-tags-btn"
            >
              {{ 'Generate Random Tags' }}
            </t-button>
            <t-select
              v-model="localGraphExtract.tags"
              multiple
              :placeholder="'Enter relationship types, e.g., works_at, colleague, friend'"
              clearable
              creatable
              filterable
              @change="handleTagsChange"
              style="flex: 1; min-width: 400px;"
            />
          </div>
          <div v-if="!modelStatus.llm.available" class="control-tip">
            <t-icon name="info-circle" class="tip-icon" />
            <span>{{ 'Please complete model configuration first' }}</span>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ 'Sample Text' }}</label>
          <p class="desc">{{ 'Sample text for testing entity-relationship extraction' }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="text-control-group">
            <t-button
              v-if="canRunGraphExtract"
              theme="default"
              size="medium"
              :disabled="!modelStatus.llm.available"
              :loading="textFabring"
              @click="handleFabriText"
              class="gen-text-btn"
            >
              {{ 'Generate Random Text' }}
            </t-button>
            <t-textarea
              v-model="localGraphExtract.text"
              :placeholder="'Enter text containing entities and relationships...'"
              :autosize="{ minRows: 6, maxRows: 12 }"
              show-word-limit
              maxlength="5000"
              @change="handleTextChange"
              style="width: 100%;"
            />
          </div>
          <div v-if="!modelStatus.llm.available" class="control-tip">
            <t-icon name="info-circle" class="tip-icon" />
            <span>{{ 'Please complete model configuration first' }}</span>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled && localGraphExtract.nodes.length > 0" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ 'Entity List' }}</label>
          <p class="desc">{{ 'Entities and their attributes extracted from text' }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="node-list">
            <div v-for="(node, nodeIndex) in localGraphExtract.nodes" :key="nodeIndex" class="node-item">
              <div class="node-header">
                <t-icon name="user" class="node-icon" />
                <t-input
                  v-model="node.name"
                  :placeholder="'Enter entity name'"
                  @change="handleNodesChange"
                  class="node-name-input"
                />
                <t-button
                  theme="default"
                  size="small"
                  @click="removeNode(nodeIndex)"
                >
                  <t-icon name="delete" />
                </t-button>
              </div>
              <div class="node-attributes">
                <div v-for="(attribute, attrIndex) in node.attributes" :key="attrIndex" class="attribute-item">
                  <t-input
                    v-model="node.attributes[attrIndex]"
                    :placeholder="'Enter attribute value'"
                    @change="handleNodesChange"
                    class="attribute-input"
                  />
                  <t-button
                    theme="default"
                    size="small"
                    @click="removeAttribute(nodeIndex, attrIndex)"
                  >
                    <t-icon name="close" />
                  </t-button>
                </div>
                <t-button
                  theme="default"
                  size="small"
                  @click="addAttribute(nodeIndex)"
                  class="add-attr-btn"
                >
                  {{ 'Add Attribute' }}
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Manage Entities' }}</label>
          <p class="desc">{{ 'Add or remove entity nodes' }}</p>
        </div>
        <div class="setting-control">
          <t-button
            theme="primary"
            @click="addNode"
          >
            {{ 'Add Entity' }}
          </t-button>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled && localGraphExtract.relations.length > 0" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ 'Relationship List' }}</label>
          <p class="desc">{{ 'Define relationship connections between entities' }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="relation-list">
            <div v-for="(relation, index) in localGraphExtract.relations" :key="index" class="relation-item">
              <t-select
                v-model="relation.node1"
                :placeholder="'Select Entity'"
                @change="handleRelationsChange"
                class="relation-select"
              >
                <t-option
                  v-for="node in localGraphExtract.nodes"
                  :key="node.name"
                  :value="node.name"
                  :label="node.name"
                />
              </t-select>
              <t-icon name="arrow-right" class="relation-arrow" />
              <t-select
                v-model="relation.type"
                :placeholder="'Select Relationship Type'"
                clearable
                creatable
                filterable
                @change="handleRelationsChange"
                class="relation-select"
              >
                <t-option
                  v-for="tag in localGraphExtract.tags"
                  :key="tag"
                  :value="tag"
                  :label="tag"
                />
              </t-select>
              <t-icon name="arrow-right" class="relation-arrow" />
              <t-select
                v-model="relation.node2"
                :placeholder="'Select Entity'"
                @change="handleRelationsChange"
                class="relation-select"
              >
                <t-option
                  v-for="node in localGraphExtract.nodes"
                  :key="node.name"
                  :value="node.name"
                  :label="node.name"
                />
              </t-select>
              <t-button
                theme="default"
                size="small"
                @click="removeRelation(index)"
              >
                <t-icon name="delete" />
              </t-button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Manage Relationships' }}</label>
          <p class="desc">{{ 'Add or remove relationships between entities' }}</p>
        </div>
        <div class="setting-control">
          <t-button
            theme="primary"
            @click="addRelation"
          >
            {{ 'Add Relationship' }}
          </t-button>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Extraction Actions' }}</label>
          <p class="desc">{{ 'Perform entity-relationship extraction or manage sample data' }}</p>
        </div>
        <div class="setting-control">
          <div class="action-buttons">
            <t-button
              v-if="canRunGraphExtract"
              theme="primary"
              :disabled="!modelStatus.llm.available || !localGraphExtract.text"
              :loading="extracting"
              @click="handleExtract"
            >
              {{ extracting ? 'Extracting...' : 'Start Extraction' }}
            </t-button>
            <t-button
              theme="default"
              @click="defaultExtractExample"
            >
              {{ 'Default Example' }}
            </t-button>
            <t-button
              theme="default"
              @click="clearExtractExample"
            >
              {{ 'Clear Example' }}
            </t-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { extractTextRelations, fabriText, fabriTag, type Node, type Relation } from '@/api/initialization'
import { useEditorResourcesStore } from '@/stores/editorResources'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

// canRunGraphExtract mirrors the backend g.Admin() guard on POST
// /initialization/extract/{fabri-tag,fabri-text,text-relation}: all three
// call the LLM and write to the database, so a contributor clicking the
// button would only hit a 403.
const canRunGraphExtract = computed(() => authStore.hasRole('admin'))

interface GraphExtractConfig {
  enabled: boolean
  text: string
  tags: string[]
  nodes: Node[]
  relations: Relation[]
  customInstructions?: string
}

interface Props {
  graphExtract: GraphExtractConfig
  modelId: string
  allModels?: any[]
  embedded?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  embedded: false,
})

const emit = defineEmits<{
  'update:graphExtract': [value: GraphExtractConfig]
}>()

const modelStatus = computed(() => ({
  llm: {
    available: !!props.modelId
  }
}))

const localGraphExtract = ref<GraphExtractConfig>({
  ...props.graphExtract,
  nodes: props.graphExtract.nodes || [],
  relations: props.graphExtract.relations || [],
  customInstructions: props.graphExtract.customInstructions || ''
})

const tagFabring = ref(false)
const textFabring = ref(false)
const extracting = ref(false)

const systemInfo = ref<any>(null)

const isGraphDatabaseEnabled = computed(() => {
  return systemInfo.value?.graph_database_engine && systemInfo.value.graph_database_engine !== 'Not Enabled'
})

// Watch for prop changes
watch(() => props.graphExtract, (newVal) => {
  localGraphExtract.value = {
    ...newVal,
    nodes: newVal.nodes || [],
    relations: newVal.relations || [],
    customInstructions: newVal.customInstructions || ''
  }
}, { deep: true })

const handleConfigChange = () => {
  emit('update:graphExtract', localGraphExtract.value)
}

const handleEnabledChange = () => {
  // Clearing the extraction toggle wipes the sample data, but keeps
  // customInstructions so it is restored if extraction is enabled again.
  if (!localGraphExtract.value.enabled) {
    localGraphExtract.value.text = ''
    localGraphExtract.value.tags = []
    localGraphExtract.value.nodes = []
    localGraphExtract.value.relations = []
  }
  handleConfigChange()
}

const handleTagsChange = () => {
  handleConfigChange()
}

const handleTextChange = () => {
  handleConfigChange()
}

const handleNodesChange = () => {
  handleConfigChange()
}

const handleRelationsChange = () => {
  handleConfigChange()
}

const addNode = () => {
  if (!localGraphExtract.value.nodes) {
    localGraphExtract.value.nodes = []
  }
  localGraphExtract.value.nodes.push({
    name: '',
    attributes: []
  })
  handleNodesChange()
}

const removeNode = (index: number) => {
  localGraphExtract.value.nodes.splice(index, 1)
  handleNodesChange()
}

const addAttribute = (nodeIndex: number) => {
  localGraphExtract.value.nodes[nodeIndex].attributes.push('')
  handleNodesChange()
}

const removeAttribute = (nodeIndex: number, attrIndex: number) => {
  localGraphExtract.value.nodes[nodeIndex].attributes.splice(attrIndex, 1)
  handleNodesChange()
}

const addRelation = () => {
  if (!localGraphExtract.value.relations) {
    localGraphExtract.value.relations = []
  }
  localGraphExtract.value.relations.push({
    node1: '',
    node2: '',
    type: ''
  })
  handleRelationsChange()
}

const removeRelation = (index: number) => {
  localGraphExtract.value.relations.splice(index, 1)
  handleRelationsChange()
}

const handleFabriTag = async () => {
  tagFabring.value = true
  try {
    const response = await fabriTag({})
    localGraphExtract.value.tags = response.tags || []
    handleTagsChange()
    MessagePlugin.success('Tags generated successfully')
  } catch (error: any) {
    console.error('Failed to generate tags:', error)
    MessagePlugin.error('Failed to generate tags')
  } finally {
    tagFabring.value = false
  }
}

const handleFabriText = async () => {
  if (!props.modelId) {
    MessagePlugin.warning('Please complete model configuration first')
    return
  }
  
  textFabring.value = true
  try {
    const response = await fabriText({
      tags: localGraphExtract.value.tags,
      model_id: props.modelId
    })
    localGraphExtract.value.text = response.text || ''
    handleTextChange()
    MessagePlugin.success('Text generated successfully')
  } catch (error: any) {
    console.error('Failed to generate text:', error)
    MessagePlugin.error('Failed to generate text')
  } finally {
    textFabring.value = false
  }
}

const handleExtract = async () => {
  if (!props.modelId) {
    MessagePlugin.warning('Please complete model configuration first')
    return
  }
  
  if (!localGraphExtract.value.text) {
    MessagePlugin.warning('Please enter sample text first')
    return
  }
  
  extracting.value = true
  try {
    const response = await extractTextRelations({
      text: localGraphExtract.value.text,
      tags: localGraphExtract.value.tags,
      model_id: props.modelId
    })
    localGraphExtract.value.nodes = response.nodes || []
    localGraphExtract.value.relations = response.relations || []
    handleNodesChange()
    MessagePlugin.success('Entity-relationship extraction successful')
  } catch (error: any) {
    console.error('Failed to extract relations:', error)
    MessagePlugin.error('Entity-relationship extraction failed')
  } finally {
    extracting.value = false
  }
}

const defaultExtractExample = () => {
  localGraphExtract.value.text = `"Romeo and Juliet" is a tragedy written by William Shakespeare early in his career, and is one of the most frequently performed plays in world literature. The play follows two young lovers from feuding families in Verona, Italy — the Montagues and the Capulets. Written around 1594-1596, it was first published in quarto in 1597. The full title is "The Most Excellent and Lamentable Tragedy of Romeo and Juliet." The story has been adapted countless times for stage, film, and other media.`
  localGraphExtract.value.tags = ['Author', 'Alias']
  localGraphExtract.value.nodes = [
    {name: 'Romeo and Juliet', attributes: ['One of the most frequently performed plays', 'Written around 1594-1596', 'A tragedy']},
    {name: 'The Most Excellent and Lamentable Tragedy of Romeo and Juliet', attributes: ['Full title of Romeo and Juliet']},
    {name: 'William Shakespeare', attributes: ['English playwright', 'Author of Romeo and Juliet']},
    {name: 'Verona', attributes: ['City in Italy', 'Setting of the play']}
  ]
  localGraphExtract.value.relations = [
    {node1: 'Romeo and Juliet', node2: 'The Most Excellent and Lamentable Tragedy of Romeo and Juliet', type: 'Alias'},
    {node1: 'Romeo and Juliet', node2: 'William Shakespeare', type: 'Author'},
    {node1: 'Romeo and Juliet', node2: 'Verona', type: 'Setting'}
  ]
  handleNodesChange()
  MessagePlugin.success('Example loaded')
}

const clearExtractExample = () => {
  localGraphExtract.value.text = ''
  localGraphExtract.value.tags = []
  localGraphExtract.value.nodes = []
  localGraphExtract.value.relations = []
  handleNodesChange()
  MessagePlugin.success('Example cleared')
}

const editorResources = useEditorResourcesStore()

const loadSystemInfo = async (force = false) => {
  try {
    await editorResources.ensureSystemInfo(force)
    systemInfo.value = editorResources.systemInfo
  } catch (error: any) {
    console.error('Failed to load system info:', error)
  }
}

const graphGuideUrl =
  import.meta.env.VITE_KG_GUIDE_URL ||
  'https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md'

// Open guide documentation to show how to enable graph database
const handleOpenGraphGuide = () => {
  window.open(graphGuideUrl, '_blank', 'noopener')
}

onMounted(async () => {
  await loadSystemInfo()
})
</script>

<style lang="less" scoped>
.graph-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 20px;

  h2 {
    font-size: var(--app-text-3xl);
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 6px 0;
  }

  .section-description {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child {
    border-bottom: none;
  }

  &.vertical {
    flex-direction: column;
    gap: 12px;

    .setting-control {
      width: 100%;
      max-width: 100%;
    }
  }
}

.setting-info {
  flex: 0 0 40%;
  max-width: 40%;
  padding-right: 24px;

  label {
    font-size: var(--app-text-lg);
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex: 0 0 55%;
  max-width: 55%;
  display: flex;
  justify-content: flex-end;
  align-items: center;

  &.full-width {
    width: 100%;
    max-width: 100%;
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
}

.tags-control-group,
.text-control-group {
  display: flex;
  gap: 12px;
  width: 100%;
  align-items: flex-start;
}

.text-control-group {
  flex-direction: column;
}

.control-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);

  .tip-icon {
    color: var(--td-brand-color);
  }
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.node-item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  padding: 16px;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;

  .node-icon {
    font-size: var(--app-text-3xl);
    color: var(--td-brand-color);
  }

  .node-name-input {
    flex: 1;
  }
}

.node-attributes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-left: 32px;
}

.attribute-item {
  display: flex;
  gap: 8px;
  align-items: center;

  .attribute-input {
    flex: 1;
  }
}

.add-attr-btn {
  align-self: flex-start;
}

.relation-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.relation-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  .relation-select {
    flex: 1;
    min-width: 150px;
  }

  .relation-arrow {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-xl);
  }
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.graph-settings--embedded {
  .embedded-graph-alert {
    margin-bottom: 12px;
  }

  .setting-row:not(.vertical) {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    padding: 12px 0;
  }

  .setting-row:not(.vertical) .setting-info {
    flex: none;
    max-width: none;
    padding-right: 0;
  }

  .setting-row:not(.vertical) .setting-control {
    align-self: flex-start;
  }
}
</style>
