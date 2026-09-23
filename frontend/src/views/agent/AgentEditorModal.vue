<template>
  <SettingsModalShell :visible="visible"
    :title="editorMode === 'create' ? 'Create Agent' : 'Edit Agent'"
    v-model="currentSection" :nav-groups="navGroups" :loading="editorInitializing" :z-index="1000"
    nav-guide="agent-editor-sidebar" nav-item-guide-prefix="agent-editor-nav" @close="modalShell.requestClose">
    <div ref="contentWrapperRef" class="content-wrapper" :class="{ 'content-wrapper--prompts': currentSection === 'prompts' }">
      <div v-show="currentSection === 'basic'" class="section">
        <div class="section-header">
          <div class="section-header-title">
            <h2>{{ 'Basic Info' }}</h2>
            <t-tooltip v-if="isBuiltinAgent" :content="'This is a built-in agent. Name and description cannot be modified, but configuration parameters can be adjusted.'" placement="top">
              <span class="builtin-agent-hint" tabindex="0" role="img"
                :aria-label="'This is a built-in agent. Name and description cannot be modified, but configuration parameters can be adjusted.'">
                <t-icon name="info-circle" />
              </span>
            </t-tooltip>
          </div>
          <p class="section-description">{{ 'Configure agent name, description, and run mode' }}</p>
        </div>

        <div class="settings-group">
          <div v-if="editorMode === 'edit' && editorAgent?.id" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Agent ID' }}</label>
              <p class="desc">{{ 'Use this ID to target the agent in API integrations' }}</p>
            </div>
            <div class="setting-control">
              <div class="agent-id-field">
                <code class="agent-id-value" :title="editorAgent.id">{{ editorAgent.id }}</code>
                <t-tooltip :content="'Copy'" placement="top">
                  <t-button theme="default" size="small" variant="text" class="agent-id-copy"
                    @click="copyAgentId">
                    <t-icon name="file-copy" />
                  </t-button>
                </t-tooltip>
              </div>
            </div>
          </div>

          <div v-if="editorMode === 'edit' && editorAgent?.id" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Publish channels' }}</label>
              <p class="desc">{{ isPostCreateSession ? 'Go to Integrations to configure IM, web embed, and other publishing channels' : 'Publish this agent to IM platforms or websites. Manage in Integrations.' }}</p>
            </div>
            <div class="setting-control">
              <div class="integration-inline">
                <button type="button" class="integration-inline__stat integration-inline__link" @click="gotoIntegrations('im')">
                  <span>{{ 'IM Integration' }} · {{ agentIMChannelCount }}</span>
                  <t-icon name="chevron-right" size="14px" />
                </button>
                <span class="integration-inline__sep" aria-hidden="true">|</span>
                <button type="button" class="integration-inline__stat integration-inline__link" @click="gotoIntegrations('embed')">
                  <span>{{ 'Web Embed' }} · {{ agentEmbedChannelCount }}</span>
                  <t-icon name="chevron-right" size="14px" />
                </button>
              </div>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Running Mode' }} <span class="required">*</span></label>
              <p class="desc">{{ agentMode === 'smart-reasoning' ? 'Multi-step thinking, deep analysis for complex questions' :
                'Quick response, direct answers' }}</p>
            </div>
            <div class="setting-control">
              <t-radio-group v-model="agentMode" :disabled="isBuiltinAgent" data-guide="agent-create-mode">
                <t-radio-button value="quick-answer">
                  {{ 'Quick Answer' }}
                </t-radio-button>
                <t-radio-button value="smart-reasoning">
                  {{ 'Smart Reasoning' }}
                </t-radio-button>
              </t-radio-group>
            </div>
          </div>

          <div v-if="isAgentMode && agentTypePresets.length > 0" class="setting-row setting-row--emphasize"
            data-guide="agent-create-agent-type">
            <div class="setting-info">
              <label>{{ 'Agent Type' }}</label>
              <p class="desc">{{ 'Picking a preset auto-fills the system prompt, tool list and recommended KB scope.' }}</p>
              <p v-if="activeAgentTypePreset" class="desc agent-type-preset-desc">{{
                agentTypePresetDescription(activeAgentTypePreset) }}</p>
            </div>
            <div class="setting-control">
              <t-select :value="agentType" @change="onAgentTypeChange" :disabled="isBuiltinAgent"
                :placeholder="'Agent Type'" :options="agentTypeSelectOptions"
                :popup-props="{ overlayClassName: 'agent-type-popup' }" class="agent-type-select">
                <template #option="{ option }">
                  <div class="agent-type-option">
                    <span class="agent-type-option-label">{{ option.label }}</span>
                    <span v-if="option.desc" class="agent-type-option-desc">{{ option.desc }}</span>
                  </div>
                </template>
              </t-select>
            </div>
          </div>

          <div class="setting-row" data-guide="agent-create-name">
            <div class="setting-info">
              <label>{{ 'Name' }} <span v-if="!isBuiltinAgent"
                  class="required">*</span></label>
              <p class="desc">{{ 'Set an easily identifiable name for the agent' }}</p>
            </div>
            <div class="setting-control">
              <div class="name-input-wrapper">
                <div v-if="isBuiltinAgent" class="builtin-avatar" :class="isAgentMode ? 'agent' : 'normal'">
                  <t-icon :name="isAgentMode ? 'control-platform' : 'chat'" size="24px" />
                </div>
                <AgentAvatar v-else :name="formData.name || '?'" size="medium" />
                <t-input v-model="formData.name" :placeholder="'Enter agent name'"
                  class="name-input" :disabled="isBuiltinAgent" />
              </div>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Description' }}</label>
              <p class="desc">{{ 'Briefly describe the purpose and features of the agent' }}</p>
            </div>
            <div class="setting-control">
              <t-textarea v-model="formData.description"
                :placeholder="'Enter agent description'"
                :autosize="{ minRows: 2, maxRows: 4 }" :disabled="isBuiltinAgent" />
            </div>
          </div>

          <!-- Long-term memory lives in Basic info, not Conversation: it is unrelated
               to the multi-turn history window, and smart reasoning is exactly the mode
               that needs this switch. The switch can only turn memory off; enabling it
               here has no effect while the workspace or personal setting is off. -->
          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Long-term Memory' }}</label>
              <p class="desc">{{ 'Let this agent read and add to your long-term memory. When off, conversations with it neither read your memories nor add new ones. Turning it on here has no effect while the workspace or personal switch is off' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.memory_enabled" />
            </div>
          </div>

        </div>
      </div>

      <div v-show="currentSection === 'prompts'" class="section section--prompts">
        <div class="prompts-panel">
          <div class="prompts-panel__header">
            <div class="section-header section-header--compact">
              <h2>{{ 'Prompts' }}</h2>
              <p class="section-description">{{ 'Configure system, context, intent, rewrite, and fallback prompts' }}</p>
            </div>

            <nav v-if="promptNavItems.length > 1" class="prompts-outline"
              :aria-label="'Prompt outline'">
              <button v-for="item in promptNavItems" :key="item.key" type="button"
                class="prompts-outline__pill"
                :class="{ 'prompts-outline__pill--active': activePromptAnchor === item.key }"
                @click="activePromptAnchor = item.key">
                <span>{{ item.label }}</span>
                <span v-if="item.customized" class="prompts-outline__dot"
                  :title="'Customized'" />
              </button>
            </nav>
          </div>

          <div class="prompts-panel__body">
            <div class="settings-group">
              <div v-show="activePromptAnchor === 'system'"
                class="setting-row setting-row-vertical prompts-panel__pane">
            <div class="setting-info">
              <label>{{ 'System Prompt' }} <span v-if="!isBuiltinAgent"
                  class="required">*</span></label>
              <p class="desc">{{ 'Custom system prompt to define the agent behavior and role' }}{{ isBuiltinAgent ?
                '(leave empty to use system default)' : '' }}</p>
              <p class="desc">{{ 'Unchanged template text follows template updates; edited text is saved as a custom prompt. In agent mode, this field defines the role and workflow; tool permissions and per-turn source selections are controlled separately.' }}</p>
              <div class="placeholder-tags">
                <span class="placeholder-label">{{ 'Available variables: ' }}</span>
                <t-tooltip v-for="placeholder in availablePlaceholders" :key="placeholder.name"
                  :content="placeholder.description + '(click to insert)'"
                  placement="top">
                  <span class="placeholder-tag" @click="handlePlaceholderClick('system', placeholder.name)"
                    v-text="'{{' + placeholder.name + '}}'"></span>
                </t-tooltip>
                <span class="placeholder-hint">{{ '(click to insert, or type {\'{{\'} to show list)' }}</span>
              </div>
            </div>
            <div class="setting-control setting-control-full" style="position: relative;">
              <div v-if="isAgentMode" class="textarea-with-template">
                <t-textarea ref="promptTextareaRef" v-model="formData.config.system_prompt"
                  :placeholder="systemPromptPlaceholder" :autosize="{ minRows: 10, maxRows: 25 }"
                  @input="handlePromptInput" class="system-prompt-textarea" />
                <PromptTemplateSelector type="agentSystemPrompt" position="corner"
                  :hasKnowledgeBase="hasKnowledgeBase" @select="handleSystemPromptTemplateSelect"
                  @reset-default="handleAgentSystemPromptResetDefault" />
              </div>
              <div v-else class="textarea-with-template">
                <t-textarea ref="promptTextareaRef" v-model="formData.config.system_prompt"
                  :placeholder="systemPromptPlaceholder" :autosize="{ minRows: 10, maxRows: 25 }"
                  @input="handlePromptInput" class="system-prompt-textarea" />
                <PromptTemplateSelector type="systemPrompt" position="corner"
                  :hasKnowledgeBase="hasKnowledgeBase" @select="handleSystemPromptTemplateSelect"
                  @reset-default="handleSystemPromptTemplateSelect" />
              </div>
              <Teleport to="body">
                <div v-if="showPlaceholderPopup && filteredPlaceholders.length > 0"
                  class="placeholder-popup-wrapper" :style="popupStyle">
                  <div class="placeholder-popup">
                    <div v-for="(placeholder, index) in filteredPlaceholders" :key="placeholder.name"
                      class="placeholder-item" :class="{ active: selectedPlaceholderIndex === index }"
                      @mousedown.prevent="insertPlaceholder(placeholder.name, true)"
                      @mouseenter="selectedPlaceholderIndex = index">
                      <div class="placeholder-name">
                        <code v-html="`{{${placeholder.name}}}`"></code>
                      </div>
                      <div class="placeholder-desc">{{ placeholder.description }}</div>
                    </div>
                  </div>
                </div>
              </Teleport>
            </div>
          </div>

          <div v-if="!isAgentMode" v-show="activePromptAnchor === 'context'"
            class="setting-row setting-row-vertical prompts-panel__pane">
            <div class="setting-info">
              <label>{{ 'Context Template' }} <span v-if="!isBuiltinAgent"
                  class="required">*</span></label>
              <p class="desc">{{ 'Define how retrieved content is formatted before passing to the model' }}{{ isBuiltinAgent ?
                '(leave empty to use system default)' : '' }}</p>
              <div class="placeholder-tags">
                <span class="placeholder-label">{{ 'Available variables: ' }}</span>
                <t-tooltip v-for="placeholder in contextTemplatePlaceholders" :key="placeholder.name"
                  :content="placeholder.description + '(click to insert)'"
                  placement="top">
                  <span class="placeholder-tag" @click="handlePlaceholderClick('context', placeholder.name)"
                    v-text="'{{' + placeholder.name + '}}'"></span>
                </t-tooltip>
                <span class="placeholder-hint">{{ '(click to insert, or type {\'{{\'} to show list)' }}</span>
              </div>
            </div>
            <div class="setting-control setting-control-full" style="position: relative;">
              <div class="textarea-with-template">
                <t-textarea ref="contextTemplateTextareaRef" v-model="formData.config.context_template"
                  :placeholder="contextTemplatePlaceholder" :autosize="{ minRows: 8, maxRows: 20 }"
                  @input="handleContextTemplateInput" class="system-prompt-textarea" />
                <PromptTemplateSelector type="contextTemplate" position="corner"
                  :hasKnowledgeBase="hasKnowledgeBase" @select="handleContextTemplateSelect"
                  @reset-default="handleContextTemplateSelect" />
              </div>
              <Teleport to="body">
                <div v-if="showContextPlaceholderPopup && filteredContextPlaceholders.length > 0"
                  class="placeholder-popup-wrapper" :style="contextPopupStyle">
                  <div class="placeholder-popup">
                    <div v-for="(placeholder, index) in filteredContextPlaceholders" :key="placeholder.name"
                      class="placeholder-item" :class="{ active: selectedContextPlaceholderIndex === index }"
                      @mousedown.prevent="insertContextPlaceholder(placeholder.name, true)"
                      @mouseenter="selectedContextPlaceholderIndex = index">
                      <div class="placeholder-name">
                        <code v-html="`{{${placeholder.name}}}`"></code>
                      </div>
                      <div class="placeholder-desc">{{ placeholder.description }}</div>
                    </div>
                  </div>
                </div>
              </Teleport>
            </div>
          </div>

          <div v-if="!isAgentMode" v-show="activePromptAnchor === 'intent'"
            class="setting-row setting-row-vertical prompts-panel__pane">
            <div class="setting-info">
              <label>{{ 'Intent Prompts' }}</label>
              <p class="desc">{{ 'Configure intent-specific system prompts; defaults apply when not customized' }}</p>
            </div>
            <div class="setting-control setting-control-full">
              <div class="intent-prompts-editor">
                <div v-if="intentPromptTemplates.length === 0" class="prompt-disabled-hint">
                  {{ 'No intent templates available' }}
                </div>
                <template v-else>
                  <div class="intent-toggle-group" role="tablist"
                    :aria-label="'Intent'">
                    <t-button v-for="template in intentPromptTemplates" :key="template.id" theme="default"
                      variant="outline" size="small" class="intent-toggle-btn"
                      :class="{ 'intent-toggle-btn--active': selectedIntent === template.id }"
                      :disabled="props.readOnly" @click="selectedIntent = template.id">
                      <span class="intent-toggle-label">
                        {{ template.name || template.id }}
                        <t-tooltip v-if="isIntentCustomized(template.id)"
                          :content="'Customized'" placement="top">
                          <span class="intent-toggle-dot" />
                        </t-tooltip>
                      </span>
                    </t-button>
                  </div>
                  <p v-if="currentIntentTemplateDesc" class="intent-active-desc">{{ currentIntentTemplateDesc
                  }}</p>

                  <div v-if="placeholderData.system_prompt.length > 0" class="placeholder-tags">
                    <span class="placeholder-label">{{ 'Available variables: ' }}</span>
                    <t-tooltip v-for="placeholder in placeholderData.system_prompt" :key="placeholder.name"
                      :content="placeholder.description + '(click to insert)'"
                      placement="top">
                      <span class="placeholder-tag"
                        @click="handlePlaceholderClick('intent', placeholder.name)"
                        v-text="'{{' + placeholder.name + '}}'" />
                    </t-tooltip>
                    <span class="placeholder-hint">{{ '(click to insert, or type {\'{{\'} to show list)' }}</span>
                  </div>

                  <div class="textarea-with-template">
                    <t-textarea ref="intentPromptTextareaRef" v-model="intentEditorValue"
                      class="system-prompt-textarea" :autosize="{ minRows: 10, maxRows: 25 }"
                      :disabled="props.readOnly || !selectedIntent"
                      :placeholder="currentIntentTemplate?.content || 'Enter a custom system prompt...'"
                      @input="handleIntentPromptInput" />
                    <PromptTemplateSelector type="intentPrompt" position="corner" :intent-id="selectedIntent"
                      :show-template-picker="false" @reset-default="resetCurrentIntentPrompt" />
                  </div>

                  <Teleport to="body">
                    <div v-if="intentPromptPopup.show && filteredIntentPlaceholders.length > 0"
                      class="placeholder-popup-wrapper" :style="intentPromptPopup.style">
                      <div class="placeholder-popup">
                        <div v-for="(placeholder, index) in filteredIntentPlaceholders" :key="placeholder.name"
                          class="placeholder-item"
                          :class="{ active: intentPromptPopup.selectedIndex === index }"
                          @mousedown.prevent="insertGenericPlaceholder('intent', placeholder.name, true)"
                          @mouseenter="intentPromptPopup.selectedIndex = index">
                          <div class="placeholder-name">
                            <code v-html="`{{${placeholder.name}}}`" />
                          </div>
                          <div class="placeholder-desc">{{ placeholder.description }}</div>
                        </div>
                      </div>
                    </div>
                  </Teleport>
                </template>
              </div>
            </div>
          </div>

          <template
            v-if="!isAgentMode && formData.config.multi_turn_enabled && formData.config.enable_rewrite">
            <div v-show="activePromptAnchor === 'rewrite-system'"
              class="setting-row setting-row-vertical prompts-panel__pane">
              <div class="setting-info">
                <label>{{ 'Rewrite System Prompt' }}</label>
                <p class="desc">{{ 'System prompt for question rewriting (leave empty for default)' }}</p>
                <div class="placeholder-tags" v-if="rewriteSystemPlaceholders.length > 0">
                  <span class="placeholder-label">{{ 'Available variables: ' }}</span>
                  <t-tooltip v-for="placeholder in rewriteSystemPlaceholders" :key="placeholder.name"
                    :content="placeholder.description + '(click to insert)'"
                    placement="top">
                    <span class="placeholder-tag"
                      @click="handlePlaceholderClick('rewriteSystem', placeholder.name)"
                      v-text="'{{' + placeholder.name + '}}'"></span>
                  </t-tooltip>
                  <span class="placeholder-hint">{{ '(click to insert, or type {\'{{\'} to show list)' }}</span>
                </div>
              </div>
              <div class="setting-control setting-control-full" style="position: relative;">
                <div class="textarea-with-template">
                  <t-textarea ref="rewriteSystemTextareaRef" v-model="formData.config.rewrite_prompt_system"
                    :placeholder="defaultRewritePromptSystem || 'Leave empty to use default prompt'"
                    :autosize="{ minRows: 4, maxRows: 10 }" @input="handleRewriteSystemInput" />
                  <PromptTemplateSelector type="rewrite" position="corner" @select="handleRewriteTemplateSelect"
                    @reset-default="handleRewriteTemplateSelect" />
                </div>
                <Teleport to="body">
                  <div v-if="rewriteSystemPopup.show && filteredRewriteSystemPlaceholders.length > 0"
                    class="placeholder-popup-wrapper" :style="rewriteSystemPopup.style">
                    <div class="placeholder-popup">
                      <div v-for="(placeholder, index) in filteredRewriteSystemPlaceholders"
                        :key="placeholder.name" class="placeholder-item"
                        :class="{ active: rewriteSystemPopup.selectedIndex === index }"
                        @mousedown.prevent="insertGenericPlaceholder('rewriteSystem', placeholder.name, true)"
                        @mouseenter="rewriteSystemPopup.selectedIndex = index">
                        <div class="placeholder-name">
                          <code v-html="`{{${placeholder.name}}}`"></code>
                        </div>
                        <div class="placeholder-desc">{{ placeholder.description }}</div>
                      </div>
                    </div>
                  </div>
                </Teleport>
              </div>
            </div>

            <div v-show="activePromptAnchor === 'rewrite-user'"
              class="setting-row setting-row-vertical prompts-panel__pane">
              <div class="setting-info">
                <label>{{ 'Rewrite User Prompt' }}</label>
                <p class="desc">{{ 'User prompt template for question rewriting (leave empty for default)' }}</p>
                <div class="placeholder-tags" v-if="rewritePlaceholders.length > 0">
                  <span class="placeholder-label">{{ 'Available variables: ' }}</span>
                  <t-tooltip v-for="placeholder in rewritePlaceholders" :key="placeholder.name"
                    :content="placeholder.description + '(click to insert)'"
                    placement="top">
                    <span class="placeholder-tag"
                      @click="handlePlaceholderClick('rewriteUser', placeholder.name)"
                      v-text="'{{' + placeholder.name + '}}'"></span>
                  </t-tooltip>
                  <span class="placeholder-hint">{{ '(click to insert, or type {\'{{\'} to show list)' }}</span>
                </div>
              </div>
              <div class="setting-control setting-control-full" style="position: relative;">
                <div class="textarea-with-template">
                  <t-textarea ref="rewriteUserTextareaRef" v-model="formData.config.rewrite_prompt_user"
                    :placeholder="defaultRewritePromptUser || 'Leave empty to use default prompt'"
                    :autosize="{ minRows: 4, maxRows: 10 }" @input="handleRewriteUserInput" />
                  <PromptTemplateSelector type="rewrite" position="corner" @select="handleRewriteTemplateSelect"
                    @reset-default="handleRewriteTemplateSelect" />
                </div>
                <Teleport to="body">
                  <div v-if="rewriteUserPopup.show && filteredRewriteUserPlaceholders.length > 0"
                    class="placeholder-popup-wrapper" :style="rewriteUserPopup.style">
                    <div class="placeholder-popup">
                      <div v-for="(placeholder, index) in filteredRewriteUserPlaceholders"
                        :key="placeholder.name" class="placeholder-item"
                        :class="{ active: rewriteUserPopup.selectedIndex === index }"
                        @mousedown.prevent="insertGenericPlaceholder('rewriteUser', placeholder.name, true)"
                        @mouseenter="rewriteUserPopup.selectedIndex = index">
                        <div class="placeholder-name">
                          <code v-html="`{{${placeholder.name}}}`"></code>
                        </div>
                        <div class="placeholder-desc">{{ placeholder.description }}</div>
                      </div>
                    </div>
                  </div>
                </Teleport>
              </div>
            </div>
          </template>

          <div v-if="!isAgentMode && hasKnowledgeBase" v-show="activePromptAnchor === 'fallback'"
            class="prompts-panel__pane prompts-panel__pane--stack">
            <div class="setting-row">
              <div class="setting-info">
                <label>{{ 'Fallback Strategy' }}</label>
                <p class="desc">{{ 'How to handle when no relevant content is found in the knowledge base' }}</p>
              </div>
              <div class="setting-control">
                <t-radio-group v-model="formData.config.fallback_strategy">
                  <t-radio-button value="fixed">{{ 'Fixed Response' }}</t-radio-button>
                  <t-radio-button value="model">{{ 'Model Generated' }}</t-radio-button>
                </t-radio-group>
              </div>
            </div>

            <div v-if="formData.config.fallback_strategy === 'fixed'"
              class="setting-row setting-row-vertical">
              <div class="setting-info">
                <label>{{ 'Fixed Response' }}</label>
                <p class="desc">{{ 'Fixed text returned when unable to answer' }}</p>
              </div>
              <div class="setting-control setting-control-full">
                <div class="textarea-with-template">
                  <t-textarea v-model="formData.config.fallback_response"
                    :placeholder="defaultFallbackResponse || 'Sorry, I cannot answer this question.'"
                    :autosize="{ minRows: 2, maxRows: 6 }" />
                  <PromptTemplateSelector type="fallback" position="corner" fallbackMode="fixed"
                    @select="handleFallbackResponseTemplateSelect"
                    @reset-default="handleFallbackResponseTemplateSelect" />
                </div>
              </div>
            </div>

            <div v-if="formData.config.fallback_strategy === 'model'"
              class="setting-row setting-row-vertical">
              <div class="setting-info">
                <label>{{ 'Fallback Prompt' }}</label>
                <p class="desc">{{ 'Prompt to guide model response when no answer is found in knowledge base' }}</p>
                <div class="placeholder-tags" v-if="fallbackPlaceholders.length > 0">
                  <span class="placeholder-label">{{ 'Available variables: ' }}</span>
                  <t-tooltip v-for="placeholder in fallbackPlaceholders" :key="placeholder.name"
                    :content="placeholder.description + '(click to insert)'"
                    placement="top">
                    <span class="placeholder-tag"
                      @click="handlePlaceholderClick('fallback', placeholder.name)"
                      v-text="'{{' + placeholder.name + '}}'"></span>
                  </t-tooltip>
                  <span class="placeholder-hint">{{ '(click to insert, or type {\'{{\'} to show list)' }}</span>
                </div>
              </div>
              <div class="setting-control setting-control-full" style="position: relative;">
                <div class="textarea-with-template">
                  <t-textarea ref="fallbackPromptTextareaRef" v-model="formData.config.fallback_prompt"
                    :placeholder="defaultFallbackPrompt || 'Leave empty to use default prompt'"
                    :autosize="{ minRows: 4, maxRows: 10 }" @input="handleFallbackPromptInput" />
                  <PromptTemplateSelector type="fallback" position="corner" fallbackMode="model"
                    @select="handleFallbackPromptTemplateSelect"
                    @reset-default="handleFallbackPromptTemplateSelect" />
                </div>
                <Teleport to="body">
                  <div v-if="fallbackPromptPopup.show && filteredFallbackPlaceholders.length > 0"
                    class="placeholder-popup-wrapper" :style="fallbackPromptPopup.style">
                    <div class="placeholder-popup">
                      <div v-for="(placeholder, index) in filteredFallbackPlaceholders"
                        :key="placeholder.name" class="placeholder-item"
                        :class="{ active: fallbackPromptPopup.selectedIndex === index }"
                        @mousedown.prevent="insertGenericPlaceholder('fallback', placeholder.name, true)"
                        @mouseenter="fallbackPromptPopup.selectedIndex = index">
                        <div class="placeholder-name">
                          <code v-html="`{{${placeholder.name}}}`"></code>
                        </div>
                        <div class="placeholder-desc">{{ placeholder.description }}</div>
                      </div>
                    </div>
                  </div>
                </Teleport>
              </div>
            </div>
          </div>

            </div>
          </div>
        </div>
      </div>

      <div v-show="currentSection === 'model'" class="section">
        <div class="section-header">
          <h2>{{ 'Model Config' }}</h2>
          <p class="section-description">{{ 'Configure chat model, auxiliary models (ReRank), and generation parameters' }}</p>
        </div>

        <div class="settings-group">
          <div
            class="setting-row"
            data-guide="agent-create-model"
            data-agent-field="summary_model"
            :class="{ 'setting-row--field-highlight': highlightedField === 'summary_model' }"
          >
            <div class="setting-info">
              <label>{{ 'Model' }} <span class="required">*</span></label>
              <p class="desc">{{ 'Select the LLM used by the agent' }}</p>
            </div>
            <div class="setting-control">
              <ModelSelector model-type="KnowledgeQA" :selected-model-id="formData.config.model_id"
                :all-models="allModels"
                @update:selected-model-id="(val: string) => formData.config.model_id = val"
                @add-model="handleAddModel('llm')" :placeholder="'Select Model'" />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Temperature' }}</label>
              <p class="desc">{{ 'Control output randomness, 0 is most deterministic, 1 is most random' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.temperature" :min="0" :max="1" :step="0.1" />
                <span class="slider-value">{{ formData.config.temperature }}</span>
              </div>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Max Completion Tokens' }}</label>
              <p class="desc">{{ isAgentMode ? 'Maximum tokens generated in each reasoning round, including tool-call JSON. Default is 4096 without a sandbox, or 24576 when a sandbox can write or edit files. A custom value is saved as entered and is not changed later.' :
                'Maximum tokens for the model reply. Default is 2048. Custom values are saved as entered.' }}
              </p>
            </div>
            <div class="setting-control max-tokens-control">
              <t-radio-group v-model="maxCompletionTokensMode">
                <t-radio-button value="default">{{ 'Default'
                  }}</t-radio-button>
                <t-radio-button value="custom">{{ 'Custom'
                  }}</t-radio-button>
              </t-radio-group>
              <span v-if="maxCompletionTokensMode === 'default'" class="max-tokens-value">
                {{ effectiveDefaultMaxCompletionTokens }}
              </span>
              <t-input-number v-else v-model="formData.config.max_completion_tokens" :min="100" :max="100000"
                :step="100" theme="column" />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Thinking Mode' }}</label>
              <p class="desc">{{ 'Enable extended thinking capability (requires model support)' }}</p>
              <p v-if="selectedChatModel && !selectedChatModelCanThink" class="desc">
                {{ 'The selected model cannot think; every option except \u0022Off\u0022 is ignored.' }}
              </p>
              <p v-else-if="selectedChatModelAlwaysThinks" class="desc">
                {{ 'The selected model always reasons and cannot be switched off; only the effort level can be changed.' }}
              </p>
            </div>
            <div class="setting-control">
              <t-select v-model="reasoningEffortLevel" class="reasoning-effort-select"
                :popup-props="{ overlayClassName: 'reasoning-level-select-popup' }">
                <t-option v-for="level in reasoningEffortOptions" :key="level" :value="level"
                  :label="levelLabel(level)" :show-overflow-tooltip="false">
                  <div class="reasoning-level-option">
                    <span class="reasoning-level-option__title">{{ levelLabel(level) }}</span>
                    <span class="reasoning-level-option__hint">{{ levelDescription(level) }}</span>
                  </div>
                </t-option>
              </t-select>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Show Source Citations' }}</label>
              <p class="desc">{{ 'Show knowledge-base and web sources in final answers; retrieval and grounding still work when disabled' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.citation_enabled" />
            </div>
          </div>

          <div
            v-if="showRerankModelField"
            class="setting-row"
            data-agent-field="rerank_model"
            :class="{ 'setting-row--field-highlight': highlightedField === 'rerank_model' }"
          >
            <div class="setting-info">
              <label>
                {{ 'ReRank Model' }}
                <span v-if="needsRerankModel" class="required">*</span>
              </label>
              <p class="desc">
                {{ 'Used to rerank knowledge base retrieval results for better accuracy' }}
                <template v-if="!needsRerankModel">
                  <br />
                  <span class="hint">{{ 'No RAG knowledge base in current scope, so this is optional. If a RAG knowledge base is added later, the workspace default rerank model will be used as a fallback. Configuring it explicitly is still recommended.' }}</span>
                </template>
              </p>
            </div>
            <div class="setting-control">
              <ModelSelector model-type="Rerank" :selected-model-id="formData.config.rerank_model_id"
                :all-models="allModels"
                :clearable="!needsRerankModel"
                @update:selected-model-id="(val: string) => formData.config.rerank_model_id = val"
                @add-model="handleAddModel('rerank')"
                :placeholder="'Select ReRank Model'" />
            </div>
          </div>

          <div
            v-if="!isAgentMode && formData.config.multi_turn_enabled && formData.config.enable_rewrite"
            class="setting-row">
            <div class="setting-info">
              <label>{{ 'Query Understand Model' }}</label>
              <p class="desc">{{ 'Model used for query understanding (rewriting and intent detection). Leave empty to reuse the main chat model.' }}</p>
            </div>
            <div class="setting-control">
              <ModelSelector model-type="KnowledgeQA"
                :selected-model-id="formData.config.query_understand_model_id" :all-models="allModels"
                clearable
                @update:selected-model-id="(val: string) => formData.config.query_understand_model_id = val"
                @add-model="handleAddModel('llm')"
                :placeholder="'Leave empty to reuse the main chat model'" />
            </div>
          </div>

          <div v-if="isAgentMode" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Max Iterations' }}</label>
              <p class="desc">{{ 'Caps how many reasoning steps one task may take. Unlimited keeps going until the model stops on its own or you stop it.' }}</p>
            </div>
            <div class="setting-control max-tokens-control">
              <t-radio-group v-model="maxIterationsMode">
                <t-radio-button value="limit">{{ 'Limit' }}</t-radio-button>
                <t-radio-button value="unlimited">{{ 'Unlimited' }}</t-radio-button>
              </t-radio-group>
              <t-input-number v-if="maxIterationsMode === 'limit'" v-model="formData.config.max_iterations"
                :min="2" :max="50" theme="column" />
            </div>
          </div>

          <div v-if="isAgentMode" class="setting-row">
            <div class="setting-info">
              <label>{{ 'LLM Call Timeout' }}</label>
              <p class="desc">{{ 'Maximum waiting time for a single LLM call (seconds). Call will be terminated if this time is exceeded' }}</p>
              <p class="desc-hint">{{ 'Leave empty or 0 to use the default (120 seconds)' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.llm_call_timeout" :min="0" :max="3600" theme="column"
                :placeholder="'Enter seconds, recommended range 60-1800'" clearable />
            </div>
          </div>

        </div>
      </div>

      <div v-show="currentSection === 'multimodal'" class="section">
        <div class="section-header">
          <h2>{{ 'Attachment Upload' }}</h2>
          <p class="section-description">{{ 'Configure image, document, and audio attachments in chat, plus parse rules and related models' }}</p>
        </div>

        <div class="settings-group">
          <div class="setting-row" data-guide="agent-create-multimodal">
            <div class="setting-info">
              <label>{{ 'Image Upload' }}</label>
              <p class="desc">{{ 'Allow users to upload images in chat for VLM understanding' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.image_upload_enabled" />
            </div>
          </div>

          <div v-if="formData.config.image_upload_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'VLM Model' }} <span class="required">*</span></label>
              <p class="desc">{{ 'Vision language model for image analysis' }}</p>
            </div>
            <div class="setting-control">
              <ModelSelector model-type="VLLM" :selected-model-id="formData.config.vlm_model_id"
                :all-models="allModels"
                @update:selected-model-id="(val: string) => formData.config.vlm_model_id = val"
                @add-model="handleAddModel('vllm')"
                :placeholder="'Select VLM model'" />
            </div>
          </div>

          <div v-if="formData.config.image_upload_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Attachment Image Understanding / Scanned OCR' }}</label>
              <p class="desc">{{ 'For image-only PDFs/PPTs (scanned documents), run VLM OCR when no text can be extracted. Increases parse latency; off by default.' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.attachment_image_understanding" />
            </div>
          </div>

          <div v-if="formData.config.image_upload_enabled && formData.config.attachment_image_understanding"
            class="setting-row">
            <div class="setting-info">
              <label>{{ 'Scanned OCR Max Pages' }}</label>
              <p class="desc">{{ 'Max pages of a scanned document sent to the VLM for OCR. More pages means better coverage but slower and costlier parsing. 0 uses the global default.' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.attachment_ocr_max_pages" :min="0" :max="64"
                :step="1" theme="normal" style="width: 160px;"
                :placeholder="'0 = global default'" />
            </div>
          </div>

          <div v-if="formData.config.image_upload_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Image Storage' }}</label>
              <p class="desc">{{ 'Storage engine for uploaded images. Leave empty to use system default' }}</p>
            </div>
            <div class="setting-control" style="flex-direction: column; align-items: flex-end;">
              <t-select v-model="formData.config.image_storage_provider" style="width: 280px;"
                :placeholder="'Select storage engine'" clearable>
                <t-option value="" :label="'System Default'" />
                <t-option v-for="opt in imageStorageOptions" :key="opt.value" :value="opt.value"
                  :label="opt.label" :disabled="opt.disabled">
                  <span class="select-option-with-tag">
                    <span>{{ opt.label }}</span>
                    <t-tag v-if="opt.disabled" theme="warning" variant="light" size="small">{{
                      'Not Configured' }}</t-tag>
                  </span>
                </t-option>
              </t-select>
              <a href="javascript:void(0)" class="go-settings-link"
                @click.prevent="uiStore.openSettings('storage')">
                {{ 'Go to Storage Settings' }}
              </a>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Audio Upload' }}</label>
              <p class="desc">{{ 'When enabled, users can upload audio files in conversations. The system will automatically transcribe them using the ASR model.' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.audio_upload_enabled" />
            </div>
          </div>

          <div v-if="formData.config.audio_upload_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'ASR Model' }}</label>
              <p class="desc">{{ 'Speech recognition model for audio transcription. If not configured, audio files will be passed as placeholders.' }}</p>
            </div>
            <div class="setting-control">
              <ModelSelector model-type="ASR" :selected-model-id="formData.config.asr_model_id"
                :all-models="allModels"
                clearable
                @update:selected-model-id="(val: string) => formData.config.asr_model_id = val"
                @add-model="handleAddModel('asr')"
                :placeholder="'Select ASR Model'" />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Attachment Parse Wait Timeout (s)' }}</label>
              <p class="desc">{{ 'How long a chat turn waits for still-parsing attachments before proceeding with only the finished ones. Raise for large / scanned files. 0 uses the global default.' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.attachment_parse_wait_timeout_sec" :min="0" :max="600"
                :step="10" theme="normal" style="width: 160px;"
                :placeholder="'0 = global default'" />
            </div>
          </div>

          <div class="parser-policy-block">
            <div class="parser-policy-block__header">
              <label>{{ 'Chat Attachment Parsing Policy' }}</label>
              <p class="desc">{{ 'Choose a parser engine per file type for this agent\'s chat attachments.' }}</p>
            </div>
            <KBParserSettings
              embedded
              :parser-engine-rules="formData.config.chat_parser_engine_rules"
              :relevant-extensions="CHAT_PARSER_EXTENSIONS"
              @update:parser-engine-rules="(val: any) => formData.config.chat_parser_engine_rules = val"
            />
          </div>

        </div>
      </div>

      <!-- Conversation. Both modes keep this group: in agent mode it explains that history
           is managed automatically by the context window, and it carries the cross-turn
           retrieval setting. The multi-turn switch is force-enabled by EnsureDefaults, so
           it is shown only in quick-answer mode. -->
      <div v-show="currentSection === 'conversation'" class="section">
        <div class="section-header">
          <h2>{{ 'Conversation' }}</h2>
          <p class="section-description">{{ conversationSectionDesc }}</p>
        </div>

        <div class="settings-group">
          <!-- Quick-answer mode only: in agent mode EnsureDefaults force-enables this, so a
               switch the user can turn off would simply be reset by the server. -->
          <div v-if="!isAgentMode" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Multi-turn Conversation' }}</label>
              <p class="desc">{{ 'When enabled, historical conversation context will be preserved' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.multi_turn_enabled" />
            </div>
          </div>

          <!-- Quick-answer mode only: agent mode loads history by context window and compresses
               the overflow into a summary (session_agent_qa.go -> LoadAgentHistory), so it never
               reads history_turns. -->
          <div v-if="!isAgentMode && formData.config.multi_turn_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'History Turns' }}</label>
              <p class="desc">{{ 'Number of recent conversation rounds to keep as context' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.history_turns" :min="1" :max="100" theme="column" />
            </div>
          </div>

          <!-- Only the agent path reads this (internal/agent/observe.go), and it only rewrites the
               history of the eight KB/Wiki tools, so it is hidden when there is no knowledge base. -->
          <div v-if="isAgentMode && hasKnowledgeBase" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Keep retrieval results' }}</label>
              <p class="desc">{{ 'Keep knowledge base results from earlier turns. When off, each turn searches again' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.retain_retrieval_history" />
            </div>
          </div>

          <div v-if="formData.config.multi_turn_enabled && !isAgentMode" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Query Rewrite' }}</label>
              <p class="desc">{{ 'Automatically rewrite user questions in multi-turn conversations to resolve references and omissions' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.enable_rewrite" />
            </div>
          </div>
        </div>
      </div>

      <div v-show="currentSection === 'suggestions'" class="section">
        <div class="section-header">
          <h2>{{ 'Conversation question suggestions' }}</h2>
          <p class="section-description">{{ 'Configure starters and contextual follow-ups in one agent-owned policy. Channels may hide them but cannot override the policy.' }}</p>
        </div>

        <t-tabs v-model="suggestionTab" class="suggestion-tabs">
          <t-tab-panel value="starters"
            :label="'Conversation starters'" />
          <t-tab-panel value="followUps"
            :label="'After-answer follow-ups'" />
        </t-tabs>

        <div v-show="suggestionTab === 'starters'" class="settings-group">
          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Show starter questions' }}</label>
              <p class="desc">{{ 'Shown before the first user message from curated, knowledge, or mixed sources.' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.question_suggestions.starters.enabled"
                :aria-label="'Show starter questions'" />
            </div>
          </div>

          <div v-if="formData.config.question_suggestions.starters.enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Source mode' }}</label>
            </div>
            <div class="setting-control">
              <t-select v-model="formData.config.question_suggestions.starters.mode"
                :options="starterSuggestionModeOptions" />
            </div>
          </div>

          <div v-if="formData.config.question_suggestions.starters.enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Question count' }}</label>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.question_suggestions.starters.count"
                :min="1" :max="8" theme="column" />
            </div>
          </div>

          <div
            v-if="formData.config.question_suggestions.starters.enabled && ['curated', 'hybrid'].includes(formData.config.question_suggestions.starters.mode)"
            class="setting-row setting-row-vertical">
            <div class="setting-info">
              <div class="setting-info-header setting-info-header--inline">
                <label>{{ 'Curated questions' }}</label>
                <span class="curated-items-count">
                  {{ formData.config.question_suggestions.starters.items.length }}/8
                </span>
              </div>
              <p class="desc">{{ 'Used for starters and prioritized in hybrid mode.' }}</p>
            </div>
            <div class="setting-control setting-control-full">
              <div class="suggested-prompts-list">
                <div v-for="(_prompt, index) in formData.config.question_suggestions.starters.items"
                  :key="index" class="prompt-item">
                  <t-input v-model="formData.config.question_suggestions.starters.items[index]"
                    :maxlength="200" />
                  <t-button variant="text" theme="danger" shape="square"
                    :aria-label="'Delete'" @click="removeStarterSuggestion(Number(index))">
                    <t-icon name="delete" />
                  </t-button>
                </div>
                <t-button variant="dashed"
                  :disabled="formData.config.question_suggestions.starters.items.length >= 8"
                  @click="addStarterSuggestion">
                  <template #icon><t-icon name="add" /></template>
                  {{ 'Add question' }}
                </t-button>
              </div>
            </div>
          </div>
        </div>

        <div v-show="suggestionTab === 'followUps'" class="settings-group">
          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Generate follow-up questions' }}</label>
              <p class="desc">{{ 'Generated asynchronously after every completed answer. Enabling this adds model usage.' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.question_suggestions.follow_ups.enabled"
                :aria-label="'Generate follow-up questions'" />
            </div>
          </div>

          <template v-if="formData.config.question_suggestions.follow_ups.enabled">
            <div class="setting-row">
              <div class="setting-info">
                <label>{{ 'Source mode' }}</label>
              </div>
              <div class="setting-control">
                <t-select v-model="formData.config.question_suggestions.follow_ups.mode"
                  :options="followUpSuggestionModeOptions" />
              </div>
            </div>

            <div class="setting-row">
              <div class="setting-info">
                <label>{{ 'Question count' }}</label>
              </div>
              <div class="setting-control">
                <t-input-number v-model="formData.config.question_suggestions.follow_ups.count"
                  :min="1" :max="5" theme="column" />
              </div>
            </div>

            <div v-if="formData.config.question_suggestions.follow_ups.mode !== 'knowledge'"
              class="setting-row">
              <div class="setting-info">
                <label>{{ 'Generation model' }}</label>
                <p class="desc">{{ 'Uses the model from the completed turn when empty.' }}</p>
              </div>
              <div class="setting-control">
                <ModelSelector model-type="KnowledgeQA"
                  :selected-model-id="formData.config.question_suggestions.follow_ups.model_id"
                  :all-models="allModels"
                  clearable
                  @update:selected-model-id="(val: string) => formData.config.question_suggestions.follow_ups.model_id = val"
                  @add-model="handleAddModel('summary')" />
              </div>
            </div>

            <div class="suggestion-advanced-divider">
              <span>{{ 'Advanced generation settings' }}</span>
            </div>

            <div class="setting-row">
              <div class="setting-info">
                <label>{{ 'Context turns' }}</label>
              </div>
              <div class="setting-control">
                <t-input-number
                  v-model="formData.config.question_suggestions.follow_ups.max_context_turns"
                  :min="1" :max="5" theme="column" />
              </div>
            </div>

            <div class="setting-row setting-row-vertical">
              <div class="setting-info">
                <label>{{ 'Question types' }}</label>
              </div>
              <div class="setting-control setting-control-full">
                <t-checkbox-group v-model="formData.config.question_suggestions.follow_ups.categories"
                  :options="followUpCategoryOptions" />
              </div>
            </div>

            <div class="setting-row setting-row-vertical">
              <div class="setting-info">
                <label>{{ 'Additional instruction' }}</label>
              </div>
              <div class="setting-control setting-control-full">
                <t-textarea
                  v-model="formData.config.question_suggestions.follow_ups.additional_instruction"
                  :placeholder="'For example: prioritize practical next-step questions and avoid broad questions or repeating the answer…'"
                  :maxlength="2000" :autosize="{ minRows: 3, maxRows: 8 }" />
              </div>
            </div>

            <div class="setting-row setting-row-vertical">
              <div class="setting-info">
                <label>{{ 'Display and fallback rules' }}</label>
              </div>
              <div class="setting-control setting-control-full">
                <div class="suggestion-checkboxes">
                  <t-checkbox v-model="formData.config.question_suggestions.follow_ups.suppress_on_fallback">{{ 'Hide after fallback answers' }}</t-checkbox>
                  <t-checkbox v-model="formData.config.question_suggestions.follow_ups.suppress_when_answer_asks_question">{{ 'Hide when the answer ends with a question' }}</t-checkbox>
                  <t-checkbox v-model="formData.config.question_suggestions.follow_ups.knowledge_fallback">{{ 'Use knowledge candidates if generation fails' }}</t-checkbox>
                  <t-checkbox v-model="formData.config.question_suggestions.follow_ups.allow_regenerate">{{ 'Allow users to regenerate' }}</t-checkbox>
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>

      <div v-show="currentSection === 'tools' && isAgentMode" class="section">
        <div class="section-header">
          <h2>{{ 'Tools' }}</h2>
          <p class="section-description">{{ 'Configure tools available to the Agent' }}</p>
        </div>

        <div class="tools-overview">
          <div class="tools-overview-row">
            <div class="tools-status-chip">
              <t-icon name="folder" />
              <template v-if="kbSelectionMode === 'none'">
                <span>{{ 'No knowledge base is linked' }}</span>
              </template>
              <template v-else>
                <span class="tools-status-metric">
                  <strong>{{ ragKbCount }}</strong> {{ 'RAG KBs' }}
                </span>
                <span class="tools-status-sep">·</span>
                <span class="tools-status-metric">
                  <strong>{{ wikiKbCount }}</strong> {{ 'Wiki KBs' }}
                </span>
              </template>
            </div>
            <div v-if="inactiveToolCount > 0" class="tools-status-chip tools-status-chip--warn">
              <t-icon name="error-circle" />
              <span>{{ `${inactiveToolCount} ticked tool(s) cannot take effect with the current config` }}</span>
            </div>
          </div>
        </div>

        <div class="settings-group">
          <div
            class="setting-row setting-row-vertical"
            data-agent-field="allowed_tools"
            :class="{ 'setting-row--field-highlight': highlightedField === 'allowed_tools' }"
          >
            <div class="setting-info">
              <label>{{ 'Allowed Tools' }}</label>
              <p class="desc">{{ 'Select tools available to the Agent' }}</p>
            </div>
            <div class="setting-control setting-control-full">
              <t-checkbox-group v-model="formData.config.allowed_tools" class="tool-groups">
                <section v-for="group in groupedAvailableTools" :key="group.key"
                  :class="['tool-group', `tool-group--${group.key}`]">
                  <header class="tool-group-header">
                    <span class="tool-group-bar" />
                    <span class="tool-group-title">{{ group.label }}</span>
                    <span class="tool-group-count">{{ group.tools.length }}</span>
                    <span v-if="group.key === 'wiki_edit'" class="tool-group-warning">
                      <t-icon name="error-circle" />
                      {{ 'Mutates Wiki content' }}
                    </span>
                  </header>
                  <div class="tool-grid">
                    <t-checkbox v-for="tool in group.tools" :key="tool.value" :value="tool.value"
                      :disabled="tool.disabled"
                      :class="['tool-card', { 'tool-card--disabled': tool.disabled, 'tool-card--danger': tool.danger }]">
                      <div class="tool-card-body">
                        <div class="tool-card-head">
                          <span class="tool-card-name">{{ tool.label }}</span>
                          <span v-if="tool.danger" class="tool-card-badge">
                            {{ 'Write' }}
                          </span>
                        </div>
                        <span v-if="tool.description" class="tool-card-desc">{{ tool.description }}</span>
                        <span v-if="tool.disabled && tool.disabledReason" class="tool-card-hint">
                          {{ tool.disabledReason }}
                        </span>
                      </div>
                    </t-checkbox>
                  </div>
                </section>
              </t-checkbox-group>
            </div>
          </div>

          <div class="setting-row setting-row-vertical">
            <div class="setting-info">
              <label>{{ 'Effective Tools' }}</label>
              <p class="desc">{{ 'Computed from the current config — these are the tools the agent will actually be able to call' }}</p>
            </div>
            <div class="setting-control setting-control-full">
              <div class="effective-tools">
                <template v-if="effectiveTools.length === 0">
                  <div class="effective-tools-empty">
                    {{ 'No tool available — the agent will fall back to plain model chat' }}
                  </div>
                </template>
                <template v-else>
                  <span v-for="item in effectiveTools" :key="item.value"
                    :class="['effective-chip', { 'effective-chip--inactive': !item.active }]"
                    :title="item.reason || ''">
                    <span class="effective-chip-label">{{ item.label }}</span>
                    <span v-if="!item.active" class="effective-chip-reason">{{ item.reason }}</span>
                  </span>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-show="currentSection === 'mcp' && isAgentMode" class="section">
        <div class="section-header">
          <h2>{{ 'MCP Services' }}</h2>
          <p class="section-description">{{ 'Select MCP services available to the Agent' }}</p>
        </div>

        <div class="settings-group">
          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'MCP Services' }}</label>
              <p class="desc">{{ 'Select MCP services available to the Agent' }}</p>
            </div>
            <div class="setting-control">
              <t-radio-group v-model="mcpSelectionMode">
                <t-radio-button value="all">{{ 'All' }}</t-radio-button>
                <t-radio-button value="selected">{{ 'Selected' }}</t-radio-button>
                <t-radio-button value="none">{{ 'Disabled' }}</t-radio-button>
              </t-radio-group>
            </div>
          </div>

          <div v-if="mcpSelectionMode === 'selected' && showMcpServiceSelect" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Select MCP Services' }}</label>
              <p class="desc">{{ 'Select MCP services to enable' }}</p>
            </div>
            <div class="setting-control">
              <t-select v-model="formData.config.mcp_services" multiple
                :placeholder="'Select MCP services'" filterable>
                <t-option v-for="mcp in mcpOptions" :key="mcp.value" :value="mcp.value" :label="mcp.label"
                  :disabled="mcp.disabled" />
              </t-select>
            </div>
          </div>

          <div v-if="mcpSelectionMode !== 'none'" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Authentication Wait Timeout (s)' }}</label>
              <p class="desc">{{ 'Maximum seconds to wait for you to complete OAuth authentication when prompted during a conversation; the prompt is skipped after it elapses (only affects OAuth MCP services).' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.mcp_auth_wait_timeout" :min="5" :max="3600"
                theme="column" :placeholder="'Default 600 seconds'" />
            </div>
          </div>
        </div>
      </div>

      <div v-show="currentSection === 'skills' && isAgentMode" class="section">
        <div class="section-header">
          <h2>{{ 'Skills' }}</h2>
          <p class="section-description">{{ 'Select a running sandbox, then pick skills below. Skills not on that sandbox show Install and can only be checked after they are installed.' }}</p>
        </div>

        <div class="settings-group">
          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Sandbox' }}</label>
              <p class="desc">{{ 'Skill scripts do not run until a sandbox is selected.' }}</p>
            </div>
            <div class="setting-control sandbox-select-control">
              <t-select
                v-model="formData.config.sandbox_config_id"
                :placeholder="'Disabled'"
                class="sandbox-config-select"
                filterable
                :popup-props="{ overlayClassName: 'sandbox-config-select-popup' }"
              >
                <t-option value="" :label="'Disabled'" />
                <t-option
                  v-for="cfg in sandboxConfigOptions"
                  :key="cfg.id"
                  :value="cfg.id"
                  :label="cfg.name"
                >
                  <div class="sandbox-option">
                    <div class="sandbox-option__row">
                      <span class="sandbox-option__name">{{ cfg.name }}</span>
                      <span v-if="cfg.sandbox_type" class="sandbox-option__type">{{ backendLabel(cfg.sandbox_type) }}</span>
                    </div>
                    <div v-if="sandboxTargetLine(cfg)" class="sandbox-option__target">{{ sandboxTargetLine(cfg) }}</div>
                  </div>
                </t-option>
              </t-select>
              <p v-if="selectedSandboxSummary" class="sandbox-selected-meta">{{ selectedSandboxSummary }}</p>
              <div class="sandbox-select-links">
                <a href="javascript:void(0)" class="go-settings-link"
                  @click.prevent="uiStore.openSettings('sandbox')">
                  {{ 'Manage sandboxes' }}
                </a>
                <template v-if="hasSandboxSelected && canInstallSkills">
                  <span class="sandbox-select-links__sep" aria-hidden="true">·</span>
                  <a
                    href="javascript:void(0)"
                    class="go-settings-link"
                    @click.prevent="openSkillSettings"
                  >
                    {{ 'Manage skills' }}
                  </a>
                </template>
              </div>
              <p v-if="sandboxConfigOptions.length === 0" class="desc empty-hint">
                {{ 'This workspace has no sandbox yet, so skill scripts will not run.' }}
              </p>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Skill list' }}</label>
              <p class="desc">{{ skillsSelectionHint }}</p>
            </div>
            <div class="setting-control sandbox-select-control">
              <t-radio-group v-model="skillsSelectionMode">
                <t-radio-button value="all" :disabled="!canEnableSkills">{{ 'All' }}</t-radio-button>
                <t-radio-button value="selected" :disabled="!canEnableSkills">{{ 'Selected' }}</t-radio-button>
                <t-radio-button value="none">{{ 'Disabled' }}</t-radio-button>
              </t-radio-group>
              <p v-if="!hasSandboxSelected && sandboxConfigOptions.length > 1" class="desc empty-hint">
                {{ 'Select a sandbox first.' }}
              </p>
              <p v-else-if="hasSandboxSelected && skillCatalog.length === 0" class="desc empty-hint">
                <span>{{ 'The workspace catalog has no skills yet.' }}</span>
                <a
                  v-if="canInstallSkills"
                  href="javascript:void(0)"
                  class="go-settings-link"
                  @click.prevent="openSkillSettings"
                >
                  {{ 'Manage skills' }}
                </a>
              </p>
            </div>
          </div>

          <div v-if="showCatalogSkillList" class="setting-row setting-row-vertical">
            <div class="setting-control setting-control-full">
              <t-checkbox-group
                v-model="formData.config.selected_skills"
                class="skill-pick-list"
              >
                <section
                  v-for="group in catalogSkillGroups"
                  :key="group.key"
                  class="skill-pick-group"
                  :class="`skill-pick-group--${group.key}`"
                >
                  <header class="skill-pick-group__header">
                    <span class="skill-pick-group__bar" />
                    <span class="skill-pick-group__title">{{ group.label }}</span>
                    <span class="skill-pick-group__count">{{ group.skills.length }}</span>
                  </header>
                  <article
                    v-for="skill in group.skills"
                    :key="skill.name"
                    class="skill-pick"
                    :class="{
                      'skill-pick--ready': skill.selectable,
                      'skill-pick--pending': !skill.selectable,
                      'skill-pick--busy': isSkillBusy(skill),
                    }"
                  >
                    <t-checkbox
                      v-if="skillsSelectionMode === 'selected'"
                      :value="skill.name"
                      :disabled="!skill.selectable"
                      class="skill-pick__check"
                    />
                    <div class="skill-pick__badge" aria-hidden="true">
                      <t-icon :name="SKILL_ICON" size="16px" />
                    </div>
                    <div class="skill-pick__body">
                      <div class="skill-pick__title-row">
                        <span class="skill-name" :title="skill.name">{{ skill.name }}</span>
                        <span
                          v-if="!skill.selectable"
                          class="skill-pick__hint"
                          :class="{ 'skill-pick__hint--busy': isSkillBusy(skill) }"
                        >
                          <t-icon :name="skillStatusIcon(skill)" size="14px" />
                          {{ skillStatusHint(skill) }}
                        </span>
                        <span
                          v-if="skill.selectable && skill.servedNote"
                          class="skill-pick__hint"
                          :class="{ 'skill-pick__hint--busy': isSkillBusy(skill) }"
                        >
                          <t-icon :name="isSkillBusy(skill) ? 'refresh' : 'error-circle'" size="14px" />
                          {{ skill.servedNote }}
                        </span>
                        <span
                          v-if="canUpgradeSkillRow(skill)"
                          class="skill-pick__hint skill-pick__hint--upgrade"
                        >
                          <t-icon name="arrow-up" size="14px" />
                          {{ skillUpgradeHint(skill) }}
                        </span>
                      </div>
                      <p
                        v-if="skill.description"
                        class="skill-desc"
                        :title="skill.description"
                      >{{ skill.description }}</p>
                    </div>
                    <t-button
                      v-if="canInstallSkillRow(skill)"
                      size="small"
                      variant="text"
                      theme="primary"
                      :loading="installingCatalogId === skill.id"
                      :title="installsAnUpgrade(skill) ? 'Upgrade this sandbox to the catalog version' : 'Install onto this sandbox'"
                      @click.stop="installCatalogToCurrent(skill)"
                    >
                      {{ installsAnUpgrade(skill) ? 'Upgrade' : 'Install' }}
                    </t-button>
                    <t-button
                      v-else-if="canUpgradeSkillRow(skill)"
                      size="small"
                      variant="text"
                      theme="primary"
                      :loading="installingCatalogId === skill.id"
                      :title="'Upgrade this sandbox to the catalog version'"
                      @click.stop="installCatalogToCurrent(skill)"
                    >
                      {{ 'Upgrade' }}
                    </t-button>
                    <t-button
                      v-else-if="canInstallSkills && isSkillBusy(skill)"
                      size="small"
                      variant="text"
                      theme="primary"
                      :title="'View progress'"
                      @click.stop="openSkillInstallProgress(skill)"
                    >
                      {{ 'View progress' }}
                    </t-button>
                  </article>
                </section>
              </t-checkbox-group>
            </div>
          </div>
        </div>
      </div>

      <div v-show="currentSection === 'knowledge'" class="section">
        <div class="section-header">
          <h2>{{ 'Knowledge Base' }}</h2>
          <p class="section-description">{{ 'Configure knowledge base scope and FAQ strategy for the agent' }}</p>
        </div>

        <div class="settings-group">
          <div class="setting-row" data-guide="agent-create-knowledge">
            <div class="setting-info">
              <label>{{ 'Knowledge Bases' }}</label>
              <p class="desc">{{ 'Select the scope of knowledge bases accessible to the agent' }}</p>
            </div>
            <div class="setting-control">
              <t-radio-group v-model="kbSelectionMode">
                <t-radio-button value="all">{{ 'All Knowledge Bases' }}</t-radio-button>
                <t-radio-button value="selected">{{ 'Selected Knowledge Bases'
                  }}</t-radio-button>
                <t-radio-button value="none">{{ 'No Knowledge Base' }}</t-radio-button>
              </t-radio-group>
            </div>
          </div>

          <div v-if="kbSelectionMode === 'selected'" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Select Knowledge Bases' }}</label>
              <p class="desc">{{ 'Select knowledge bases to associate (including collaborative ones)' }}</p>
            </div>
            <div class="setting-control">
              <t-select v-model="formData.config.knowledge_bases" multiple
                :placeholder="'Select Knowledge Bases'" filterable :min-collapsed-num="3">
                <t-option-group v-if="filteredMyKbOptions.length"
                  :label="'My Knowledge Bases'">
                  <t-option v-for="kb in filteredMyKbOptions" :key="kb.value" :value="kb.value"
                    :label="kb.label" :disabled="kb.disabled">
                    <div class="kb-option-item" :title="kb.disabled ? kb.disabledReason : ''">
                      <span class="kb-option-icon" :class="kb.type === 'faq' ? 'faq-icon' : 'doc-icon'">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" />
                      </span>
                      <span class="kb-option-label">{{ kb.label }}</span>
                      <span v-if="kb.ragEnabled" class="kb-option-tag tag-rag">RAG</span>
                      <span v-if="kb.wikiEnabled" class="kb-option-tag tag-wiki">Wiki</span>
                      <span class="kb-option-count">{{ kb.count || 0 }}</span>
                      <span v-if="kb.disabled" class="kb-option-disabled-hint">{{ kb.disabledReason }}</span>
                    </div>
                  </t-option>
                </t-option-group>
                <t-option-group v-if="filteredSharedKbOptions.length"
                  :label="'Collaborative Knowledge Bases'">
                  <t-option v-for="kb in filteredSharedKbOptions" :key="kb.value" :value="kb.value"
                    :label="kb.label" :disabled="kb.disabled">
                    <div class="kb-option-item" :title="kb.disabled ? kb.disabledReason : ''">
                      <span class="kb-option-icon" :class="kb.type === 'faq' ? 'faq-icon' : 'doc-icon'">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" />
                      </span>
                      <span class="kb-option-label">{{ kb.label }}</span>
                      <span v-if="kb.ragEnabled" class="kb-option-tag tag-rag">RAG</span>
                      <span v-if="kb.wikiEnabled" class="kb-option-tag tag-wiki">Wiki</span>
                      <span v-if="kb.orgName" class="kb-option-org">{{ kb.orgName }}</span>
                      <span class="kb-option-count">{{ kb.count || 0 }}</span>
                      <span v-if="kb.disabled" class="kb-option-disabled-hint">{{ kb.disabledReason }}</span>
                    </div>
                  </t-option>
                </t-option-group>
              </t-select>
            </div>
          </div>

          <div v-if="hasKnowledgeBase" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Supported File Types' }}</label>
              <p class="desc">{{ 'Restrict selectable file types, leave empty to support all types' }}</p>
            </div>
            <div class="setting-control">
              <t-select v-model="formData.config.supported_file_types" multiple
                :placeholder="'All Types'" :min-collapsed-num="3" clearable>
                <t-option v-for="ft in availableFileTypes" :key="ft.value" :value="ft.value"
                  :label="ft.label" />
              </t-select>
            </div>
          </div>

          <div v-if="hasKnowledgeBase" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Retrieve Only When Mentioned' }}</label>
              <p class="desc">{{ 'Off: auto-retrieve configured KBs; On: retrieve only when user {\'@\'} mentions' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.retrieve_kb_only_when_mentioned" />
            </div>
          </div>

        </div>
      </div>

      <div v-show="currentSection === 'websearch'" class="section">
        <div class="section-header">
          <h2>{{ 'Web Search' }}</h2>
          <p class="section-description">{{ 'Configure web search capabilities for the agent' }}</p>
        </div>

        <div class="settings-group">
          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Web Search' }}</label>
              <p class="desc">{{ 'When enabled, the agent can search the internet for information' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.web_search_enabled" />
            </div>
          </div>

          <div v-if="formData.config.web_search_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Search Engine' }}</label>
              <p class="desc">{{ 'Specify a search engine for this agent. Leave empty to use the default.' }}</p>
            </div>
            <div class="setting-control">
              <t-select v-model="formData.config.web_search_provider_id" clearable
                :placeholder="'Use default search engine'" style="width: 240px;">
                <t-option v-for="p in webSearchProviderList" :key="p.id" :value="p.id" :label="p.name">
                  <span>{{ p.name }}</span>
                  <t-tag v-if="p.is_default" theme="primary" size="small" style="margin-left: 6px;">{{
                    'Default'
                    }}</t-tag>
                </t-option>
              </t-select>
            </div>
          </div>

          <div v-if="formData.config.web_search_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Max Search Results' }}</label>
              <p class="desc">{{ 'Maximum number of results returned per search' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.web_search_max_results" :min="1" :max="10" />
                <span class="slider-value">{{ formData.config.web_search_max_results }}</span>
              </div>
            </div>
          </div>

          <div v-if="formData.config.web_search_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Auto-Fetch Page Content' }}</label>
              <p class="desc">{{ 'After reranking, auto-fetch full page content from top web results for better answers' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.web_fetch_enabled" />
            </div>
          </div>

          <div v-if="formData.config.web_search_enabled && formData.config.web_fetch_enabled"
            class="setting-row">
            <div class="setting-info">
              <label>{{ 'Pages to Fetch' }}</label>
              <p class="desc">{{ 'Maximum number of web pages to fetch after reranking' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.web_fetch_top_n" :min="1" :max="10" />
                <span class="slider-value">{{ formData.config.web_fetch_top_n }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-show="currentSection === 'retrieval' && hasKnowledgeBase" class="section">
        <div class="section-header">
          <h2>{{ 'Retrieval Strategy' }}</h2>
          <p class="section-description">{{ 'Configure knowledge base retrieval, ranking, and FAQ priority strategy' }}</p>
        </div>

        <div class="settings-group">
          <div v-if="!isAgentMode" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Query Expansion' }}</label>
              <p class="desc">{{ 'Automatically expand query terms to improve recall' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.enable_query_expansion" />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Embedding Top K' }}</label>
              <p class="desc">{{ 'Maximum number of results from vector retrieval' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.embedding_top_k" :min="1" :max="50" theme="column" />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Keyword Threshold' }}</label>
              <p class="desc">{{ 'Minimum relevance score for keyword retrieval' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.keyword_threshold" :min="0" :max="1" :step="0.01" />
                <span class="slider-value">{{ formData.config.keyword_threshold?.toFixed(2) }}</span>
              </div>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <label>{{ 'Vector Threshold' }}</label>
              <p class="desc">{{ 'Minimum similarity score for vector retrieval' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.vector_threshold" :min="0" :max="1" :step="0.01" />
                <span class="slider-value">{{ formData.config.vector_threshold?.toFixed(2) }}</span>
              </div>
            </div>
          </div>

          <div v-if="formData.config.rerank_model_id" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Rerank Top K' }}</label>
              <p class="desc">{{ 'Maximum number of results retained after reranking' }}</p>
            </div>
            <div class="setting-control">
              <t-input-number v-model="formData.config.rerank_top_k" :min="1" :max="20" theme="column" />
            </div>
          </div>

          <div v-if="formData.config.rerank_model_id" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Rerank Threshold' }}</label>
              <p class="desc">{{ 'Minimum relevance score for reranking' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.rerank_threshold" :min="-10" :max="10" :step="0.01" />
                <span class="slider-value">{{ formData.config.rerank_threshold?.toFixed(1) }}</span>
              </div>
            </div>
          </div>

          <div v-if="hasFaqKnowledgeBase" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Enable FAQ Priority' }}</label>
              <p class="desc">{{ 'FAQ answers will be prioritized over regular documents, improving response accuracy' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.faq_priority_enabled" />
            </div>
          </div>

          <div v-if="hasFaqKnowledgeBase && formData.config.faq_priority_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Direct Answer Threshold' }}</label>
              <p class="desc">{{ 'When the similarity between the question and FAQ exceeds this value, use the FAQ answer directly' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.faq_direct_answer_threshold" :min="0.7" :max="1"
                  :step="0.05" />
                <span class="slider-value">{{ formData.config.faq_direct_answer_threshold?.toFixed(2)
                  }}</span>
              </div>
            </div>
          </div>

          <div v-if="hasFaqKnowledgeBase && formData.config.faq_priority_enabled" class="setting-row">
            <div class="setting-info">
              <label>{{ 'FAQ Score Boost' }}</label>
              <p class="desc">{{ 'Multiply FAQ relevance scores by this factor to rank them higher' }}</p>
            </div>
            <div class="setting-control">
              <div class="slider-wrapper">
                <t-slider v-model="formData.config.faq_score_boost" :min="1" :max="2" :step="0.1" />
                <span class="slider-value">{{ formData.config.faq_score_boost?.toFixed(1) }}x</span>
              </div>
            </div>
          </div>

          <div v-if="!isAgentMode" class="setting-row">
            <div class="setting-info">
              <label>{{ 'Enable Tabular Data Analysis' }}</label>
              <p class="desc">{{ 'When the retrieved chunks come from a CSV/Excel file, ask the LLM to generate a DuckDB SQL query before answering. This adds one extra LLM call and several seconds of latency, so only enable it when you actually need SQL-style analysis.' }}</p>
            </div>
            <div class="setting-control">
              <t-switch v-model="formData.config.data_analysis_enabled" />
            </div>
          </div>
        </div>
      </div>

      <div v-if="editorMode === 'edit' && editorAgent?.id && !editorAgent?.is_builtin"
        v-show="currentSection === 'share'" class="section">
        <AgentShareSettings :agent-id="editorAgent.id" :agent="editorAgent" />
      </div>
    </div>

    <template #footer-note>
      <p v-if="isPostCreateSession" class="settings-footer-note">
        <t-icon name="check-circle-filled" class="settings-footer-note__icon" />
        <span>
          <strong>{{ 'Created successfully' }}</strong>
          {{ 'Keep adjusting settings, configure sharing and publishing, then click \u0022Save and Close\u0022.' }}
        </span>
      </p>
    </template>
    <template #footer>
      <t-button variant="outline" @click="modalShell.requestClose">
        {{ props.readOnly ? 'Close' : 'Cancel' }}
      </t-button>
      <t-button v-if="!props.readOnly" theme="primary" data-guide="agent-create-submit" :loading="saving"
        :disabled="editorInitializing" @click="handleSave">
        {{ saveButtonLabel }}
      </t-button>
    </template>
  </SettingsModalShell>

  <SettingDrawer
    v-model:visible="showSkillProgress"
    :title="skillProgressTitle"
    :description="skillProgressDesc"
    :icon="SKILL_ICON"
    width="680px"
    :min-width="560"
    :max-width="920"
    storage-key="setting-drawer:width:skill-catalog-manage"
    :hide-footer="true"
  >
    <SandboxSkillsPanel
      v-if="showSkillProgress && skillProgressRecord && skillProgressId"
      :record="skillProgressRecord"
      mode="list"
      hide-add
      :focus-skill-id="skillProgressId"
      @updated="onSkillProgressUpdated"
      @skills-changed="onSkillProgressChanged"
    />
  </SettingDrawer>

  <AgentCreateContextualGuide :when="visible && editorMode === 'create'" :is-agent-mode="isAgentMode" />
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue';
import { useRouter } from 'vue-router';
import AgentCreateContextualGuide from '@/components/AgentCreateContextualGuide.vue';
import {
  AGENT_EDITOR_FOCUS_SECTION_EVENT,
  markContextualGuideDone,
} from '@/config/contextualGuides';
import { selectInitialModelId } from '@/utils/modelDefaults';
import { hydrateAgentPromptRefs, serializeAgentPrompts } from '@/utils/agentPromptTemplates';
import { copyWithToast } from '@/utils/clipboard';
import { MessagePlugin } from 'tdesign-vue-next';
import { useModalShell } from '@/composables/useModalShell'
import SettingsModalShell from '@/components/SettingsModalShell.vue'
import {
  createAgent,
  updateAgent,
  listIMChannels,
  type CustomAgent,
  type PlaceholderDefinition,
  type AgentTypePreset,
  type AgentType,
  type AgentTypeKBFilter,
  type KBCapabilities,
} from '@/api/agent';
import { type ModelConfig } from '@/api/model';
import { type AgentNotReadyReasonKey, agentRequiresRerankModel } from '@/utils/agent-readiness';
import { normalizeLegacyToolNames } from '@/utils/legacy-tool-names';
import { installSkillCatalog, type SkillCatalogItem } from '@/api/skill';
import { installUpgradable, servedPreviousText, upgradeVersions } from '@/utils/skillUpgrade';
import { type WebSearchProviderEntity } from '@/api/web-search-provider';
import {
  isNamedSandboxBackend,
  type SandboxConfigRecord,
  type StorageEngineStatusItem,
  type PromptTemplate,
  type PromptTemplatesConfig,
} from '@/api/system';
import { useUIStore } from '@/stores/ui';
import { useAuthStore } from '@/stores/auth';
import { useOrganizationStore } from '@/stores/organization';
import { useChatResourcesStore } from '@/stores/chatResources';
import { useEditorResourcesStore } from '@/stores/editorResources';
import AgentAvatar from '@/components/AgentAvatar.vue';
import PromptTemplateSelector from '@/components/PromptTemplateSelector.vue';
import ModelSelector from '@/components/ModelSelector.vue';
import SandboxSkillsPanel from '@/components/SandboxSkillsPanel.vue';
import SettingDrawer from '@/components/settings/SettingDrawer.vue';
import KBParserSettings, { type ParserEngineRule } from '@/views/knowledge/settings/KBParserSettings.vue';
import AgentShareSettings from '@/components/AgentShareSettings.vue';
import { SKILL_ICON } from '@/types/mention';
import { listEmbedChannels } from '@/api/embed';
import { getRootZoom, rectToCssPx } from '@/utils/zoom';
import { integrationSectionKey } from '@/config/settingsRoute';
import {
  evaluateToolRequirement,
  deriveKbFilterFromTools,
  type RequirementMissKind,
  type ScopeCapabilities,
} from '@/utils/tool-capabilities';
import {
  clampLevel,
  levelDescription,
  levelEnablesThinking,
  levelFromLegacy,
  levelLabel,
  modelCanThink,
  modelCannotDisableThinking,
  optionsFor,
  type ReasoningLevel,
} from '@/utils/reasoningEffort';

const SETTINGS_SANDBOX_BACKENDS_LABELS: Record<string, string> = {
  disabled: 'Disabled',
  local: 'Local process',
  docker: 'Docker',
  cube: 'CubeSandbox',
  e2b: 'E2B',
}

const AGENT_EDITOR_AGENT_TYPE_KB_MISMATCH_LABELS: Record<string, string> = {
  ragQa: 'RAG retrieval not enabled',
  wikiQa: 'Wiki not enabled',
  hybridRagWiki: 'No retrieval surface enabled',
  dataAnalysis: 'Requires RAG (FAQ not supported)',
  quickAnswer: 'Quick Answer mode requires RAG retrieval',
  generic: 'Not compatible with current type',
}

// File extensions offered in the agent-level chat attachment parsing policy.
const CHAT_PARSER_EXTENSIONS = [
  'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'epub', 'mhtml',
  'txt', 'md', 'markdown', 'csv', 'json', 'xml', 'html', 'yaml', 'yml', 'log',
  'jpg', 'jpeg', 'png', 'gif', 'bmp', 'tiff', 'webp',
];

const uiStore = useUIStore();
const authStore = useAuthStore();
const router = useRouter();
const orgStore = useOrganizationStore();
const chatResources = useChatResourcesStore();
const editorResources = useEditorResourcesStore();


const props = defineProps<{
  visible: boolean;
  mode: 'create' | 'edit';
  agent?: CustomAgent | null;
  initialSection?: string;
  initialHighlightField?: string;
  // readOnly hides the save button so a Viewer who clicks an agent
  // card to inspect its config doesn't see a save button that 403s on the
  // backend update endpoint. Field-level disable is intentionally NOT
  // wired here yet (the modal has 3000+ lines of form inputs); instead
  // we just remove the only mutation surface — the footer button.
  readOnly?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:visible', visible: boolean): void;
  (e: 'success', agent?: CustomAgent): void;
}>();

/** After the first successful save, stay in the modal and switch to edit mode locally so the IM / embed entries show up */
const savedAgent = ref<CustomAgent | null>(null);
const editorMode = computed(() => (savedAgent.value ? 'edit' : props.mode));
const editorAgent = computed(() => savedAgent.value ?? props.agent ?? null);
const isPostCreateSession = computed(() => !!savedAgent.value);
const saveButtonLabel = computed(() =>
  editorMode.value === 'create'
    ? 'Create Agent'
    : 'Save and Close'
);

const copyAgentId = async () => {
  await copyWithToast(editorAgent.value?.id, 'Copied');
};

// The legacy entry point had a separate sandbox tab; after the merge, section=sandbox is still accepted.
const AGENT_EDITOR_SECTION_ALIASES: Record<string, string> = {
  sandbox: 'skills',
};

function resolveEditorSection(section?: string | null): string {
  const key = section || 'basic';
  return AGENT_EDITOR_SECTION_ALIASES[key] || key;
}

const currentSection = ref(resolveEditorSection(props.initialSection));
const suggestionTab = ref<'starters' | 'followUps'>('starters');
const contentWrapperRef = ref<HTMLElement | null>(null);
const highlightedField = ref<AgentNotReadyReasonKey | null>(null);
let highlightClearTimer: ReturnType<typeof setTimeout> | null = null;

const VALID_HIGHLIGHT_FIELDS: AgentNotReadyReasonKey[] = ['summary_model', 'rerank_model', 'allowed_tools'];

const sectionForHighlightField = (field: AgentNotReadyReasonKey): string => {
  if (field === 'allowed_tools') return 'tools';
  return 'model';
};

const FIELD_FLASH_DURATION_MS = 2400;

const clearFieldHighlight = () => {
  if (highlightClearTimer) {
    clearTimeout(highlightClearTimer);
    highlightClearTimer = null;
  }
  highlightedField.value = null;
};

const applyInitialFieldHighlight = async (field: string) => {
  if (!VALID_HIGHLIGHT_FIELDS.includes(field as AgentNotReadyReasonKey)) return;

  const targetField = field as AgentNotReadyReasonKey;
  currentSection.value = sectionForHighlightField(targetField);

  await nextTick();
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
  });

  clearFieldHighlight();
  highlightedField.value = null;

  const wrapper = contentWrapperRef.value;
  const row = wrapper?.querySelector(`[data-agent-field="${targetField}"]`) as HTMLElement | null;
  if (row && wrapper) {
    const rowTop = row.offsetTop;
    const scrollTarget = rowTop - wrapper.clientHeight / 2 + row.clientHeight / 2;
    wrapper.scrollTo({ top: Math.max(0, scrollTarget), behavior: 'auto' });

    await nextTick();
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });
  }

  highlightedField.value = targetField;

  if (row) {
    const focusTarget = row.querySelector('.t-input, .t-select-input, input, .t-checkbox') as HTMLElement | null;
    focusTarget?.focus({ preventScroll: true });
  }

  highlightClearTimer = setTimeout(() => {
    if (highlightedField.value === targetField) {
      highlightedField.value = null;
    }
    highlightClearTimer = null;
  }, FIELD_FLASH_DURATION_MS);
};

const onAgentEditorFocusSection = (event: Event) => {
  const section = (event as CustomEvent<{ section?: string }>).detail?.section
  if (!section) return
  const resolved = resolveEditorSection(section)
  if (navItems.value.some((item) => item.key === resolved)) {
    currentSection.value = resolved
  }
}

onMounted(() => {
  window.addEventListener(AGENT_EDITOR_FOCUS_SECTION_EVENT, onAgentEditorFocusSection)
})

onBeforeUnmount(() => {
  window.removeEventListener(AGENT_EDITOR_FOCUS_SECTION_EVENT, onAgentEditorFocusSection)
})

const saving = ref(false);
const editorInitializing = ref(false);
const allModels = ref<ModelConfig[]>([]);
const kbOptions = ref<{ label: string; value: string; type?: 'document' | 'faq'; count?: number; shared?: boolean; orgName?: string; ragEnabled?: boolean; wikiEnabled?: boolean; capabilities?: KBCapabilities }[]>([]);

const agentTypePresets = ref<AgentTypePreset[]>([]);
const agentSystemPromptTemplates = ref<PromptTemplate[]>([]);
const promptTemplates = ref<PromptTemplatesConfig | null>(null);
const intentPromptTemplates = ref<PromptTemplate[]>([]);
type McpSelectOption = { label: string; value: string; disabled?: boolean };

const mcpOptions = computed<McpSelectOption[]>(() => {
  const services = editorResources.mcpServices || [];
  const selectedIds = new Set<string>(formData.value.config.mcp_services || []);
  const serviceById = new Map(services.map((mcp) => [mcp.id, mcp]));
  const options: McpSelectOption[] = [];

  for (const mcp of services) {
    if (mcp.enabled) {
      options.push({ label: mcp.name, value: mcp.id });
    }
  }

  for (const id of selectedIds) {
    const mcp = serviceById.get(id);
    if (mcp && !mcp.enabled) {
      options.push({
        label: `${mcp.name} (${'Disabled'})`,
        value: mcp.id,
        disabled: true,
      });
    } else if (!mcp) {
      options.push({
        label: 'Unavailable service',
        value: id,
        disabled: true,
      });
    }
  }

  return options;
});

const showMcpServiceSelect = computed(() =>
  mcpOptions.value.length > 0 || (formData.value.config.mcp_services?.length ?? 0) > 0,
);
const webSearchProviderList = ref<WebSearchProviderEntity[]>([]);
const skillCatalog = ref<SkillCatalogItem[]>([]);
const catalogReady = ref(false);
const installingCatalogId = ref('');
const skillsSelectionMode = ref<'all' | 'selected' | 'none'>('none');
const hasSandboxSelected = computed(() => !!formData.value.config.sandbox_config_id);
const canEnableSkills = computed(() =>
  hasSandboxSelected.value || namedSandboxConfigs().length === 1,
);
const canInstallSkills = computed(() => authStore.hasRole('admin'));

type CatalogSkillRow = SkillCatalogItem & {
  installed: boolean
  selectable: boolean
  installStatus: string
  installEnabled: boolean
  // The install on this sandbox is still on an archive the catalog has moved past.
  upgradable: boolean
  installVersion: string
  // Set while a newer install runs or after it failed: the sandbox still runs
  // the previous version, so the skill stays usable.
  servedNote: string
}

const catalogSkillRows = computed<CatalogSkillRow[]>(() => {
  const sandboxId = formData.value.config.sandbox_config_id || ''
  return skillCatalog.value.map((item) => {
    const inst = sandboxId
      ? (item.installations || []).find((row) => row.sandbox_config_id === sandboxId)
      : undefined
    const installStatus = inst?.status || ''
    const installEnabled = Boolean(inst?.enabled)
    const installed = Boolean(inst) && installStatus !== 'removed'
    const servedNote = inst ? servedPreviousText(inst) : ''
    const selectable = installEnabled && (installStatus === 'ready' || Boolean(servedNote))
    const upgradable = Boolean(inst && installUpgradable(item, inst))
    return {
      ...item, installed, selectable, installStatus, installEnabled,
      upgradable, installVersion: inst?.version || '', servedNote,
    }
  })
})

const showCatalogSkillList = computed(() =>
  skillsSelectionMode.value !== 'none'
  && hasSandboxSelected.value
  && catalogSkillRows.value.length > 0,
)

const skillsSelectionHint = computed(() => {
  if (skillsSelectionMode.value === 'all') return 'All only includes skills already installed on this sandbox. Uninstalled skills are not added until you install them.'
  if (skillsSelectionMode.value === 'selected') return 'Check the skills this agent should use. Uninstalled skills cannot be checked — click Install on the right first.'
  return 'All workspace skills are listed here. Installed ones can be used now; others need Install first.'
})

const catalogSkillGroups = computed(() => {
  const ready = catalogSkillRows.value.filter((skill) => skill.selectable)
  const pending = catalogSkillRows.value.filter((skill) => !skill.selectable)
  const groups: { key: 'ready' | 'pending'; label: string; skills: CatalogSkillRow[] }[] = []
  if (ready.length) {
    groups.push({
      key: 'ready',
      label: 'Available',
      skills: ready,
    })
  }
  if (pending.length) {
    groups.push({
      key: 'pending',
      label: 'Unavailable',
      skills: pending,
    })
  }
  return groups
})

function skillStatusHint(skill: CatalogSkillRow): string {
  if (!skill.installed) return 'Not installed'
  if (skill.installStatus === 'installing') return 'Installing'
  if (skill.installStatus === 'failed') return 'Failed'
  if (skill.installStatus === 'removing') return 'Removing'
  if (skill.installStatus === 'ready' && !skill.installEnabled) {
    return 'Disabled on this sandbox'
  }
  return 'Not ready yet'
}

function skillStatusIcon(skill: CatalogSkillRow): string {
  if (!skill.installed || skill.installStatus === 'failed') return 'download'
  if (skill.installStatus === 'installing' || skill.installStatus === 'removing') return 'refresh'
  if (skill.installStatus === 'ready' && !skill.installEnabled) return 'close-circle'
  return 'time'
}

function isSkillBusy(skill: CatalogSkillRow): boolean {
  return skill.installStatus === 'installing' || skill.installStatus === 'removing'
}

function canInstallSkillRow(skill: CatalogSkillRow): boolean {
  if (!canInstallSkills.value || !hasSandboxSelected.value) return false
  return !skill.installed || skill.installStatus === 'failed'
}

// Upgrading writes the sandbox image through the same admin-only catalog
// install, so it is offered, and even mentioned, only to those who can run it.
function canUpgradeSkillRow(skill: CatalogSkillRow): boolean {
  return canInstallSkills.value && hasSandboxSelected.value && skill.upgradable
}

// Installing the catalog version over what this sandbox has is an upgrade:
// over an outdated install, or over a failed upgrade whose previous version
// still runs. Only a skill the sandbox has never carried is a plain install.
function installsAnUpgrade(skill: CatalogSkillRow): boolean {
  return skill.upgradable || Boolean(skill.servedNote)
}

function skillUpgradeHint(skill: CatalogSkillRow): string {
  const versions = upgradeVersions(skill, { version: skill.installVersion })
  return versions
    ? `Upgrade ${versions.from} → ${versions.to}`
    : 'Upgrade available'
}

function namedSandboxConfigs(): SandboxConfigRecord[] {
  return chatResources.sandboxConfigs.filter((cfg) => isNamedSandboxBackend(cfg.sandbox_type))
}

function autoBindSoleSandbox() {
  if (skillsSelectionMode.value === 'none') return
  if (formData.value.config.sandbox_config_id) return
  const configs = namedSandboxConfigs()
  if (configs.length === 1) {
    formData.value.config.sandbox_config_id = configs[0].id
  }
}

function openSkillSettings() {
  const configId = formData.value.config.sandbox_config_id || ''
  uiStore.openSettings('skills', configId || undefined)
}

const showSkillProgress = ref(false)
const skillProgressRecord = ref<SandboxConfigRecord | null>(null)
const skillProgressId = ref('')
const skillProgressTitle = ref('')
const skillProgressDesc = computed(() => {
  const record = skillProgressRecord.value
  if (!record) return ''
  return `Manage enablement, variables, and uninstall on sandbox “${record.name}”.`
})

function sandboxRecordById(configId: string): SandboxConfigRecord | undefined {
  return chatResources.sandboxConfigs.find((cfg) => cfg.id === configId)
}

function installOnCurrentSandbox(skill: CatalogSkillRow, configId: string) {
  return (skill.installations || []).find((row) => row.sandbox_config_id === configId)
}

async function openSkillInstallProgress(skill: CatalogSkillRow) {
  const configId = formData.value.config.sandbox_config_id || ''
  const record = sandboxRecordById(configId)
  if (!record) {
    openSkillSettings()
    return
  }
  let inst = installOnCurrentSandbox(skill, configId)
  if (!inst?.skill_id) {
    await syncInstalledSkills(true)
    const latest = catalogSkillRows.value.find((row) => row.id === skill.id)
    inst = latest ? installOnCurrentSandbox(latest, configId) : undefined
  }
  if (!inst?.skill_id) {
    openSkillSettings()
    return
  }
  skillProgressRecord.value = record
  skillProgressId.value = inst.skill_id
  skillProgressTitle.value = skill.name
  showSkillProgress.value = true
}

function onSkillProgressUpdated() {
  void syncInstalledSkills(true)
}

function onSkillProgressChanged() {
  void syncInstalledSkills(true)
}

function pruneSelectedSkills() {
  if (!catalogReady.value) return
  // A skill being upgraded is briefly not ready, and dropping it here would
  // silently unselect it for good once the agent is saved.
  const names = new Set(catalogSkillRows.value
    .filter((skill) => skill.selectable || (skill.installed && isSkillBusy(skill)))
    .map((skill) => skill.name))
  const selected: string[] = formData.value.config.selected_skills || []
  const kept = selected.filter((name: string) => names.has(name))
  if (kept.length !== selected.length) {
    formData.value.config.selected_skills = kept
  }
}

async function syncInstalledSkills(force = false) {
  autoBindSoleSandbox()
  const configId = formData.value.config.sandbox_config_id || ''
  // The editor only edits this workspace's agents, so the sandbox config is
  // local and needs no source-workspace scope.
  await editorResources.ensureSkills(configId, undefined, force)
  try {
    await editorResources.ensureSkillCatalog(force)
    skillCatalog.value = [...editorResources.skillCatalog]
    catalogReady.value = true
  } catch {
    catalogReady.value = false
  }
  pruneSelectedSkills()
}

async function installCatalogToCurrent(skill: CatalogSkillRow) {
  const configId = formData.value.config.sandbox_config_id || ''
  if (!configId || installingCatalogId.value) return
  installingCatalogId.value = skill.id
  const upgrading = installsAnUpgrade(skill)
  try {
    const res = await installSkillCatalog(skill.id, [configId])
    const failed = Object.keys(res?.data?.errors || {}).length
    if (failed > 0) {
      MessagePlugin.warning(`Started on some sandboxes. ${failed} could not start.`)
    } else {
      MessagePlugin.success((upgrading ? 'Upgrade started' : 'Install started'))
    }
    await syncInstalledSkills(true)
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to upload the skill')
  } finally {
    installingCatalogId.value = ''
  }
}
// Named sandbox backend configs in this workspace. The currently selected one is always
// included even if it was deleted, otherwise the select would silently read "sandbox
// disabled" and hide the fact that the agent points at a config that no longer exists.
const sandboxConfigOptions = computed(() => {
  const configs = chatResources.sandboxConfigs.filter((cfg) => isNamedSandboxBackend(cfg.sandbox_type));
  const selected = formData.value.config.sandbox_config_id;
  if (!selected || configs.some((cfg) => cfg.id === selected)) return configs;
  return [
    ...configs,
    { id: selected, name: 'Config deleted', sandbox_type: '' } as SandboxConfigRecord,
  ];
});
const backendLabel = (type: string) =>
  type ? (SETTINGS_SANDBOX_BACKENDS_LABELS[type] ?? '') : 'Error';

function sandboxTargetLine(cfg: SandboxConfigRecord): string {
  if (cfg.sandbox_type === 'docker') {
    return cfg.config?.docker?.image?.trim() || ''
  }
  const remote = cfg.config?.e2b || cfg.config?.cube
  const raw = remote?.api_url?.trim() || ''
  if (!raw) return ''
  try {
    return new URL(raw).host
  } catch {
    return raw
  }
}

const selectedSandboxSummary = computed(() => {
  const id = formData.value.config.sandbox_config_id
  const cfg = sandboxConfigOptions.value.find((item) => item.id === id)
  if (!cfg?.sandbox_type) return ''
  const parts = [backendLabel(cfg.sandbox_type), sandboxTargetLine(cfg)]
  const desc = cfg.description?.trim()
  if (desc) parts.push(desc)
  return parts.filter(Boolean).join(' · ')
})
const storageEngineStatus = ref<StorageEngineStatusItem[]>([]);
const imageStorageOptions = computed(() => {
  const statusMap: Record<string, boolean> = {};
  for (const e of storageEngineStatus.value) {
    statusMap[e.name] = e.available;
  }
  return [
    { value: 'local', label: 'Local', disabled: false },
    { value: 'minio', label: 'MinIO', disabled: statusMap.minio === false },
    { value: 's3', label: 'Amazon S3', disabled: statusMap.s3 === false },
  ];
});

// Default system prompt for agent (smart-reasoning) mode. Taken from the prompt-templates
// agent_system_prompt array, the entry with mode==='rag' && default, i.e. the same source of
// truth as the backend agent.GetProgressiveRAGSystemPrompt.
const defaultAgentSystemPrompt = ref('');
const defaultNormalSystemPrompt = ref('');
const defaultContextTemplate = ref('');
const defaultRewritePromptSystem = ref('');
const defaultRewritePromptUser = ref('');
const defaultFallbackPrompt = ref('');
const defaultFallbackResponse = ref('');
const defaultEmbeddingTopK = ref(10);
const defaultKeywordThreshold = ref(0.3);
const defaultVectorThreshold = ref(0.5);
const defaultRerankTopK = ref(5);
const defaultRerankThreshold = ref(0.5);
const defaultQuickAnswerMaxCompletionTokens = 2048;
const defaultSmartReasoningMaxCompletionTokens = 4096;
const defaultSandboxWriteMaxCompletionTokens = 24576;
const defaultTemperature = ref(0.7);

const defaultMaxCompletionTokensFor = (mode: string, sandboxConfigId?: string) => {
  if (mode === 'smart-reasoning') {
    return sandboxConfigId
      ? defaultSandboxWriteMaxCompletionTokens
      : defaultSmartReasoningMaxCompletionTokens;
  }
  return defaultQuickAnswerMaxCompletionTokens;
};

const knowledgeBaseTools = ['search_knowledge', 'read_document', 'list_documents'];

const wikiReadTools = ['wiki_search', 'wiki_read_page', 'read_document', 'wiki_flag_issue'];

const isInitializing = ref(false);

const kbSelectionMode = ref<'all' | 'selected' | 'none'>('none');

const mcpSelectionMode = ref<'all' | 'selected' | 'none'>('none');

// A tool's KB capability requirements are declared once in `@/utils/tool-capabilities` and
// read here through `evaluateToolRequirement`; do not duplicate them in this list.
const allTools = computed(() => [
  { value: 'thinking', label: 'Thinking', description: 'Dynamic and reflective problem-solving thinking tool', group: 'base' },
  { value: 'todo_write', label: 'Plan', description: 'Create structured research plans', group: 'base' },
  // The legacy grep_chunks / knowledge_search / list_knowledge_chunks / get_document_info tools
  // were merged; old configs are mapped onto the new names by normalizeLegacyToolNames.
  { value: 'search_knowledge', label: 'Search knowledge', description: 'Semantic, keyword or hybrid search over knowledge-base chunks', group: 'rag' },
  { value: 'read_document', label: 'Read document', description: 'Read a document\'s metadata and chunks, with paging and in-document search', group: 'rag' },
  { value: 'list_documents', label: 'List documents', description: 'Page through the documents of a knowledge base', group: 'rag' },
  { value: 'query_knowledge_graph', label: 'Query Knowledge Graph', description: 'Query relationships from knowledge graph', group: 'rag' },
  { value: 'database_query', label: 'Query Database', description: 'Query information from the database', group: 'rag' },
  { value: 'wiki_search', label: 'Search Wiki', description: 'Keyword / semantic search over Wiki pages', group: 'wiki_read' },
  { value: 'wiki_read_page', label: 'Read Wiki Page', description: 'Read the full content of a specific Wiki page', group: 'wiki_read' },
  { value: 'wiki_flag_issue', label: 'Flag Wiki Issue', description: 'Flag factual errors or merge conflicts on a Wiki page', group: 'wiki_read' },
  { value: 'wiki_write_page', label: 'Create / Overwrite Wiki', description: 'Create a new page or fully overwrite an existing one', group: 'wiki_edit', danger: true },
  { value: 'wiki_replace_text', label: 'Replace Text in Wiki', description: 'Replace specific text in a Wiki page', group: 'wiki_edit', danger: true },
  { value: 'wiki_rename_page', label: 'Rename Wiki Page', description: 'Rename a Wiki page and auto-update cross-links', group: 'wiki_edit', danger: true },
  { value: 'wiki_delete_page', label: 'Delete Wiki Page', description: 'Delete a Wiki page and clean up dead links', group: 'wiki_edit', danger: true },
  { value: 'wiki_read_issue', label: 'View Wiki Issue', description: 'View details of a Wiki page issue', group: 'wiki_issue' },
  { value: 'wiki_update_issue', label: 'Update Wiki Issue', description: 'Update the status of a Wiki page issue', group: 'wiki_issue' },
  { value: 'data_analysis', label: 'Data Analysis', description: 'Understand data files and perform data analysis', group: 'data' },
  { value: 'data_schema', label: 'View Data Schema', description: 'Get metadata of tabular files', group: 'data' },
]);

const toolGroups = computed(() => [
  { key: 'base', label: 'Basic' },
  { key: 'rag', label: 'Knowledge Retrieval (RAG)' },
  { key: 'wiki_read', label: 'Wiki Read' },
  { key: 'wiki_edit', label: 'Wiki Edit' },
  { key: 'wiki_issue', label: 'Wiki Review' },
  { key: 'data', label: 'Data Analysis' },
]);

const myKbOptions = computed(() => kbOptions.value.filter(kb => !kb.shared));
const sharedKbOptions = computed(() => kbOptions.value.filter(kb => kb.shared));

const hasKnowledgeBase = computed(() => {
  return kbSelectionMode.value !== 'none';
});

const showRerankModelField = computed(() => {
  if (!isAgentMode.value) return hasKnowledgeBase.value;
  return hasKnowledgeBase.value || agentRequiresRerankModel(formData.value.config);
});

// Note: the user may select knowledge_bases (KB level) or knowledge_ids (document level).
// This is only used for UI tool-availability checks, so it is computed at KB level.
const kbsInScope = computed(() => {
  if (kbSelectionMode.value === 'none') return [];
  if (kbSelectionMode.value === 'all') return kbOptions.value;
  const selectedIds = formData.value.config.knowledge_bases || [];
  return kbOptions.value.filter(kb => selectedIds.includes(kb.value));
});

const hasRagKnowledgeBase = computed(() => {
  return kbsInScope.value.some(kb => kb.ragEnabled);
});

const hasWikiKnowledgeBase = computed(() => {
  return kbsInScope.value.some(kb => kb.wikiEnabled);
});

const ragKbCount = computed(() => kbsInScope.value.filter(kb => kb.ragEnabled).length);
const wikiKbCount = computed(() => kbsInScope.value.filter(kb => kb.wikiEnabled).length);

const hasFaqKnowledgeBase = computed(() => {
  if (kbSelectionMode.value === 'none') return false;
  if (kbSelectionMode.value === 'all') {
    return kbOptions.value.some(kb => kb.type === 'faq');
  }
  const selectedKbIds = formData.value.config.knowledge_bases || [];
  return kbOptions.value.some(kb => selectedKbIds.includes(kb.value) && kb.type === 'faq');
});

// Aggregates the KB capabilities in scope into one ScopeCapabilities object for
// `evaluateToolRequirement`; every availability hint in the UI must come from here.
const scopeCapabilities = computed<ScopeCapabilities>(() => {
  const scope: ScopeCapabilities = { vector: false, keyword: false, wiki: false, graph: false, faq: false };
  for (const kb of kbsInScope.value) {
    const caps = kb.capabilities;
    if (caps) {
      if (caps.vector) scope.vector = true;
      if (caps.keyword) scope.keyword = true;
      if (caps.wiki) scope.wiki = true;
      if (caps.graph) scope.graph = true;
      if (caps.faq) scope.faq = true;
    } else {
      // Backward compatible: capabilities not loaded yet, fall back to ragEnabled/wikiEnabled
      if (kb.ragEnabled) { scope.vector = true; scope.keyword = true; }
      if (kb.wikiEnabled) scope.wiki = true;
      if (kb.type === 'faq') scope.faq = true;
    }
  }
  return scope;
});

const missKindToReason = (kind: RequirementMissKind): string | undefined => {
  switch (kind) {
    case 'needsKb': return '(requires knowledge base configuration)';
    case 'needsWiki': return '(requires a Wiki-enabled knowledge base)';
    case 'needsRag':
    case 'needsGraph':
    case 'needsFaq': return '(requires a KB with vector/keyword indexing enabled)';
    case 'none':
    default: return undefined;
  }
};

const availableTools = computed(() => {
  const scope = scopeCapabilities.value;
  const hasAnyKb = hasKnowledgeBase.value;
  return allTools.value.map(tool => {
    const { ok, missKind } = evaluateToolRequirement(tool.value, scope, hasAnyKb);
    return {
      ...tool,
      disabled: !ok,
      disabledReason: ok ? undefined : missKindToReason(missKind),
    };
  });
});

const groupedAvailableTools = computed(() => {
  const map: Record<string, typeof availableTools.value> = {};
  for (const tool of availableTools.value) {
    const g = tool.group || 'base';
    if (!map[g]) map[g] = [];
    map[g].push(tool);
  }
  return toolGroups.value
    .map(g => ({
      ...g,
      tools: map[g.key] || [],
    }))
    .filter(g => g.tools.length > 0);
});

// ==================== Effective tool preview ====================
// The tools the agent can actually use at runtime (preview only).
// Rules, applied on top of allowed_tools:
//   1) tools that are checked but missing the required capability (no KB / no Wiki-capable KB)
//      are greyed out or hidden
//   2) web_search / web_fetch follow web_search_enabled whether or not they are checked
//   3) when kb_selection_mode === 'none', RAG and Wiki tools are all treated as unavailable
const effectiveTools = computed(() => {
  const chosen = new Set(formData.value.config.allowed_tools || []);
  const items: Array<{ value: string; label: string; reason?: string; active: boolean }> = [];
  for (const tool of availableTools.value) {
    const picked = chosen.has(tool.value);
    if (!picked) continue;
    if (tool.disabled) {
      items.push({ value: tool.value, label: tool.label, active: false, reason: tool.disabledReason });
    } else {
      items.push({ value: tool.value, label: tool.label, active: true });
    }
  }
  if (formData.value.config.web_search_enabled) {
    items.push({ value: 'web_search', label: 'Web Search', active: true });
    items.push({ value: 'web_fetch', label: 'Web Fetch', active: true });
  }
  return items;
});

const inactiveToolCount = computed(() => effectiveTools.value.filter(i => !i.active).length);

const availableFileTypes = [
  { value: 'pdf', label: 'PDF', description: 'PDF Documents' },
  { value: 'docx', label: 'Word', description: 'Word Documents (.docx/.doc)' },
  { value: 'txt', label: 'Text', description: 'Plain Text Files (.txt)' },
  { value: 'md', label: 'Markdown', description: 'Markdown Documents' },
  { value: 'csv', label: 'CSV', description: 'Comma-Separated Value Files' },
  { value: 'xlsx', label: 'Excel', description: 'Excel Spreadsheets (.xlsx/.xls)' },
  { value: 'jpg', label: 'Images', description: 'Image Files (.jpg/.jpeg/.png)' },
];

const placeholderData = ref<{
  system_prompt: PlaceholderDefinition[];
  agent_system_prompt: PlaceholderDefinition[];
  context_template: PlaceholderDefinition[];
  rewrite_system_prompt: PlaceholderDefinition[];
  rewrite_prompt: PlaceholderDefinition[];
  fallback_prompt: PlaceholderDefinition[];
}>({
  system_prompt: [],
  agent_system_prompt: [],
  context_template: [],
  rewrite_system_prompt: [],
  rewrite_prompt: [],
  fallback_prompt: [],
});

const availablePlaceholders = computed(() => {
  return isAgentMode.value ? placeholderData.value.agent_system_prompt : placeholderData.value.system_prompt;
});

const contextTemplatePlaceholders = computed(() => placeholderData.value.context_template);

const rewriteSystemPlaceholders = computed(() => placeholderData.value.rewrite_system_prompt);

const rewritePlaceholders = computed(() => placeholderData.value.rewrite_prompt);

const fallbackPlaceholders = computed(() => placeholderData.value.fallback_prompt);

const promptTextareaRef = ref<any>(null);
const showPlaceholderPopup = ref(false);
const selectedPlaceholderIndex = ref(0);
const placeholderPrefix = ref('');
const popupStyle = ref({ top: '0px', left: '0px' });
let placeholderPopupTimer: any = null;

const contextTemplateTextareaRef = ref<any>(null);
const showContextPlaceholderPopup = ref(false);
const selectedContextPlaceholderIndex = ref(0);
const contextPlaceholderPrefix = ref('');
const contextPopupStyle = ref({ top: '0px', left: '0px' });
let contextPlaceholderPopupTimer: any = null;

const selectedIntent = ref('');
const intentEditorValue = ref('');
const intentPromptsSyncing = ref(false);
const intentPromptTextareaRef = ref<any>(null);

interface PlaceholderPopupState {
  show: boolean;
  selectedIndex: number;
  prefix: string;
  style: { top: string; left: string };
  timer: any;
  fieldKey: string;
  placeholders: PlaceholderDefinition[];
}

const intentPromptPopup = ref<PlaceholderPopupState>({
  show: false, selectedIndex: 0, prefix: '', style: { top: '0px', left: '0px' }, timer: null, fieldKey: 'intent_prompt', placeholders: []
});

const rewriteSystemPopup = ref<PlaceholderPopupState>({
  show: false, selectedIndex: 0, prefix: '', style: { top: '0px', left: '0px' }, timer: null, fieldKey: 'rewrite_prompt_system', placeholders: []
});
const rewriteUserPopup = ref<PlaceholderPopupState>({
  show: false, selectedIndex: 0, prefix: '', style: { top: '0px', left: '0px' }, timer: null, fieldKey: 'rewrite_prompt_user', placeholders: []
});
const fallbackPromptPopup = ref<PlaceholderPopupState>({
  show: false, selectedIndex: 0, prefix: '', style: { top: '0px', left: '0px' }, timer: null, fieldKey: 'fallback_prompt', placeholders: []
});

const rewriteSystemTextareaRef = ref<any>(null);
const rewriteUserTextareaRef = ref<any>(null);
const fallbackPromptTextareaRef = ref<any>(null);

const navItems = computed(() => {
  const items: { key: string; icon: string; label: string; badge?: number }[] = [
    { key: 'basic', icon: 'info-circle', label: 'Basic Info' },
    { key: 'prompts', icon: 'file-paste', label: 'Prompts', badge: promptNavItems.value.length > 1 ? promptNavItems.value.length : undefined },
    { key: 'model', icon: 'control-platform', label: 'Model Config' },
    { key: 'suggestions', icon: 'help-circle', label: 'Question suggestions' },
  ];
  items.push({ key: 'conversation', icon: 'chat', label: 'Conversation' });
  items.push({ key: 'knowledge', icon: 'folder', label: 'Knowledge Base' });
  if (hasKnowledgeBase.value) {
    items.push({ key: 'retrieval', icon: 'search', label: 'Retrieval Strategy' });
  }
  items.push({ key: 'websearch', icon: 'internet', label: 'Web Search' });
  items.push({ key: 'multimodal', icon: 'attach', label: 'Attachment Upload' });
  if (isAgentMode.value) {
    items.push({ key: 'tools', icon: 'tools', label: 'Tools' });
    items.push({ key: 'mcp', icon: 'server', label: 'MCP Services' });
    items.push({ key: 'skills', icon: SKILL_ICON, label: 'Skills' });
  }
  if (editorMode.value === 'edit' && editorAgent.value?.id && !editorAgent.value?.is_builtin && !authStore.isLiteMode) {
    items.push({ key: 'share', icon: 'share', label: 'Sharing' });
  }
  return items;
});

const navGroups = computed(() => {
  const itemMap = new Map(navItems.value.map((item) => [item.key, item]));
  const pickItems = (keys: string[]) =>
    keys.map((key) => itemMap.get(key)).filter(Boolean) as typeof navItems.value;
  return [
    {
      key: 'basic',
      label: 'Basics',
      items: pickItems(['basic', 'prompts', 'model', 'conversation', 'suggestions']),
    },
    {
      key: 'knowledge',
      label: 'Knowledge & Retrieval',
      items: pickItems(['knowledge', 'retrieval', 'websearch']),
    },
    {
      key: 'capability',
      label: 'Extensions',
      items: pickItems(['multimodal', 'tools', 'mcp', 'skills']),
    },
    {
      key: 'integration',
      label: 'Publish & Integrations',
      items: pickItems(['share']),
    },
  ].filter((group) => group.items.length > 0);
});

const defaultFormData = {
  name: '',
  description: '',
  is_builtin: false,
  config: {
    agent_mode: 'smart-reasoning' as 'quick-answer' | 'smart-reasoning',
    system_prompt: '',
    context_template: '',
    model_id: '',
    rerank_model_id: '',
    temperature: 0.7,
    max_completion_tokens: 0,
    thinking: false,
    reasoning_effort: 'off',
    citation_enabled: true,
    max_iterations: 10,
    llm_call_timeout: 120,  // 120 seconds
    allowed_tools: [] as string[],
    reflection_enabled: false,
    mcp_selection_mode: 'none' as 'all' | 'selected' | 'none',
    mcp_services: [] as string[],
    mcp_auth_wait_timeout: 600,
    skills_selection_mode: 'none' as 'all' | 'selected' | 'none',
    selected_skills: [] as string[],
    // Which workspace sandbox config the skill scripts run on. Empty disables script execution.
    sandbox_config_id: '' as string,
    // New agents default to all knowledge bases so users can start without picking KBs first.
    kb_selection_mode: 'all' as 'all' | 'selected' | 'none',
    knowledge_bases: [] as string[],
    retrieve_kb_only_when_mentioned: false,
    // Type preset for smart reasoning; overwritten by the agent's own agent_type when editing.
    agent_type: 'rag-qa' as AgentType,
    system_prompt_id: '' as string,
    image_upload_enabled: false,
    vlm_model_id: '',
    image_storage_provider: '',
    attachment_image_understanding: false,
    attachment_ocr_max_pages: 0,
    attachment_parse_wait_timeout_sec: 0,
    chat_parser_engine_rules: [] as ParserEngineRule[],
    supported_file_types: [] as string[],
    data_analysis_enabled: false,
    faq_priority_enabled: true,
    faq_direct_answer_threshold: 0.9,
    faq_score_boost: 1.2,
    web_search_enabled: false,
    web_search_max_results: 5,
    multi_turn_enabled: false,
    history_turns: 5,
    retain_retrieval_history: false,
    // Long-term memory follows the workspace setting by default: true and unset are equivalent,
    // only false makes this agent skip memory on its own.
    memory_enabled: true,
    embedding_top_k: 10,
    keyword_threshold: 0.3,
    vector_threshold: 0.5,
    rerank_top_k: 5,
    rerank_threshold: 0.5,
    enable_query_expansion: true,
    enable_rewrite: true,
    query_understand_model_id: '',
    rewrite_prompt_system: '',
    rewrite_prompt_user: '',
    fallback_strategy: 'model' as 'fixed' | 'model',
    fallback_response: '',
    fallback_prompt: '',
    question_suggestions: {
      starters: {
        enabled: true,
        mode: 'hybrid' as 'curated' | 'knowledge' | 'hybrid',
        items: [] as string[],
        count: 6,
      },
      follow_ups: {
        enabled: false,
        mode: 'hybrid' as 'generated' | 'knowledge' | 'hybrid',
        count: 3,
        model_id: '',
        additional_instruction: '',
        categories: ['clarify', 'deepen', 'action'] as Array<'clarify' | 'deepen' | 'action'>,
        max_context_turns: 2,
        suppress_on_fallback: true,
        suppress_when_answer_asks_question: true,
        knowledge_fallback: true,
        allow_regenerate: false,
      },
    },
    welcome_message: '',
  }
};

const formData = ref(JSON.parse(JSON.stringify(defaultFormData)));

const starterSuggestionModeOptions = computed(() => [
  { value: 'curated', label: 'Curated' },
  { value: 'knowledge', label: 'Knowledge' },
  { value: 'hybrid', label: 'Hybrid' },
]);
const followUpSuggestionModeOptions = computed(() => [
  { value: 'generated', label: 'Generated' },
  { value: 'knowledge', label: 'Knowledge' },
  { value: 'hybrid', label: 'Hybrid' },
]);
const followUpCategoryOptions = computed(() => [
  { value: 'clarify', label: 'Clarify' },
  { value: 'deepen', label: 'Deepen' },
  { value: 'action', label: 'Next step' },
]);

const addStarterSuggestion = () => {
  const items = formData.value.config.question_suggestions.starters.items;
  if (items.length < 8) items.push('');
};

const removeStarterSuggestion = (index: number) => {
  formData.value.config.question_suggestions.starters.items.splice(index, 1);
};

const applyDefaultModelsIfEmpty = () => {
  if (props.mode !== 'create' || !formData.value) return
  const chatModelId = selectInitialModelId(allModels.value, 'KnowledgeQA')
  const rerankModelId = selectInitialModelId(allModels.value, 'Rerank')
  if (!formData.value.config.model_id && chatModelId) {
    formData.value.config.model_id = chatModelId
  }
  if (!formData.value.config.rerank_model_id && rerankModelId) {
    formData.value.config.rerank_model_id = rerankModelId
  }
}

const agentMode = computed({
  get: () => formData.value.config.agent_mode,
  set: (val: 'quick-answer' | 'smart-reasoning') => { formData.value.config.agent_mode = val; }
});

const isAgentMode = computed(() => agentMode.value === 'smart-reasoning');

const effectiveDefaultMaxCompletionTokens = computed(() =>
  defaultMaxCompletionTokensFor(agentMode.value, formData.value.config.sandbox_config_id),
);

const maxCompletionTokensMode = computed({
  get: () => (formData.value.config.max_completion_tokens > 0 ? 'custom' : 'default'),
  set: (mode: 'default' | 'custom') => {
    if (mode === 'default') {
      formData.value.config.max_completion_tokens = 0;
      return;
    }
    if (!formData.value.config.max_completion_tokens) {
      formData.value.config.max_completion_tokens = effectiveDefaultMaxCompletionTokens.value;
    }
  },
});

const lastFiniteMaxIterations = ref(10);
const maxIterationsMode = computed({
  get: () => (formData.value.config.max_iterations < 0 ? 'unlimited' : 'limit'),
  set: (mode: 'limit' | 'unlimited') => {
    if (mode === 'unlimited') {
      if (formData.value.config.max_iterations > 1) {
        lastFiniteMaxIterations.value = formData.value.config.max_iterations;
      }
      formData.value.config.max_iterations = -1;
      return;
    }
    const restored = lastFiniteMaxIterations.value > 1 ? lastFiniteMaxIterations.value : 10;
    formData.value.config.max_iterations = restored;
  },
});

const currentIntentTemplate = computed(() =>
  intentPromptTemplates.value.find((template) => template.id === selectedIntent.value),
);

const currentIntentTemplateDesc = computed(() =>
  currentIntentTemplate.value?.description || '',
);

const isIntentCustomized = (intentId: string) => {
  const overrides = formData.value.config.intent_prompts || {};
  const override = overrides[intentId];
  if (!override?.trim()) return false;
  const template = intentPromptTemplates.value.find((item) => item.id === intentId);
  return override.trim() !== (template?.content || '').trim();
};

const activePromptAnchor = ref('system');

const hasAnyIntentCustomized = computed(() =>
  intentPromptTemplates.value.some((item) => isIntentCustomized(item.id)),
);

const conversationSectionDesc = computed(() =>
  isAgentMode.value
    ? 'Smart reasoning is always multi-turn. Earlier conversation is kept up to the model context window, and older turns are summarized automatically once it fills'
    : 'Configure multi-turn conversation and query rewriting parameters',
);

const showRewritePrompts = computed(() =>
  !isAgentMode.value
  && formData.value.config.multi_turn_enabled
  && formData.value.config.enable_rewrite,
);

const promptNavItems = computed(() => {
  type PromptNavItem = { key: string; label: string; customized?: boolean };
  const items: PromptNavItem[] = [
    {
      key: 'system',
      label: 'System prompt',
      customized: !!formData.value.config.system_prompt?.trim(),
    },
  ];
  if (!isAgentMode.value) {
    items.push({
      key: 'context',
      label: 'Context template',
      customized: !!formData.value.config.context_template?.trim(),
    });
    items.push({
      key: 'intent',
      label: 'Intent prompts',
      customized: hasAnyIntentCustomized.value,
    });
    if (showRewritePrompts.value) {
      items.push(
        {
          key: 'rewrite-system',
          label: 'Rewrite · System',
          customized: !!formData.value.config.rewrite_prompt_system?.trim(),
        },
        {
          key: 'rewrite-user',
          label: 'Rewrite · User',
          customized: !!formData.value.config.rewrite_prompt_user?.trim(),
        },
      );
    }
    if (hasKnowledgeBase.value) {
      items.push({
        key: 'fallback',
        label: 'Retrieval fallback',
      });
    }
  }
  return items;
});

const syncActivePromptAnchor = () => {
  const items = promptNavItems.value;
  if (!items.length) return;
  if (!items.some((item) => item.key === activePromptAnchor.value)) {
    activePromptAnchor.value = items[0].key;
  }
};

watch(promptNavItems, syncActivePromptAnchor);

watch(currentSection, (section) => {
  if (section === 'prompts') {
    syncActivePromptAnchor();
  }
});

const agentIMChannelCount = ref(0);
const agentEmbedChannelCount = ref(0);

async function loadAgentIntegrationCounts(agentId: string) {
  try {
    const [imResp, embedResp] = await Promise.all([
      listIMChannels(agentId),
      listEmbedChannels(agentId),
    ]);
    agentIMChannelCount.value = imResp?.data?.length ?? 0;
    agentEmbedChannelCount.value = embedResp?.data?.length ?? 0;
  } catch {
    agentIMChannelCount.value = 0;
    agentEmbedChannelCount.value = 0;
  }
}

function gotoIntegrations(tab: 'im' | 'embed') {
  const agentId = editorAgent.value?.id;
  if (!agentId) return;
  handleClose();
  router.push({ path: '/platform/settings', query: { section: integrationSectionKey(tab), agentId } });
}

const filteredIntentPlaceholders = computed(() => {
  if (!intentPromptPopup.value.prefix) {
    return placeholderData.value.system_prompt;
  }
  const prefix = intentPromptPopup.value.prefix.toLowerCase();
  return placeholderData.value.system_prompt.filter(p => p.name.toLowerCase().startsWith(prefix));
});

const syncIntentEditorFromSelection = () => {
  const key = selectedIntent.value;
  if (!key) {
    intentEditorValue.value = '';
    return;
  }
  const overrides = formData.value.config.intent_prompts || {};
  intentEditorValue.value = overrides[key] ?? currentIntentTemplate.value?.content ?? '';
};

watch(selectedIntent, () => {
  intentPromptPopup.value.show = false;
  intentPromptPopup.value.prefix = '';
  syncIntentEditorFromSelection();
});

watch(
  () => intentPromptTemplates.value,
  (templates) => {
    if (!selectedIntent.value && templates.length > 0) {
      selectedIntent.value = templates[0].id;
    } else if (selectedIntent.value) {
      syncIntentEditorFromSelection();
    }
  },
  { immediate: true },
);

watch(intentEditorValue, (value) => {
  const key = selectedIntent.value;
  if (!key || intentPromptsSyncing.value) return;
  const defaultContent = currentIntentTemplate.value?.content || '';
  const next = value.trim();
  if (!next || next === defaultContent.trim()) {
    if (formData.value.config.intent_prompts) {
      const { [key]: _removed, ...rest } = formData.value.config.intent_prompts;
      if (Object.keys(rest).length === 0) {
        delete formData.value.config.intent_prompts;
      } else {
        formData.value.config.intent_prompts = rest;
      }
    }
    return;
  }
  formData.value.config.intent_prompts = {
    ...(formData.value.config.intent_prompts || {}),
    [key]: value,
  };
});

watch(
  () => formData.value.config.intent_prompts,
  () => {
    intentPromptsSyncing.value = true;
    syncIntentEditorFromSelection();
    intentPromptsSyncing.value = false;
  },
  { deep: true },
);

const resetCurrentIntentPrompt = () => {
  const key = selectedIntent.value;
  if (!key || !formData.value.config.intent_prompts) return;
  const { [key]: _removed, ...rest } = formData.value.config.intent_prompts;
  if (Object.keys(rest).length === 0) {
    delete formData.value.config.intent_prompts;
  } else {
    formData.value.config.intent_prompts = rest;
  }
  syncIntentEditorFromSelection();
};


const agentType = computed({
  get: () => (formData.value.config.agent_type as AgentType) || 'custom',
  set: (val: AgentType) => { formData.value.config.agent_type = val; },
});

const activeAgentTypePreset = computed<AgentTypePreset | null>(() => {
  if (!isAgentMode.value) return null;
  const id = agentType.value;
  if (!id || id === 'custom') return null;
  return agentTypePresets.value.find(p => p.id === id) || null;
});

const agentTypePresetLabel = (p: AgentTypePreset): string => p.i18n?.default?.label || p.id;
const agentTypePresetDescription = (p: AgentTypePreset): string => p.i18n?.default?.description || '';

const agentTypeSelectOptions = computed(() => {
  return agentTypePresets.value.map(p => ({
    value: p.id,
    label: agentTypePresetLabel(p),
    desc: agentTypePresetDescription(p),
  }));
});

const getPresetDefaultName = (preset: AgentTypePreset | null): string => {
  if (!preset || preset.id === 'custom') return '';
  return `My ${agentTypePresetLabel(preset)}`;
};
const getPresetDefaultDescription = (preset: AgentTypePreset | null): string => {
  if (!preset) return '';
  return agentTypePresetDescription(preset);
};

// Whether the name/description is still a system-generated value (any preset default, or empty);
// switching type must overwrite only values the user has not edited by hand.
const isNameSystemGenerated = (name: string): boolean => {
  if (!name) return true;
  return agentTypePresets.value.some(p => getPresetDefaultName(p) === name);
};
const isDescriptionSystemGenerated = (desc: string): boolean => {
  if (!desc) return true;
  return agentTypePresets.value.some(p => getPresetDefaultDescription(p) === desc);
};

// User-facing reason why a KB does not fit a preset. Do not surface the underlying capability
// names (vector / keyword / wiki): users only want to know why their knowledge base cannot be used.
const presetKbMismatchKeyMap: Record<string, string> = {
  'rag-qa': 'ragQa',
  'wiki-qa': 'wikiQa',
  'hybrid-rag-wiki': 'hybridRagWiki',
  'data-analysis': 'dataAnalysis',
};
const presetKbMismatchReason = (preset: AgentTypePreset): string => {
  const subKey = presetKbMismatchKeyMap[preset.id];
  if (subKey) return (AGENT_EDITOR_AGENT_TYPE_KB_MISMATCH_LABELS[subKey] ?? '');
  return 'Not compatible with current type';
};

// Effective KB filter for a preset: derived from its tools, plus the YAML increment.
//
// Design rules:
//   - tools -> any_of ("the KB must be usable by at least one of them") is computed by
//     `deriveKbFilterFromTools`;
//   - `kb_filter` in YAML only carries the business rules the tools cannot express (such as
//     data-analysis's `none_of: ["faq"]`), merged as an increment rather than a full override;
//   - `all_of` / `none_of` are inherited straight from YAML (tools express no such constraint).
//
// So rag-qa / wiki-qa / hybrid declare no `kb_filter` at all, data-analysis declares only the extra
// `none_of`, and the tool-to-capability mapping is maintained in one place,
// `@/utils/tool-capabilities`.
const effectiveKbFilter = (preset: AgentTypePreset | null): AgentTypeKBFilter | null => {
  if (!preset) return null;
  const derived = deriveKbFilterFromTools(preset.config?.allowed_tools || []);
  const yaml = preset.kb_filter;

  const anyOf = (yaml?.any_of && yaml.any_of.length > 0) ? yaml.any_of : (derived?.any_of ?? []);
  const allOf = yaml?.all_of ?? [];
  const noneOf = yaml?.none_of ?? [];
  if (anyOf.length === 0 && allOf.length === 0 && noneOf.length === 0) return null;
  return { any_of: anyOf, all_of: allOf, none_of: noneOf };
};

const kbSatisfiesPresetFilter = (kb: { capabilities?: KBCapabilities; ragEnabled?: boolean; wikiEnabled?: boolean; type?: string }, preset: AgentTypePreset | null): { ok: boolean; reason: string } => {
  const filter = effectiveKbFilter(preset);
  if (!preset || !filter) return { ok: true, reason: '' };
  const caps = kb.capabilities || {
    vector: !!kb.ragEnabled,
    keyword: !!kb.ragEnabled,
    wiki: !!kb.wikiEnabled,
    graph: false,
    faq: kb.type === 'faq',
  };
  const has = (name: string): boolean => {
    switch (name) {
      case 'vector': return !!caps.vector;
      case 'keyword': return !!caps.keyword;
      case 'wiki': return !!caps.wiki;
      case 'graph': return !!caps.graph;
      case 'faq': return !!caps.faq;
      default: return false;
    }
  };
  const reason = presetKbMismatchReason(preset);
  if (filter.all_of && filter.all_of.length > 0) {
    for (const n of filter.all_of) {
      if (!has(n)) return { ok: false, reason };
    }
  }
  if (filter.any_of && filter.any_of.length > 0) {
    if (!filter.any_of.some(n => has(n))) {
      return { ok: false, reason };
    }
  }
  if (filter.none_of && filter.none_of.length > 0) {
    for (const n of filter.none_of) {
      if (has(n)) return { ok: false, reason };
    }
  }
  return { ok: true, reason: '' };
};

// Implicit KB requirement of quick-answer / RAG mode: a vector or keyword index.
// Kept decoupled from `activeAgentTypePreset` because quick-answer has no agent_type, so the preset
// chain is always null there, yet a wiki-only KB always retrieves nothing in RAG mode and must be
// disabled with a hint rather than left silently useless.
const kbSatisfiesQuickAnswerMode = (kb: { capabilities?: KBCapabilities; ragEnabled?: boolean }): { ok: boolean; reason: string } => {
  if (agentMode.value !== 'quick-answer') return { ok: true, reason: '' };
  const hasRag = kb.capabilities
    ? (!!kb.capabilities.vector || !!kb.capabilities.keyword)
    : !!kb.ragEnabled;
  if (hasRag) return { ok: true, reason: '' };
  return { ok: false, reason: 'Quick Answer mode requires RAG retrieval' };
};

const filteredKbOptionsForPreset = computed(() => {
  const preset = activeAgentTypePreset.value;
  return kbOptions.value.map(kb => {
    const presetResult = kbSatisfiesPresetFilter(kb, preset);
    const modeResult = kbSatisfiesQuickAnswerMode(kb);
    const ok = presetResult.ok && modeResult.ok;
    const reason = !presetResult.ok ? presetResult.reason : (!modeResult.ok ? modeResult.reason : '');
    return { ...kb, disabled: !ok, disabledReason: reason };
  });
});
const filteredMyKbOptions = computed(() => filteredKbOptionsForPreset.value.filter(kb => !kb.shared));
const filteredSharedKbOptions = computed(() => filteredKbOptionsForPreset.value.filter(kb => kb.shared));

// How many of the currently selected KBs would be disabled by the new preset / mode (used to warn
// before saving). In quick-answer mode the preset is always null but a wiki-only KB still counts as
// disabled, so this looks at the disabled selected options instead of the preset.
const incompatibleSelectedKbCount = computed(() => {
  if (kbSelectionMode.value !== 'selected') return 0;
  const selected = new Set(formData.value.config.knowledge_bases || []);
  return filteredKbOptionsForPreset.value.filter(kb => selected.has(kb.value) && kb.disabled).length;
});

const applyAgentTypePreset = (preset: AgentTypePreset | null) => {
  if (!preset || !preset.config) return;
  const c = preset.config;
  const target = formData.value.config;
  if (c.system_prompt_id !== undefined) {
    target.system_prompt_id = c.system_prompt_id;
    const tmpl = agentSystemPromptTemplates.value.find(t => t.id === c.system_prompt_id);
    if (tmpl && typeof tmpl.content === 'string') {
      target.system_prompt = tmpl.content;
    } else {
      target.system_prompt = '';
      if (c.system_prompt_id) {
        console.warn(`[AgentType] system_prompt_id "${c.system_prompt_id}" not found in agent_system_prompt templates`);
      }
    }
  }
  if (typeof c.temperature === 'number') target.temperature = c.temperature;
  if (typeof c.max_iterations === 'number') target.max_iterations = c.max_iterations;
  if (Array.isArray(c.allowed_tools)) target.allowed_tools = normalizeLegacyToolNames(c.allowed_tools);
  if (typeof c.retain_retrieval_history === 'boolean') target.retain_retrieval_history = c.retain_retrieval_history;
  if (typeof c.faq_priority_enabled === 'boolean') target.faq_priority_enabled = c.faq_priority_enabled;
  if (typeof c.web_search_enabled === 'boolean') target.web_search_enabled = c.web_search_enabled;
  // supported_file_types uses hard-sync semantics: only data-analysis restricts it to csv/xlsx, and
  // every other type must clear it, otherwise leftovers carry over from the previous type.
  if (Array.isArray(c.supported_file_types)) {
    target.supported_file_types = [...c.supported_file_types];
  } else {
    target.supported_file_types = [];
  }
  // Sync kb_selection_mode to both formData and the UI state, or the radio buttons will not update.
  if (c.kb_selection_mode) {
    target.kb_selection_mode = c.kb_selection_mode;
    kbSelectionMode.value = c.kb_selection_mode;
  }
};

const onAgentTypeChange = (val: AgentType) => {
  // Capture whether name/description may be safely overwritten: anything the user edited (a value
  // that matches no preset default) is never overwritten.
  const canOverrideName = isNameSystemGenerated(formData.value.name);
  const canOverrideDesc = isDescriptionSystemGenerated(formData.value.description);

  agentType.value = val;
  const preset = agentTypePresets.value.find(p => p.id === val) || null;
  if (val !== 'custom') {
    applyAgentTypePreset(preset);
  }

  if (canOverrideName) {
    formData.value.name = getPresetDefaultName(preset);
  }
  if (canOverrideDesc) {
    formData.value.description = getPresetDefaultDescription(preset);
  }

  if (incompatibleSelectedKbCount.value > 0) {
    MessagePlugin.warning(
      `${incompatibleSelectedKbCount.value} selected KB(s) are not compatible with this type, please adjust manually.`,
      4000,
    );
  }
};

// reasoning_effort is the source of truth; legacy data with only the thinking boolean maps
// true->auto / false->off. The boolean is kept in sync on write so older backends still work.
const selectedChatModel = computed(() =>
  allModels.value.find(model => model.id === formData.value.config.model_id),
);
const selectedChatModelCanThink = computed(() => modelCanThink(selectedChatModel.value?.capabilities));
// Hint for models that cannot turn thinking off (deepseek-reasoner / qwq-plus / gemini-3 and
// similar), otherwise the missing "Off" entry in the dropdown looks like a bug.
const selectedChatModelAlwaysThinks = computed(
  () => modelCannotDisableThinking(selectedChatModel.value?.capabilities),
);
// When the catalog reports capabilities, follow thinking_levels exactly (including the fact that
// there is no off). Models without capabilities (local / Ollama / list not loaded) use the generic ladder.
const reasoningEffortOptions = computed<ReasoningLevel[]>(() => optionsFor(selectedChatModel.value?.capabilities));
const reasoningEffortLevel = computed<ReasoningLevel>({
  get: () => levelFromLegacy(formData.value.config.thinking, formData.value.config.reasoning_effort),
  set: (level: ReasoningLevel) => {
    formData.value.config.reasoning_effort = level;
    formData.value.config.thinking = levelEnablesThinking(level);
  },
});
// The stored level may not be in the selected model's option set (the model changed, or an older
// agent was loaded): clamp it and keep the thinking boolean in sync through the setter, so the UI
// never shows "Off" while the backend sends no switch at all and the model keeps thinking.
//
// Clamp only once the model actually resolves: while the model list is loading, capabilities are
// undefined and the generic ladder would downgrade a saved max/xhigh to auto.
const clampReasoningEffortToModel = () => {
  if (editorInitializing.value || !selectedChatModel.value) return;
  const clamped = clampLevel(reasoningEffortLevel.value, reasoningEffortOptions.value);
  if (clamped !== reasoningEffortLevel.value) reasoningEffortLevel.value = clamped;
};
watch(
  () => [
    editorInitializing.value,
    formData.value.config.model_id,
    reasoningEffortOptions.value.join(','),
  ].join('|'),
  () => clampReasoningEffortToModel(),
  { immediate: true },
);

const isBuiltinAgent = computed(() => {
  return formData.value.is_builtin === true;
});

const systemPromptPlaceholder = computed(() => {
  return 'Custom system prompt to define agent behavior and role (use {\'{{\'}web_search_status{\'}}\'} placeholder for dynamic web search behavior)';
});

const contextTemplatePlaceholder = computed(() => {
  return 'Custom context template...';
});

const needsRerankModel = computed(() => {
  if (!hasKnowledgeBase.value) return false;
  const mode = kbSelectionMode.value;
  if (mode === 'all') {
    return kbOptions.value.some(kb => kb.ragEnabled);
  }
  if (mode === 'selected') {
    const selectedIds = formData.value.config.knowledge_bases || [];
    return kbOptions.value.some(kb => selectedIds.includes(kb.value) && kb.ragEnabled);
  }
  return false;
});

let editorInitializationGeneration = 0;

watch(() => props.visible, async (val) => {
  const generation = ++editorInitializationGeneration;
  if (val) {
    editorInitializing.value = true;
    try {
    savedAgent.value = null;
    currentSection.value = resolveEditorSection(props.initialSection);
    await loadDependencies();
    if (generation !== editorInitializationGeneration || !props.visible) return;

    if (props.mode === 'edit' && props.agent) {
      const agentData = JSON.parse(JSON.stringify(props.agent));

      if (!agentData.config) {
        agentData.config = JSON.parse(JSON.stringify(defaultFormData.config));
      }

      agentData.config = { ...defaultFormData.config, ...agentData.config };
      if (agentData.config.thinking == null) {
        agentData.config.thinking = false;
      }
      // Legacy rows carry only the boolean: derive the graded level once so
      // the selector and the persisted config agree (true → auto, false → off).
      agentData.config.reasoning_effort = levelFromLegacy(agentData.config.thinking, agentData.config.reasoning_effort);
      agentData.config.thinking = levelEnablesThinking(agentData.config.reasoning_effort);

      agentData.config.question_suggestions = {
        starters: {
          ...defaultFormData.config.question_suggestions.starters,
          ...(agentData.config.question_suggestions?.starters || {}),
          items: agentData.config.question_suggestions?.starters?.items || [],
        },
        follow_ups: {
          ...defaultFormData.config.question_suggestions.follow_ups,
          ...(agentData.config.question_suggestions?.follow_ups || {}),
          categories: agentData.config.question_suggestions?.follow_ups?.categories
            || [...defaultFormData.config.question_suggestions.follow_ups.categories],
        },
      };
      if (!agentData.config.knowledge_bases) agentData.config.knowledge_bases = [];
      // Old configs may still carry the merged tool names (knowledge_search / grep_chunks /
      // list_knowledge_chunks / get_document_info / wiki_read_source_doc); map them onto the new
      // names and dedupe, otherwise the checkboxes will not match allTools.
      agentData.config.allowed_tools = normalizeLegacyToolNames(agentData.config.allowed_tools);
      if (!agentData.config.mcp_services) agentData.config.mcp_services = [];
      if (agentData.config.mcp_auth_wait_timeout == null || agentData.config.mcp_auth_wait_timeout <= 0) {
        agentData.config.mcp_auth_wait_timeout = 600;
      }
      if (!agentData.config.selected_skills) agentData.config.selected_skills = [];
      if (!agentData.config.supported_file_types) agentData.config.supported_file_types = [];
      if (!agentData.config.chat_parser_engine_rules) agentData.config.chat_parser_engine_rules = [];
      if (agentData.config.attachment_ocr_max_pages == null) agentData.config.attachment_ocr_max_pages = 0;
      if (agentData.config.attachment_parse_wait_timeout_sec == null) agentData.config.attachment_parse_wait_timeout_sec = 0;
      if (agentData.config.max_completion_tokens == null) agentData.config.max_completion_tokens = 0;
      // Long-term memory uses omitempty on the backend, so agents that follow the workspace setting
      // omit the field. Without this default the switch would read as off and a casual save would
      // really disable memory.
      if (agentData.config.memory_enabled == null) agentData.config.memory_enabled = true;

      if (!agentData.config.agent_mode) {
        const isAgent = agentData.config.max_iterations < 0 || agentData.config.max_iterations > 1 || (agentData.config.allowed_tools && agentData.config.allowed_tools.length > 0);
        agentData.config.agent_mode = isAgent ? 'smart-reasoning' : 'quick-answer';
      }

      isInitializing.value = true;
      agentData.config = hydrateAgentPromptRefs(agentData.config, promptTemplates.value);
      formData.value = agentData;
      if (agentData.config.max_iterations > 1) {
        lastFiniteMaxIterations.value = agentData.config.max_iterations;
      }
      initKbSelectionMode();
      initMcpSelectionMode();
      initSkillsSelectionMode();
      nextTick(() => {
        isInitializing.value = false;
      });
      // Display inherited defaults for all agents without persisting a copy.
      fillBuiltinAgentDefaults();
      void loadAgentIntegrationCounts(agentData.id);
    } else {
      const newFormData = JSON.parse(JSON.stringify(defaultFormData));
      newFormData.config.embedding_top_k = defaultEmbeddingTopK.value;
      newFormData.config.keyword_threshold = defaultKeywordThreshold.value;
      newFormData.config.vector_threshold = defaultVectorThreshold.value;
      newFormData.config.rerank_top_k = defaultRerankTopK.value;
      newFormData.config.rerank_threshold = defaultRerankThreshold.value;
      newFormData.config.max_completion_tokens = 0;
      newFormData.config.temperature = defaultTemperature.value;
      const isAgent = newFormData.config.agent_mode === 'smart-reasoning';
      if (isAgent) {
        if (defaultAgentSystemPrompt.value) {
          newFormData.config.system_prompt = defaultAgentSystemPrompt.value;
        }
      } else {
        if (defaultNormalSystemPrompt.value) {
          newFormData.config.system_prompt = defaultNormalSystemPrompt.value;
        }
        if (defaultContextTemplate.value) {
          newFormData.config.context_template = defaultContextTemplate.value;
        }
        if (defaultRewritePromptSystem.value) {
          newFormData.config.rewrite_prompt_system = defaultRewritePromptSystem.value;
        }
        if (defaultRewritePromptUser.value) {
          newFormData.config.rewrite_prompt_user = defaultRewritePromptUser.value;
        }
        if (defaultFallbackPrompt.value) {
          newFormData.config.fallback_prompt = defaultFallbackPrompt.value;
        }
        if (defaultFallbackResponse.value) {
          newFormData.config.fallback_response = defaultFallbackResponse.value;
        }
      }
      formData.value = newFormData;
      kbSelectionMode.value = 'all';
      mcpSelectionMode.value = 'none';
      skillsSelectionMode.value = 'none';

      // Apply the default agent_type preset immediately when creating a smart-reasoning agent (to
      // fill system_prompt / allowed_tools / kb_selection_mode), otherwise the default form the user
      // sees when the modal opens would not match the type shown in the dropdown.
      if (newFormData.config.agent_mode === 'smart-reasoning') {
        const defaultTypeId = newFormData.config.agent_type as AgentType;
        const preset = agentTypePresets.value.find(p => p.id === defaultTypeId) || null;
        if (defaultTypeId && defaultTypeId !== 'custom') {
          applyAgentTypePreset(preset);
        }
        if (!formData.value.name) {
          formData.value.name = getPresetDefaultName(preset);
        }
        if (!formData.value.description) {
          formData.value.description = getPresetDefaultDescription(preset);
        }
      }
      applyDefaultModelsIfEmpty()
    }

    await syncInstalledSkills()
    if (generation !== editorInitializationGeneration || !props.visible) return;

    if (props.initialHighlightField) {
      await applyInitialFieldHighlight(props.initialHighlightField);
      if (generation !== editorInitializationGeneration || !props.visible) return;
    }
    } catch (error) {
      console.error('Failed to initialize agent editor', error);
    } finally {
      if (generation === editorInitializationGeneration && props.visible) {
        editorInitializing.value = false;
        modalShell.markClean();
      }
    }
  } else {
    editorInitializing.value = false;
    clearFieldHighlight();
    agentIMChannelCount.value = 0;
    agentEmbedChannelCount.value = 0;
    showSkillProgress.value = false;
    skillProgressRecord.value = null;
    skillProgressId.value = '';
  }
});

const initKbSelectionMode = () => {
  if (formData.value.config.kb_selection_mode) {
    kbSelectionMode.value = formData.value.config.kb_selection_mode;
  } else if (formData.value.config.knowledge_bases?.length > 0) {
    kbSelectionMode.value = 'selected';
  } else {
    kbSelectionMode.value = 'none';
  }
};

const initMcpSelectionMode = () => {
  if (formData.value.config.mcp_selection_mode) {
    mcpSelectionMode.value = formData.value.config.mcp_selection_mode;
  } else if (formData.value.config.mcp_services?.length > 0) {
    mcpSelectionMode.value = 'selected';
  } else {
    mcpSelectionMode.value = 'none';
  }
};

const initSkillsSelectionMode = () => {
  if (formData.value.config.skills_selection_mode) {
    skillsSelectionMode.value = formData.value.config.skills_selection_mode;
  } else if (formData.value.config.selected_skills?.length > 0) {
    skillsSelectionMode.value = 'selected';
  } else {
    skillsSelectionMode.value = 'none';
  }
  autoBindSoleSandbox();
};

const fillBuiltinAgentDefaults = () => {
  const config = formData.value.config;
  const isAgent = config.agent_mode === 'smart-reasoning';

  if (isAgent) {
    if (!config.system_prompt && defaultAgentSystemPrompt.value) {
      config.system_prompt = defaultAgentSystemPrompt.value;
    }
  } else {
    if (!config.system_prompt && defaultNormalSystemPrompt.value) {
      config.system_prompt = defaultNormalSystemPrompt.value;
    }
    if (!config.context_template && defaultContextTemplate.value) {
      config.context_template = defaultContextTemplate.value;
    }
  }

  if (!config.rewrite_prompt_system && defaultRewritePromptSystem.value) {
    config.rewrite_prompt_system = defaultRewritePromptSystem.value;
  }
  if (!config.rewrite_prompt_user && defaultRewritePromptUser.value) {
    config.rewrite_prompt_user = defaultRewritePromptUser.value;
  }
  if (!config.fallback_prompt && defaultFallbackPrompt.value) {
    config.fallback_prompt = defaultFallbackPrompt.value;
  }
  if (!config.fallback_response && defaultFallbackResponse.value) {
    config.fallback_response = defaultFallbackResponse.value;
  }
};

watch(kbSelectionMode, (mode) => {
  formData.value.config.kb_selection_mode = mode;
  if (mode === 'none') {
    formData.value.config.knowledge_bases = [];
  } else if (mode === 'all') {
    formData.value.config.knowledge_bases = [];
  }
});

watch(mcpSelectionMode, (mode) => {
  formData.value.config.mcp_selection_mode = mode;
  if (mode === 'none') {
    formData.value.config.mcp_services = [];
  } else if (mode === 'all') {
    formData.value.config.mcp_services = [];
  }
});

watch(() => formData.value.config.sandbox_config_id, async () => {
  if (!props.visible) return
  await syncInstalledSkills()
})

let catalogPollTimer: number | null = null

function stopCatalogPoll() {
  if (catalogPollTimer != null) {
    window.clearInterval(catalogPollTimer)
    catalogPollTimer = null
  }
}

watch(
  [() => props.visible, catalogSkillRows],
  () => {
    const busy = catalogSkillRows.value.some((skill) =>
      skill.installStatus === 'installing' || skill.installStatus === 'removing',
    )
    if (!props.visible || !busy) {
      stopCatalogPoll()
      return
    }
    if (catalogPollTimer != null) return
    catalogPollTimer = window.setInterval(() => {
      void syncInstalledSkills(true)
    }, 2500)
  },
  { flush: 'post' },
)

watch(skillsSelectionMode, (mode) => {
  formData.value.config.skills_selection_mode = mode;
  if (mode === 'none') {
    formData.value.config.selected_skills = [];
  } else if (mode === 'all') {
    formData.value.config.selected_skills = [];
    autoBindSoleSandbox()
  } else {
    autoBindSoleSandbox()
  }
});

watch(agentMode, (val, _oldVal) => {
  if (val === 'smart-reasoning') {
    if (formData.value.config.allowed_tools.length === 0) {
      const tools: string[] = [];
      if (hasRagKnowledgeBase.value) {
        tools.push(...knowledgeBaseTools);
      }
      if (hasWikiKnowledgeBase.value) {
        tools.push(...wikiReadTools);
      }
      formData.value.config.allowed_tools = tools;
    }
    if (formData.value.config.max_iterations >= 0 && formData.value.config.max_iterations <= 1) {
      formData.value.config.max_iterations = 10;
    }
    if (defaultAgentSystemPrompt.value) {
      const isDefaultNormalPrompt = formData.value.config.system_prompt === defaultNormalSystemPrompt.value;
      if (!formData.value.config.system_prompt || isDefaultNormalPrompt) {
        formData.value.config.system_prompt = defaultAgentSystemPrompt.value;
      }
    }
  } else {
    formData.value.config.allowed_tools = [];
    formData.value.config.max_iterations = 1; // 1 means a single-round RAG turn
    if (defaultNormalSystemPrompt.value) {
      const isDefaultAgentPrompt = formData.value.config.system_prompt === defaultAgentSystemPrompt.value;
      if (!formData.value.config.system_prompt || isDefaultAgentPrompt) {
        formData.value.config.system_prompt = defaultNormalSystemPrompt.value;
      }
    }
    if (!formData.value.config.context_template && defaultContextTemplate.value) {
      formData.value.config.context_template = defaultContextTemplate.value;
    }
    if (!formData.value.config.rewrite_prompt_system && defaultRewritePromptSystem.value) {
      formData.value.config.rewrite_prompt_system = defaultRewritePromptSystem.value;
    }
    if (!formData.value.config.rewrite_prompt_user && defaultRewritePromptUser.value) {
      formData.value.config.rewrite_prompt_user = defaultRewritePromptUser.value;
    }
    if (!formData.value.config.fallback_prompt && defaultFallbackPrompt.value) {
      formData.value.config.fallback_prompt = defaultFallbackPrompt.value;
    }
    if (!formData.value.config.fallback_response && defaultFallbackResponse.value) {
      formData.value.config.fallback_response = defaultFallbackResponse.value;
    }
  }
});

// Watch knowledge-base capability changes:
//   - none -> some: seed the basic RAG tools so the agent works out of the box (seed only);
//   - some -> none: tools are NOT erased any more. Unmet dependencies are greyed out by
//     `availableTools` and filtered by the runtime tool registry. `allowed_tools` represents user
//     intent and should change only on an explicit user action (agent_type, agent_mode, a checkbox).
// History: older versions erased KB/Wiki tools when the KB capability disappeared, silently losing
// tools while the user had switched `kb_selection_mode` to "selected" but not yet picked a KB, which
// was fatal for the built-in wiki agent whose default tools are all wiki_*.
watch(hasKnowledgeBase, (hasKB, oldHasKB) => {
  if (!hasKB && currentSection.value === 'retrieval') {
    currentSection.value = 'basic';
  }

  if (isInitializing.value || !isAgentMode.value) return;

  if (hasKB && !oldHasKB) {
    const currentTools = formData.value.config.allowed_tools || [];
    const toolsToAdd = knowledgeBaseTools.filter((tool: string) => !currentTools.includes(tool));
    formData.value.config.allowed_tools = [...currentTools, ...toolsToAdd];
  }
});

watch(isAgentMode, (isAgent) => {
  if (isAgent && currentSection.value === 'advanced') {
    currentSection.value = 'basic';
  }
  if (!isAgent && (currentSection.value === 'skills' || currentSection.value === 'sandbox')) {
    currentSection.value = 'basic';
  }
});

watch(() => uiStore.showSettingsModal, async (visible, prevVisible) => {
  if (prevVisible && !visible && props.visible) {
    try {
      await Promise.all([
        chatResources.ensureModels(true),
        editorResources.ensureStorageEngine(true),
        chatResources.ensureSandboxConfigs(true),
      ]);
      if (chatResources.allModels.length > 0) {
        allModels.value = chatResources.allModels;
      }
      if (editorResources.storageStatus.length > 0) {
        storageEngineStatus.value = editorResources.storageStatus;
      }
      await syncInstalledSkills(true);
    } catch (e) {
      console.warn('Failed to refresh data after settings closed', e);
    }
  }
});

watch(() => chatResources.allModels, (list) => {
  if (props.visible) {
    allModels.value = list;
  }
});

const mapKbToOption = (kb: any, shared: boolean, orgName?: string) => {
  const strategy = kb.indexing_strategy;
  const caps: KBCapabilities | undefined = kb.capabilities;
  return {
    label: kb.name,
    value: kb.id,
    type: kb.type || 'document',
    count: kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count || 0),
    shared,
    orgName,
    ragEnabled: caps ? (caps.vector || caps.keyword) : (!strategy || strategy.vector_enabled || strategy.keyword_enabled),
    wikiEnabled: caps ? caps.wiki : (strategy?.wiki_enabled || false),
    capabilities: caps,
  };
};

const applyPromptTemplateDefaults = (cfg: PromptTemplatesConfig | null) => {
  promptTemplates.value = cfg;
  if (!cfg) return;
  if (cfg.agent_system_prompt && Array.isArray(cfg.agent_system_prompt)) {
    agentSystemPromptTemplates.value = cfg.agent_system_prompt;
    const ragDefault =
      cfg.agent_system_prompt.find(t => t.mode === 'rag' && t.default) ||
      cfg.agent_system_prompt.find(t => t.mode === 'rag');
    if (ragDefault?.content) {
      defaultAgentSystemPrompt.value = ragDefault.content;
    }
  }
  const pickDefault = (arr?: PromptTemplate[]): PromptTemplate | undefined =>
    Array.isArray(arr) ? arr.find(t => t.default) : undefined;
  const sysPrompt = pickDefault(cfg.system_prompt);
  if (sysPrompt?.content) defaultNormalSystemPrompt.value = sysPrompt.content;
  const ctxTmpl = pickDefault(cfg.context_template);
  if (ctxTmpl?.content) defaultContextTemplate.value = ctxTmpl.content;
  const rewriteTmpl = pickDefault(cfg.rewrite);
  if (rewriteTmpl?.content) defaultRewritePromptSystem.value = rewriteTmpl.content;
  if (rewriteTmpl?.user) defaultRewritePromptUser.value = rewriteTmpl.user;
  const fallbackList = Array.isArray(cfg.fallback) ? cfg.fallback : [];
  const fixedFallback = fallbackList.find(t => t.default && t.mode !== 'model');
  if (fixedFallback?.content) defaultFallbackResponse.value = fixedFallback.content;
  const modelFallback = fallbackList.find(t => t.mode === 'model' && t.default) || fallbackList.find(t => t.mode === 'model');
  if (modelFallback?.content) defaultFallbackPrompt.value = modelFallback.content;
  if (Array.isArray(cfg.intent_prompts)) {
    intentPromptTemplates.value = cfg.intent_prompts;
  }
};

const loadDependencies = async () => {
  try {
    await Promise.all([
      chatResources.ensureModels(),
      chatResources.ensureKnowledgeBases(),
      chatResources.ensureWebSearchProviders(),
      chatResources.ensureSandboxConfigs(),
      editorResources.prefetchAgentEditorDeps(),
    ]);

    if (chatResources.allModels.length > 0) {
      allModels.value = chatResources.allModels;
    }

    const myKbs = chatResources.rawKnowledgeBases.map((kb: any) => mapKbToOption(kb, false));
    const myKbIds = new Set(myKbs.map(kb => kb.value));
    const sharedKbs = (orgStore.sharedKnowledgeBases || [])
      .filter((shared: any) => shared.knowledge_base && !myKbIds.has(shared.knowledge_base.id))
      .map((shared: any) => mapKbToOption(shared.knowledge_base, true, shared.org_name));
    kbOptions.value = [...myKbs, ...sharedKbs];

    agentTypePresets.value = editorResources.agentTypePresets as AgentTypePreset[];
    applyPromptTemplateDefaults(editorResources.promptTemplates);

    storageEngineStatus.value = editorResources.storageStatus;

    webSearchProviderList.value = chatResources.webSearchProviders as WebSearchProviderEntity[];

    if (editorResources.placeholders) {
      placeholderData.value = editorResources.placeholders;
    }

    const rc = editorResources.tenantRetrievalConfig as Record<string, number> | null;
    if (rc?.embedding_top_k) defaultEmbeddingTopK.value = rc.embedding_top_k;
    if (rc?.keyword_threshold !== undefined) defaultKeywordThreshold.value = rc.keyword_threshold;
    if (rc?.vector_threshold !== undefined) defaultVectorThreshold.value = rc.vector_threshold;
    if (rc?.rerank_top_k) defaultRerankTopK.value = rc.rerank_top_k;
    if (rc?.rerank_threshold !== undefined) defaultRerankThreshold.value = rc.rerank_threshold;
  } catch (e) {
    console.error('Failed to load dependencies', e);
  }
};

const handleAddModel = (subSection: string) => {
  uiStore.openSettings('models', subSection);
};

const handleClose = () => {
  showPlaceholderPopup.value = false;
  showContextPlaceholderPopup.value = false;
  intentPromptPopup.value.show = false;
  rewriteSystemPopup.value.show = false;
  rewriteUserPopup.value.show = false;
  fallbackPromptPopup.value.show = false;
  emit('update:visible', false);
};

const modalShell = useModalShell({
  visible: () => props.visible,
  close: handleClose,
  snapshot: () => formData.value,
  ignoreEscape: () =>
    showPlaceholderPopup.value ||
    showContextPlaceholderPopup.value ||
    intentPromptPopup.value.show ||
    rewriteSystemPopup.value.show ||
    rewriteUserPopup.value.show ||
    fallbackPromptPopup.value.show,
});

const filteredPlaceholders = computed(() => {
  if (!placeholderPrefix.value) {
    return availablePlaceholders.value;
  }
  const prefix = placeholderPrefix.value.toLowerCase();
  return availablePlaceholders.value.filter(p =>
    p.name.toLowerCase().startsWith(prefix)
  );
});

const filteredContextPlaceholders = computed(() => {
  if (!contextPlaceholderPrefix.value) {
    return contextTemplatePlaceholders.value;
  }
  const prefix = contextPlaceholderPrefix.value.toLowerCase();
  return contextTemplatePlaceholders.value.filter(p =>
    p.name.toLowerCase().startsWith(prefix)
  );
});

const filteredRewriteSystemPlaceholders = computed(() => {
  if (!rewriteSystemPopup.value.prefix) {
    return rewriteSystemPlaceholders.value;
  }
  const prefix = rewriteSystemPopup.value.prefix.toLowerCase();
  return rewriteSystemPlaceholders.value.filter(p =>
    p.name.toLowerCase().startsWith(prefix)
  );
});

const filteredRewriteUserPlaceholders = computed(() => {
  if (!rewriteUserPopup.value.prefix) {
    return rewritePlaceholders.value;
  }
  const prefix = rewriteUserPopup.value.prefix.toLowerCase();
  return rewritePlaceholders.value.filter(p =>
    p.name.toLowerCase().startsWith(prefix)
  );
});

const filteredFallbackPlaceholders = computed(() => {
  if (!fallbackPromptPopup.value.prefix) {
    return fallbackPlaceholders.value;
  }
  const prefix = fallbackPromptPopup.value.prefix.toLowerCase();
  return fallbackPlaceholders.value.filter(p =>
    p.name.toLowerCase().startsWith(prefix)
  );
});

const getTextareaElement = (): HTMLTextAreaElement | null => {
  if (promptTextareaRef.value) {
    if (promptTextareaRef.value.$el) {
      return promptTextareaRef.value.$el.querySelector('textarea');
    }
    if (promptTextareaRef.value instanceof HTMLTextAreaElement) {
      return promptTextareaRef.value;
    }
  }
  return null;
};

const calculateCursorPosition = (textarea: HTMLTextAreaElement) => {
  const cursorPos = textarea.selectionStart;
  const textBeforeCursor = formData.value.config.system_prompt.substring(0, cursorPos);

  const style = window.getComputedStyle(textarea);
  // Placeholder popup is `position: fixed` under the root zoom; normalize the
  // visual-pixel rect to CSS pixels so the popup actually lands on the caret.
  const textareaRect = rectToCssPx(textarea.getBoundingClientRect(), getRootZoom());

  const lineHeight = parseFloat(style.lineHeight) || 20;
  const paddingTop = parseFloat(style.paddingTop) || 0;
  const paddingLeft = parseFloat(style.paddingLeft) || 0;

  const lines = textBeforeCursor.split('\n');
  const currentLine = lines.length - 1;
  const currentLineText = lines[currentLine];

  const span = document.createElement('span');
  span.style.font = style.font;
  span.style.visibility = 'hidden';
  span.style.position = 'absolute';
  span.style.whiteSpace = 'pre';
  span.textContent = currentLineText;
  document.body.appendChild(span);
  const textWidth = span.offsetWidth;
  document.body.removeChild(span);

  const scrollTop = textarea.scrollTop;
  const top = textareaRect.top + paddingTop + (currentLine * lineHeight) - scrollTop + lineHeight + 4;
  const scrollLeft = textarea.scrollLeft;
  const left = textareaRect.left + paddingLeft + textWidth - scrollLeft;

  return { top, left };
};

const checkAndShowPlaceholderPopup = () => {
  const textarea = getTextareaElement();
  if (!textarea) return;

  const cursorPos = textarea.selectionStart;
  const textBeforeCursor = formData.value.config.system_prompt.substring(0, cursorPos);

  let lastOpenPos = -1;
  for (let i = textBeforeCursor.length - 1; i >= 1; i--) {
    if (textBeforeCursor[i] === '{' && textBeforeCursor[i - 1] === '{') {
      const textAfterOpen = textBeforeCursor.substring(i + 1);
      if (!textAfterOpen.includes('}}')) {
        lastOpenPos = i - 1;
        break;
      }
    }
  }

  if (lastOpenPos === -1) {
    showPlaceholderPopup.value = false;
    placeholderPrefix.value = '';
    return;
  }

  const textAfterOpen = textBeforeCursor.substring(lastOpenPos + 2);
  placeholderPrefix.value = textAfterOpen;

  const filtered = filteredPlaceholders.value;
  if (filtered.length > 0) {
    nextTick(() => {
      const position = calculateCursorPosition(textarea);
      popupStyle.value = {
        top: `${position.top}px`,
        left: `${position.left}px`
      };
      showPlaceholderPopup.value = true;
      selectedPlaceholderIndex.value = 0;
    });
  } else {
    showPlaceholderPopup.value = false;
  }
};

const handlePromptInput = () => {
  if (placeholderPopupTimer) {
    clearTimeout(placeholderPopupTimer);
  }
  placeholderPopupTimer = setTimeout(() => {
    checkAndShowPlaceholderPopup();
  }, 50);
};

const insertPlaceholder = (placeholderName: string, fromPopup: boolean = false) => {
  const textarea = getTextareaElement();
  if (!textarea) return;

  showPlaceholderPopup.value = false;
  placeholderPrefix.value = '';
  selectedPlaceholderIndex.value = 0;

  nextTick(() => {
    const cursorPos = textarea.selectionStart;
    const currentValue = formData.value.config.system_prompt || '';
    const textBeforeCursor = currentValue.substring(0, cursorPos);
    const textAfterCursor = currentValue.substring(cursorPos);

    if (fromPopup) {
      let lastOpenPos = -1;
      for (let i = textBeforeCursor.length - 1; i >= 1; i--) {
        if (textBeforeCursor[i] === '{' && textBeforeCursor[i - 1] === '{') {
          lastOpenPos = i - 1;
          break;
        }
      }

      if (lastOpenPos !== -1) {
        const textBeforeOpen = currentValue.substring(0, lastOpenPos);
        const newValue = textBeforeOpen + `{{${placeholderName}}}` + textAfterCursor;
        formData.value.config.system_prompt = newValue;

        nextTick(() => {
          const newCursorPos = textBeforeOpen.length + placeholderName.length + 4;
          textarea.setSelectionRange(newCursorPos, newCursorPos);
          textarea.focus();
        });
        return;
      }
    }

    const newValue = textBeforeCursor + `{{${placeholderName}}}` + textAfterCursor;
    formData.value.config.system_prompt = newValue;

    nextTick(() => {
      const newCursorPos = cursorPos + placeholderName.length + 4;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
      textarea.focus();
    });
  });
};

const getContextTemplateTextareaElement = (): HTMLTextAreaElement | null => {
  if (contextTemplateTextareaRef.value) {
    if (contextTemplateTextareaRef.value.$el) {
      return contextTemplateTextareaRef.value.$el.querySelector('textarea');
    }
    if (contextTemplateTextareaRef.value instanceof HTMLTextAreaElement) {
      return contextTemplateTextareaRef.value;
    }
  }
  return null;
};

const calculateContextCursorPosition = (textarea: HTMLTextAreaElement) => {
  const cursorPos = textarea.selectionStart;
  const textBeforeCursor = formData.value.config.context_template.substring(0, cursorPos);

  const style = window.getComputedStyle(textarea);
  // See `calculateCursorPosition` for the zoom rationale.
  const textareaRect = rectToCssPx(textarea.getBoundingClientRect(), getRootZoom());

  const lineHeight = parseFloat(style.lineHeight) || 20;
  const paddingTop = parseFloat(style.paddingTop) || 0;
  const paddingLeft = parseFloat(style.paddingLeft) || 0;

  const lines = textBeforeCursor.split('\n');
  const currentLine = lines.length - 1;
  const currentLineText = lines[currentLine];

  const span = document.createElement('span');
  span.style.font = style.font;
  span.style.visibility = 'hidden';
  span.style.position = 'absolute';
  span.style.whiteSpace = 'pre';
  span.textContent = currentLineText;
  document.body.appendChild(span);
  const textWidth = span.offsetWidth;
  document.body.removeChild(span);

  const scrollTop = textarea.scrollTop;
  const top = textareaRect.top + paddingTop + (currentLine * lineHeight) - scrollTop + lineHeight + 4;
  const scrollLeft = textarea.scrollLeft;
  const left = textareaRect.left + paddingLeft + textWidth - scrollLeft;

  return { top, left };
};

const checkAndShowContextPlaceholderPopup = () => {
  const textarea = getContextTemplateTextareaElement();
  if (!textarea) return;

  const cursorPos = textarea.selectionStart;
  const textBeforeCursor = formData.value.config.context_template.substring(0, cursorPos);

  let lastOpenPos = -1;
  for (let i = textBeforeCursor.length - 1; i >= 1; i--) {
    if (textBeforeCursor[i] === '{' && textBeforeCursor[i - 1] === '{') {
      const textAfterOpen = textBeforeCursor.substring(i + 1);
      if (!textAfterOpen.includes('}}')) {
        lastOpenPos = i - 1;
        break;
      }
    }
  }

  if (lastOpenPos === -1) {
    showContextPlaceholderPopup.value = false;
    contextPlaceholderPrefix.value = '';
    return;
  }

  const textAfterOpen = textBeforeCursor.substring(lastOpenPos + 2);
  contextPlaceholderPrefix.value = textAfterOpen;

  const filtered = filteredContextPlaceholders.value;
  if (filtered.length > 0) {
    nextTick(() => {
      const position = calculateContextCursorPosition(textarea);
      contextPopupStyle.value = {
        top: `${position.top}px`,
        left: `${position.left}px`
      };
      showContextPlaceholderPopup.value = true;
      selectedContextPlaceholderIndex.value = 0;
    });
  } else {
    showContextPlaceholderPopup.value = false;
  }
};

const handleContextTemplateInput = () => {
  if (contextPlaceholderPopupTimer) {
    clearTimeout(contextPlaceholderPopupTimer);
  }
  contextPlaceholderPopupTimer = setTimeout(() => {
    checkAndShowContextPlaceholderPopup();
  }, 50);
};

const insertContextPlaceholder = (placeholderName: string, fromPopup: boolean = false) => {
  const textarea = getContextTemplateTextareaElement();
  if (!textarea) return;

  showContextPlaceholderPopup.value = false;
  contextPlaceholderPrefix.value = '';
  selectedContextPlaceholderIndex.value = 0;

  nextTick(() => {
    const cursorPos = textarea.selectionStart;
    const currentValue = formData.value.config.context_template || '';
    const textBeforeCursor = currentValue.substring(0, cursorPos);
    const textAfterCursor = currentValue.substring(cursorPos);

    if (fromPopup) {
      let lastOpenPos = -1;
      for (let i = textBeforeCursor.length - 1; i >= 1; i--) {
        if (textBeforeCursor[i] === '{' && textBeforeCursor[i - 1] === '{') {
          lastOpenPos = i - 1;
          break;
        }
      }

      if (lastOpenPos !== -1) {
        const textBeforeOpen = currentValue.substring(0, lastOpenPos);
        const newValue = textBeforeOpen + `{{${placeholderName}}}` + textAfterCursor;
        formData.value.config.context_template = newValue;

        nextTick(() => {
          const newCursorPos = textBeforeOpen.length + placeholderName.length + 4;
          textarea.setSelectionRange(newCursorPos, newCursorPos);
          textarea.focus();
        });
        return;
      }
    }

    const newValue = textBeforeCursor + `{{${placeholderName}}}` + textAfterCursor;
    formData.value.config.context_template = newValue;

    nextTick(() => {
      const newCursorPos = cursorPos + placeholderName.length + 4;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
      textarea.focus();
    });
  });
};

type GenericPlaceholderType = 'rewriteSystem' | 'rewriteUser' | 'fallback' | 'intent';

const genericPlaceholderFieldKeyMap: Record<Exclude<GenericPlaceholderType, 'intent'>, keyof typeof formData.value.config> = {
  rewriteSystem: 'rewrite_prompt_system',
  rewriteUser: 'rewrite_prompt_user',
  fallback: 'fallback_prompt',
};

const getGenericPlaceholderFieldValue = (type: GenericPlaceholderType): string => {
  if (type === 'intent') return intentEditorValue.value || '';
  return String(formData.value.config[genericPlaceholderFieldKeyMap[type]] || '');
};

const setGenericPlaceholderFieldValue = (type: GenericPlaceholderType, value: string) => {
  if (type === 'intent') {
    intentEditorValue.value = value;
    return;
  }
  (formData.value.config as any)[genericPlaceholderFieldKeyMap[type]] = value;
};

const getGenericTextareaElement = (type: GenericPlaceholderType): HTMLTextAreaElement | null => {
  const refMap = {
    rewriteSystem: rewriteSystemTextareaRef,
    rewriteUser: rewriteUserTextareaRef,
    fallback: fallbackPromptTextareaRef,
    intent: intentPromptTextareaRef,
  };
  const ref = refMap[type];
  if (ref.value) {
    if (ref.value.$el) {
      return ref.value.$el.querySelector('textarea');
    }
    if (ref.value instanceof HTMLTextAreaElement) {
      return ref.value;
    }
  }
  return null;
};

const calculateGenericCursorPosition = (textarea: HTMLTextAreaElement, fieldValue: string) => {
  const cursorPos = textarea.selectionStart;
  const textBeforeCursor = fieldValue.substring(0, cursorPos);
  const lines = textBeforeCursor.split('\n');
  const currentLine = lines.length - 1;
  const currentLineText = lines[currentLine];

  // See `calculateCursorPosition` for the zoom rationale.
  const textareaRect = rectToCssPx(textarea.getBoundingClientRect(), getRootZoom());
  const style = window.getComputedStyle(textarea);
  const lineHeight = parseFloat(style.lineHeight) || 20;
  const paddingTop = parseFloat(style.paddingTop) || 0;
  const paddingLeft = parseFloat(style.paddingLeft) || 0;

  const span = document.createElement('span');
  span.style.font = style.font;
  span.style.visibility = 'hidden';
  span.style.position = 'absolute';
  span.style.whiteSpace = 'pre';
  span.textContent = currentLineText;
  document.body.appendChild(span);
  const textWidth = span.offsetWidth;
  document.body.removeChild(span);

  const scrollTop = textarea.scrollTop;
  const top = textareaRect.top + paddingTop + (currentLine * lineHeight) - scrollTop + lineHeight + 4;
  const scrollLeft = textarea.scrollLeft;
  const left = textareaRect.left + paddingLeft + textWidth - scrollLeft;

  return { top, left };
};

const checkAndShowGenericPlaceholderPopup = (
  type: GenericPlaceholderType,
  popup: typeof rewriteSystemPopup,
  filteredPlaceholders: PlaceholderDefinition[]
) => {
  const textarea = getGenericTextareaElement(type);
  if (!textarea) return;

  const cursorPos = textarea.selectionStart;
  const fieldValue = getGenericPlaceholderFieldValue(type);
  const textBeforeCursor = fieldValue.substring(0, cursorPos);

  let lastOpenPos = -1;
  for (let i = textBeforeCursor.length - 1; i >= 1; i--) {
    if (textBeforeCursor[i] === '{' && textBeforeCursor[i - 1] === '{') {
      const textAfterOpen = textBeforeCursor.substring(i + 1);
      if (!textAfterOpen.includes('}}')) {
        lastOpenPos = i - 1;
        break;
      }
    }
  }

  if (lastOpenPos === -1) {
    popup.value.show = false;
    popup.value.prefix = '';
    return;
  }

  const textAfterOpen = textBeforeCursor.substring(lastOpenPos + 2);
  popup.value.prefix = textAfterOpen;

  if (filteredPlaceholders.length > 0) {
    nextTick(() => {
      const position = calculateGenericCursorPosition(textarea, fieldValue);
      popup.value.style = {
        top: `${position.top}px`,
        left: `${position.left}px`
      };
      popup.value.show = true;
      popup.value.selectedIndex = 0;
    });
  } else {
    popup.value.show = false;
  }
};

const handleRewriteSystemInput = () => {
  if (rewriteSystemPopup.value.timer) {
    clearTimeout(rewriteSystemPopup.value.timer);
  }
  rewriteSystemPopup.value.timer = setTimeout(() => {
    checkAndShowGenericPlaceholderPopup('rewriteSystem', rewriteSystemPopup, filteredRewriteSystemPlaceholders.value);
  }, 50);
};

const handleRewriteUserInput = () => {
  if (rewriteUserPopup.value.timer) {
    clearTimeout(rewriteUserPopup.value.timer);
  }
  rewriteUserPopup.value.timer = setTimeout(() => {
    checkAndShowGenericPlaceholderPopup('rewriteUser', rewriteUserPopup, filteredRewriteUserPlaceholders.value);
  }, 50);
};

const handleFallbackPromptInput = () => {
  if (fallbackPromptPopup.value.timer) {
    clearTimeout(fallbackPromptPopup.value.timer);
  }
  fallbackPromptPopup.value.timer = setTimeout(() => {
    checkAndShowGenericPlaceholderPopup('fallback', fallbackPromptPopup, filteredFallbackPlaceholders.value);
  }, 50);
};

const handleIntentPromptInput = () => {
  if (intentPromptPopup.value.timer) {
    clearTimeout(intentPromptPopup.value.timer);
  }
  intentPromptPopup.value.timer = setTimeout(() => {
    checkAndShowGenericPlaceholderPopup('intent', intentPromptPopup, filteredIntentPlaceholders.value);
  }, 50);
};

const insertGenericPlaceholder = (type: GenericPlaceholderType, placeholderName: string, fromPopup: boolean = false) => {
  const textarea = getGenericTextareaElement(type);
  if (!textarea) return;

  const popupMap = {
    rewriteSystem: rewriteSystemPopup,
    rewriteUser: rewriteUserPopup,
    fallback: fallbackPromptPopup,
    intent: intentPromptPopup,
  };

  const popup = popupMap[type];

  popup.value.show = false;
  popup.value.prefix = '';
  popup.value.selectedIndex = 0;

  nextTick(() => {
    const cursorPos = textarea.selectionStart;
    const currentValue = getGenericPlaceholderFieldValue(type);
    const textBeforeCursor = currentValue.substring(0, cursorPos);
    const textAfterCursor = currentValue.substring(cursorPos);

    if (fromPopup) {
      let lastOpenPos = -1;
      for (let i = textBeforeCursor.length - 1; i >= 1; i--) {
        if (textBeforeCursor[i] === '{' && textBeforeCursor[i - 1] === '{') {
          lastOpenPos = i - 1;
          break;
        }
      }

      if (lastOpenPos !== -1) {
        const textBeforeOpen = currentValue.substring(0, lastOpenPos);
        const newValue = textBeforeOpen + `{{${placeholderName}}}` + textAfterCursor;
        setGenericPlaceholderFieldValue(type, newValue);

        nextTick(() => {
          const newCursorPos = textBeforeOpen.length + placeholderName.length + 4;
          textarea.setSelectionRange(newCursorPos, newCursorPos);
          textarea.focus();
        });
        return;
      }
    }

    const newValue = textBeforeCursor + `{{${placeholderName}}}` + textAfterCursor;
    setGenericPlaceholderFieldValue(type, newValue);

    nextTick(() => {
      const newCursorPos = cursorPos + placeholderName.length + 4;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
      textarea.focus();
    });
  });
};

const setupContextTemplateEventListeners = () => {
  nextTick(() => {
    const textarea = getContextTemplateTextareaElement();
    if (textarea) {
      textarea.addEventListener('keydown', (e: KeyboardEvent) => {
        if (showContextPlaceholderPopup.value && filteredContextPlaceholders.value.length > 0) {
          if (e.key === 'ArrowDown') {
            e.preventDefault();
            e.stopPropagation();
            if (selectedContextPlaceholderIndex.value < filteredContextPlaceholders.value.length - 1) {
              selectedContextPlaceholderIndex.value++;
            } else {
              selectedContextPlaceholderIndex.value = 0;
            }
          } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            e.stopPropagation();
            if (selectedContextPlaceholderIndex.value > 0) {
              selectedContextPlaceholderIndex.value--;
            } else {
              selectedContextPlaceholderIndex.value = filteredContextPlaceholders.value.length - 1;
            }
          } else if (e.key === 'Enter' || e.key === 'Tab') {
            e.preventDefault();
            e.stopPropagation();
            const selected = filteredContextPlaceholders.value[selectedContextPlaceholderIndex.value];
            if (selected) {
              insertContextPlaceholder(selected.name, true);
            }
          } else if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            showContextPlaceholderPopup.value = false;
            contextPlaceholderPrefix.value = '';
          }
        }
      }, true);
    }
  });
};

const setupTextareaEventListeners = () => {
  nextTick(() => {
    const textarea = getTextareaElement();
    if (textarea) {
      textarea.addEventListener('keydown', (e: KeyboardEvent) => {
        if (showPlaceholderPopup.value && filteredPlaceholders.value.length > 0) {
          if (e.key === 'ArrowDown') {
            e.preventDefault();
            e.stopPropagation();
            if (selectedPlaceholderIndex.value < filteredPlaceholders.value.length - 1) {
              selectedPlaceholderIndex.value++;
            } else {
              selectedPlaceholderIndex.value = 0;
            }
          } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            e.stopPropagation();
            if (selectedPlaceholderIndex.value > 0) {
              selectedPlaceholderIndex.value--;
            } else {
              selectedPlaceholderIndex.value = filteredPlaceholders.value.length - 1;
            }
          } else if (e.key === 'Enter' || e.key === 'Tab') {
            e.preventDefault();
            e.stopPropagation();
            const selected = filteredPlaceholders.value[selectedPlaceholderIndex.value];
            if (selected) {
              insertPlaceholder(selected.name, true);
            }
          } else if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            showPlaceholderPopup.value = false;
            placeholderPrefix.value = '';
          }
        }
      }, true);
    }
  });
};

const setupGenericTextareaEventListeners = (
  type: GenericPlaceholderType,
  popup: typeof rewriteSystemPopup,
  filteredPlaceholders: () => PlaceholderDefinition[]
) => {
  nextTick(() => {
    const textarea = getGenericTextareaElement(type);
    if (textarea) {
      textarea.addEventListener('keydown', (e: KeyboardEvent) => {
        const filtered = filteredPlaceholders();
        if (popup.value.show && filtered.length > 0) {
          if (e.key === 'ArrowDown') {
            e.preventDefault();
            e.stopPropagation();
            if (popup.value.selectedIndex < filtered.length - 1) {
              popup.value.selectedIndex++;
            } else {
              popup.value.selectedIndex = 0;
            }
          } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            e.stopPropagation();
            if (popup.value.selectedIndex > 0) {
              popup.value.selectedIndex--;
            } else {
              popup.value.selectedIndex = filtered.length - 1;
            }
          } else if (e.key === 'Enter' || e.key === 'Tab') {
            e.preventDefault();
            e.stopPropagation();
            const selected = filtered[popup.value.selectedIndex];
            if (selected) {
              insertGenericPlaceholder(type, selected.name, true);
            }
          } else if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            popup.value.show = false;
            popup.value.prefix = '';
          }
        }
      }, true);
    }
  });
};

const handlePlaceholderClick = (type: 'system' | 'context' | 'rewriteSystem' | 'rewriteUser' | 'fallback' | 'intent', placeholderName: string) => {
  if (type === 'system') {
    insertPlaceholder(placeholderName);
  } else if (type === 'context') {
    insertContextPlaceholder(placeholderName);
  } else {
    insertGenericPlaceholder(type, placeholderName);
  }
};

watch(() => props.visible, (val) => {
  if (val) {
    nextTick(() => {
      setupTextareaEventListeners();
      setupContextTemplateEventListeners();
      setupGenericTextareaEventListeners('intent', intentPromptPopup, () => filteredIntentPlaceholders.value);
      setupGenericTextareaEventListeners('rewriteSystem', rewriteSystemPopup, () => filteredRewriteSystemPlaceholders.value);
      setupGenericTextareaEventListeners('rewriteUser', rewriteUserPopup, () => filteredRewriteUserPlaceholders.value);
      setupGenericTextareaEventListeners('fallback', fallbackPromptPopup, () => filteredFallbackPlaceholders.value);
    });
  }
});

const handleSystemPromptTemplateSelect = (template: PromptTemplate) => {
  formData.value.config.system_prompt = template.content;
  formData.value.config.system_prompt_id = template.id;
};

// "Reset to default" for the agent system prompt: when a non-custom agent type is selected, the
// default is the prompt bound to that type's preset (Wiki QA -> wiki_researcher, for example), not
// the globally default: true entry in the agent_system_prompt template table. Only a custom type, or
// a preset whose template cannot be found, falls back to the global default from PromptTemplateSelector.
const handleAgentSystemPromptResetDefault = (fallback: PromptTemplate) => {
  const typeId = agentType.value;
  if (typeId && typeId !== 'custom') {
    const preset = agentTypePresets.value.find(p => p.id === typeId);
    const presetPromptId = preset?.config?.system_prompt_id;
    if (presetPromptId) {
      const tmpl = agentSystemPromptTemplates.value.find(t => t.id === presetPromptId);
      if (tmpl && typeof tmpl.content === 'string') {
        formData.value.config.system_prompt = tmpl.content;
        formData.value.config.system_prompt_id = tmpl.id;
        return;
      }
    }
  }
  // Fallback: no suitable type preset, so use the global default found by PromptTemplateSelector
  formData.value.config.system_prompt = fallback.content;
  formData.value.config.system_prompt_id = fallback.id;
};

const handleContextTemplateSelect = (template: PromptTemplate) => {
  formData.value.config.context_template = template.content;
  formData.value.config.context_template_id = template.id;
};

const handleRewriteTemplateSelect = (template: PromptTemplate) => {
  // Rewrite templates contain both content (system) and user fields
  formData.value.config.rewrite_prompt_system = template.content;
  if (template.user) {
    formData.value.config.rewrite_prompt_user = template.user;
  }
};

const handleFallbackResponseTemplateSelect = (template: PromptTemplate) => {
  formData.value.config.fallback_response = template.content;
};

const handleFallbackPromptTemplateSelect = (template: PromptTemplate) => {
  formData.value.config.fallback_prompt = template.content;
};

const hasPlaceholder = (text: string | undefined, placeholder: string): boolean => {
  if (!text) return false;
  return text.includes(`{{${placeholder}}}`);
};

const handleSave = async () => {
  if (!isBuiltinAgent.value) {
    if (!formData.value.name || !formData.value.name.trim()) {
      MessagePlugin.error('Agent name is required');
      currentSection.value = 'basic';
      return;
    }

    if (!formData.value.config.system_prompt || !formData.value.config.system_prompt.trim()) {
      MessagePlugin.error('System prompt is required');
      currentSection.value = 'prompts';
      return;
    }

    if (!isAgentMode.value && (!formData.value.config.context_template || !formData.value.config.context_template.trim())) {
      MessagePlugin.error('Context template is required');
      currentSection.value = 'prompts';
      return;
    }
  }

  if (!isAgentMode.value && formData.value.config.multi_turn_enabled && formData.value.config.enable_rewrite) {
    const rewritePrompt = formData.value.config.rewrite_prompt_user || '';
    if (rewritePrompt.trim()) {
      if (!hasPlaceholder(rewritePrompt, 'query')) {
        MessagePlugin.error('Rewrite user prompt must contain {\'{{\'}query{\'}}\'} placeholder');
        currentSection.value = 'prompts';
        return;
      }
    }
  }

  if (!isAgentMode.value && formData.value.config.fallback_strategy === 'model') {
    const fallbackPrompt = formData.value.config.fallback_prompt || '';
    if (fallbackPrompt.trim() && !hasPlaceholder(fallbackPrompt, 'query')) {
      MessagePlugin.error('Fallback prompt must contain {\'{{\'}query{\'}}\'} placeholder');
      currentSection.value = 'prompts';
      return;
    }
  }

  if (!formData.value.config.model_id) {
    MessagePlugin.error('Please select a model');
    currentSection.value = 'model';
    return;
  }

  if (formData.value.config.image_upload_enabled && !formData.value.config.vlm_model_id) {
    MessagePlugin.error('VLM model is required when image upload is enabled');
    currentSection.value = 'multimodal';
    return;
  }

  // The ReRank model is used on demand by runtime scope: it is not needed when the knowledge base
  // scope is none or search_knowledge is disabled; otherwise the chat entry warns before use.

  formData.value.config.question_suggestions.starters.items =
    formData.value.config.question_suggestions.starters.items
      .map((p: string) => p.trim())
      .filter(Boolean);

  if (!formData.value.config.intent_prompts || Object.keys(formData.value.config.intent_prompts).length === 0) {
    delete formData.value.config.intent_prompts;
  }

  pruneSelectedSkills()

  const payload = { ...formData.value, config: serializeAgentPrompts(formData.value.config, promptTemplates.value) };
  saving.value = true;
  try {
    if (editorMode.value === 'create') {
      const result: any = await createAgent(payload);
      const created = result?.data as CustomAgent | undefined;
      if (!created?.id) {
        throw new Error(result?.message || 'Save failed');
      }
      savedAgent.value = created;
      formData.value.id = created.id;
      modalShell.markClean();
      markContextualGuideDone('agentCreate')
      currentSection.value = 'basic';
      void loadAgentIntegrationCounts(created.id);
      MessagePlugin.success('Agent created successfully');
      emit('success', created);
    } else {
      await updateAgent(formData.value.id, payload);
      MessagePlugin.success('Agent updated successfully');
      emit('success');
      handleClose();
    }
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Save failed');
  } finally {
    saving.value = false;
  }
};
</script>

<style scoped lang="less">
.section--prompts {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.prompts-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.prompts-panel__header {
  flex-shrink: 0;
  margin: 0 0 0;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.prompts-panel__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  margin: 0 -4px;
  padding: 4px 4px 8px;
}

.prompts-panel__pane {
  &.setting-row:last-child {
    border-bottom: none;
  }
}

.prompts-panel__pane--stack {
  .setting-row:last-child {
    border-bottom: none;
  }
}

.section-header--compact {
  margin-bottom: 0;

  h2 {
    margin-bottom: 4px;
  }

  .section-description {
    font-size: var(--app-text-md);
  }
}

.prompts-outline {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
  min-width: 0;

  &__pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 5px 12px;
    border: none;
    border-radius: var(--app-radius-sm);
    background: var(--td-bg-color-secondarycontainer);
    font: inherit;
    font-size: var(--app-text-md);
    line-height: 1.4;
    color: var(--td-text-color-secondary);
    cursor: pointer;
    transition: color var(--app-motion-fast) ease, background var(--app-motion-fast) ease;

    &:hover,
    &:focus-visible {
      color: var(--td-brand-color);
      background: color-mix(in srgb, var(--td-brand-color) 8%, var(--td-bg-color-secondarycontainer));
      outline: none;
    }

    &--active {
      background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
      color: var(--td-brand-color);
      font-weight: 500;
    }
  }

  &__dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--td-brand-color);
    flex-shrink: 0;
  }
}

.content-wrapper {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
  padding: 28px 40px 48px;
  box-sizing: border-box;
  scroll-padding-bottom: 24px;

  &--prompts {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding-bottom: 28px;
  }
}

.section {
  width: 100%;
  animation: sectionFadeIn 0.25s ease;
}

@keyframes sectionFadeIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.section-header {
  margin-bottom: 20px;

  .section-header-title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;

    h2 {
      margin: 0;
    }
  }

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

    .doc-link {
      margin-left: 8px;
    }
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.parser-policy-block {
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &__header {
    margin-bottom: 12px;

    label {
      display: block;
      font-size: var(--app-text-lg);
      font-weight: 500;
      color: var(--td-text-color-primary);
      margin-bottom: 4px;
    }

    .desc {
      margin: 0;
      font-size: var(--app-text-md);
      color: var(--td-text-color-secondary);
      line-height: 1.5;
    }
  }
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);
  min-width: 0;

  &:last-child {
    border-bottom: none;
  }

  &.setting-row-vertical {
    flex-direction: column;
    gap: 12px;

    .setting-info {
      max-width: 100%;
      padding-right: 0;
    }
  }

  &.setting-row--emphasize {
    position: relative;
    padding-left: 14px;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 18px;
      bottom: 18px;
      width: 3px;
      border-radius: 2px;
      background: var(--td-brand-color);
    }

    .setting-info label {
      font-weight: 600;
    }
  }

  &.setting-row--field-highlight {
    border-radius: var(--app-radius-sm);
    animation: agent-field-flash 0.8s ease-in-out 3;
  }
}

@keyframes agent-field-flash {
  0%,
  100% {
    background-color: transparent;
    box-shadow: none;
  }

  50% {
    background-color: var(--td-warning-color-light);
    box-shadow: inset 0 0 0 1px rgba(237, 123, 47, 0.35);
  }
}

.setting-info {
  flex: 0 0 42%;
  max-width: 42%;
  min-width: 0;
  padding-right: 0;

  &.full-width {
    max-width: 100%;
    padding-right: 0;
  }

  .setting-info-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;

    label {
      margin-bottom: 0;
    }
  }

  label {
    font-size: var(--app-text-lg);
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;

    .required {
      color: var(--td-error-color);
      margin-left: 2px;
    }
  }

  .desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;

    .hint {
      color: var(--td-warning-color);
    }
  }
}

.setting-control {
  flex: 1 1 58%;
  min-width: 0;
  max-width: 58%;
  display: flex;
  justify-content: flex-end;
  align-items: flex-start;
  overflow: hidden;

  .reasoning-effort-select {
    width: 100%;
    max-width: 220px;
  }

  &.setting-control-full {
    width: 100%;
    min-width: 100%;
    max-width: 100%;
    justify-content: flex-start;
  }

  &.max-tokens-control {
    flex-direction: column;
    align-items: flex-end;
    gap: 8px;

    :deep(.t-input-number) {
      width: 140px;
    }
  }

  :deep(.t-select),
  :deep(.t-input),
  :deep(.t-textarea) {
    width: 100%;
    min-width: 0;
  }

  :deep(.t-select-input) {
    min-width: 0;
  }

  :deep(.t-select .t-tag) {
    max-width: 160px;
  }

  :deep(.t-select .t-tag__text) {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :deep(.t-input-number) {
    width: 120px;
  }
}

.integration-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;

  &__stat {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);

    &.integration-inline__link {
      display: inline-flex;
      align-items: center;
      gap: 2px;
      padding: 0;
      border: none;
      background: transparent;
      line-height: 1;
      color: var(--td-brand-color);
      cursor: pointer;

      &:hover {
        opacity: 0.85;
      }
    }
  }

  &__sep {
    color: var(--td-component-stroke);
    font-size: var(--app-text-sm);
  }

  &__link {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    margin-left: 4px;
    padding: 0;
    border: none;
    background: transparent;
    font-size: var(--app-text-md);
    line-height: 1;
    color: var(--td-brand-color);
    cursor: pointer;

    &:hover {
      opacity: 0.85;
    }

    :deep(.t-icon) {
      display: block;
    }
  }
}

.select-option-with-tag {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 8px;
}

.go-settings-link {
  font-size: var(--app-text-sm);
  color: var(--td-brand-color);
  margin-top: 0;
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}

.sandbox-select-links {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}

.sandbox-select-links__sep {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
}

.sandbox-select-control {
  flex-direction: column;
  align-items: flex-end;
}

.sandbox-config-select {
  width: 280px;
}

.sandbox-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  min-width: 0;
}

.sandbox-option__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.sandbox-option__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sandbox-option__type {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
}

.sandbox-option__target {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.35;
}

.sandbox-selected-meta {
  margin: 6px 0 0;
  max-width: 280px;
  font-size: var(--app-text-sm);
  line-height: 1.45;
  color: var(--td-text-color-secondary);
  word-break: break-word;
}

.name-input-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;

  .name-input {
    flex: 1;
  }
}

.agent-id-field {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  padding: 6px 8px 6px 12px;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);

  .agent-id-value {
    flex: 1;
    min-width: 0;
    margin: 0;
    padding: 0;
    background: none;
    border: none;
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-md);
    line-height: 1.5;
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .agent-id-copy {
    flex-shrink: 0;
    color: var(--td-text-color-secondary);

    &:hover {
      color: var(--td-brand-color);
    }
  }
}

.content-wrapper::-webkit-scrollbar {
  width: 6px;
}

.content-wrapper::-webkit-scrollbar-track {
  background: var(--td-bg-color-container);
}

.content-wrapper::-webkit-scrollbar-thumb {
  background: var(--td-gray-color-5);
  border-radius: 3px;
}

.content-wrapper::-webkit-scrollbar-thumb:hover {
  background: var(--td-gray-color-6);
}

.mode-hint {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  background: var(--td-success-color-light);
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--td-success-color-focus);
  color: var(--td-brand-color);
  font-size: var(--app-text-md);
  line-height: 1.5;
}

.slider-wrapper {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;

  :deep(.t-slider) {
    flex: 1;
  }
}

.slider-value {
  width: 40px;
  text-align: right;
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-base);
  color: var(--td-text-color-primary);
}

.max-tokens-value {
  font-family: var(--app-font-family-mono);
  font-variant-numeric: tabular-nums;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
}

.suggested-prompts-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.prompt-item {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input) {
    flex: 1;
  }
}

.suggestion-tabs {
  margin-bottom: 4px;

  :deep(.t-tabs__nav-item) {
    font-size: var(--app-text-base);
  }

  :deep(.t-tabs__operations) {
    display: none;
  }

  :deep(.t-tabs__content) {
    display: none;
  }
}

.suggestion-advanced-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0 2px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 18px;

  &::before,
  &::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--td-component-stroke);
  }

  span {
    flex-shrink: 0;
    color: var(--td-text-color-secondary);
    font-weight: 500;
  }
}

// Keep the count badge next to the label so space-between does not push it away in a full-width row
// Needs the same specificity as the base `.setting-info .setting-info-header` in order to override it
.setting-info-header.setting-info-header--inline {
  justify-content: flex-start;
  gap: 8px;
}

.curated-items-count {
  flex-shrink: 0;
  padding: 0 8px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-secondarycontainer);
  font-size: var(--app-text-sm);
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-secondary);
}

.suggestion-checkboxes {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.tools-overview {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 12px;
  padding: 12px 14px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--td-component-stroke);
}

.tools-overview-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;

  &--preset {
    border-top: 1px dashed var(--td-component-stroke);
    padding-top: 10px;
    justify-content: space-between;
  }
}

.tools-status-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-pill);
  border: 1px solid var(--td-component-stroke);

  .t-icon {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-base);
  }

  .tools-status-metric {
    display: inline-flex;
    align-items: baseline;
    gap: 4px;

    strong {
      font-size: var(--app-text-base);
      font-weight: 600;
      color: var(--td-text-color-primary);
    }
  }

  .tools-status-sep {
    color: var(--td-text-color-placeholder);
  }

  &--warn {
    color: var(--td-warning-color);
    background: var(--td-warning-color-1);
    border-color: var(--td-warning-color-light);

    .t-icon {
      color: var(--td-warning-color);
    }
  }
}

.tool-groups {
  display: flex;
  flex-direction: column;
  gap: 22px;
  width: 100%;
}

.tool-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tool-group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 2px;

  .tool-group-bar {
    display: inline-block;
    width: 3px;
    height: 14px;
    border-radius: 2px;
    background: var(--td-brand-color);
  }

  .tool-group-title {
    font-size: var(--app-text-md);
    font-weight: 600;
    color: var(--td-text-color-primary);
    letter-spacing: 0.2px;
  }

  .tool-group-count {
    min-width: 20px;
    padding: 0 6px;
    font-size: var(--app-text-xs);
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
    border-radius: var(--app-radius-pill);
    text-align: center;
    line-height: 18px;
  }

  .tool-group-warning {
    margin-left: auto;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    font-size: var(--app-text-sm);
    color: var(--td-warning-color);
    background: var(--td-warning-color-1);
    border: 1px solid var(--td-warning-color-light);
    border-radius: var(--app-radius-pill);

    .t-icon {
      font-size: var(--app-text-md);
    }
  }
}

.tool-group--base .tool-group-bar {
  background: var(--td-gray-color-6);
}

.tool-group--rag .tool-group-bar {
  background: var(--td-brand-color);
}

.tool-group--wiki_read .tool-group-bar {
  background: var(--td-success-color);
}

.tool-group--wiki_edit .tool-group-bar {
  background: var(--td-warning-color);
}

.tool-group--wiki_issue .tool-group-bar {
  background: var(--td-purple-5, #8e56dd);
}

.tool-group--data .tool-group-bar {
  background: var(--td-cyan-6, #09a3b7);
}

.tool-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  width: 100%;

  @media (max-width: 720px) {
    grid-template-columns: 1fr;
  }
}

.tool-card {
  margin: 0;
  padding: 12px 14px;
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-md);
  border: 1px solid var(--td-component-stroke);
  transition: border-color var(--app-motion-base), background var(--app-motion-base);
  cursor: pointer;
  overflow: hidden;

  &:hover:not(.tool-card--disabled) {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-1);
  }

  :deep(.t-checkbox__input) {
    margin-top: 2px;
    flex-shrink: 0;
  }

  :deep(.t-checkbox__label) {
    flex: 1;
    min-width: 0;
    padding-left: 10px;
  }

  &.t-is-checked {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-1);
  }

  &--disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  &--danger {
    border-color: var(--td-warning-color-light);

    &:hover:not(.tool-card--disabled) {
      border-color: var(--td-warning-color);
      background: var(--td-warning-color-1);
    }

    &.t-is-checked {
      border-color: var(--td-warning-color);
      background: var(--td-warning-color-1);
    }
  }
}

.tool-card-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.tool-card-head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.tool-card-name {
  font-size: 13.5px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 0 1 auto;
  min-width: 0;
}

.tool-card-badge {
  flex: 0 0 auto;
  font-size: 10.5px;
  line-height: 1;
  padding: 3px 6px;
  color: var(--td-warning-color);
  background: transparent;
  border: 1px solid var(--td-warning-color-light);
  border-radius: var(--app-radius-xs);
  letter-spacing: 0.3px;
}

.tool-card-desc {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tool-card-hint {
  font-size: var(--app-text-xs);
  color: var(--td-warning-color);
  font-style: italic;
  line-height: 1.4;
}

.tool-card--disabled {

  .tool-card-name,
  .tool-card-desc {
    color: var(--td-text-color-placeholder);
  }
}

.effective-tools {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 12px;
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-md);
  border: 1px dashed var(--td-component-stroke);
  min-height: 52px;
  align-items: flex-start;
}

.effective-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  font-size: var(--app-text-sm);
  line-height: 18px;
  color: var(--td-brand-color);
  background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 22%, transparent);
  border-radius: var(--app-radius-pill);
  max-width: 100%;
}

.effective-chip-label {
  font-weight: 500;
}

.effective-chip-reason {
  font-size: var(--app-text-xs);
  color: var(--td-warning-color);
  font-style: normal;

  &::before {
    content: "· ";
    color: var(--td-text-color-placeholder);
    margin-right: 2px;
  }
}

.effective-chip--inactive {
  color: var(--td-text-color-placeholder);
  background: var(--td-bg-color-secondarycontainer);
  border-color: var(--td-component-stroke);

  .effective-chip-label {
    text-decoration: line-through;
  }
}

.effective-tools-empty {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  font-style: italic;
}

.skill-pick-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.skill-pick-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skill-pick-group__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 2px;
}

.skill-pick-group__bar {
  display: inline-block;
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--td-brand-color);
}

.skill-pick-group--pending .skill-pick-group__bar {
  background: var(--td-text-color-placeholder);
}

.skill-pick-group__title {
  font-size: var(--app-text-md);
  font-weight: 600;
  color: var(--td-text-color-primary);
  letter-spacing: 0.2px;
}

.skill-pick-group__count {
  min-width: 20px;
  padding: 0 6px;
  font-size: var(--app-text-xs);
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--app-radius-pill);
  text-align: center;
  line-height: 18px;
}

.skill-pick {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 8px 10px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
}

.skill-pick--pending {
  background: var(--td-bg-color-secondarycontainer);
}

.skill-pick__check {
  flex-shrink: 0;
}

.skill-pick__badge {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 7px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
}

.skill-pick--ready .skill-pick__badge {
  background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: var(--td-brand-color);
}

.skill-pick--pending .skill-pick__badge {
  background: var(--td-bg-color-container);
  color: var(--td-text-color-placeholder);
}

.skill-pick__body {
  flex: 1;
  min-width: 0;
}

.skill-pick__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.skill-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.skill-pick__hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);

  .t-icon {
    flex-shrink: 0;
  }
}

.skill-pick__hint--upgrade {
  color: var(--td-warning-color);
}

.skill-pick__hint--busy {
  color: var(--td-brand-color);

  .t-icon {
    animation: wk-spin 1s linear infinite;
  }
}

.skill-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  margin: 2px 0 0;
  overflow: hidden;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  line-height: 1.45;
  white-space: pre-line;
  word-break: break-word;
}

.hint-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  border: none;
  border-radius: var(--app-radius-xs);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: help;
  line-height: 1;

  &:hover,
  &:focus-visible {
    color: var(--td-brand-color);
    outline: none;
  }
}

.hint-popover {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 340px;
}

.hint-popover__title {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.hint-popover__text {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.55;
}

.empty-hint {
  color: var(--td-text-color-placeholder);
  font-style: italic;

  .go-settings-link {
    display: inline-block;
    font-style: normal;
  }
}

.textarea-with-template {
  position: relative;
  width: 100%;
}

.intent-prompts-editor {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.intent-toggle-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.intent-toggle-group :deep(.intent-toggle-btn--active) {
  background-color: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
  border-color: var(--td-brand-color);
  color: var(--td-brand-color);
  font-weight: 500;

  &:hover,
  &:focus-visible {
    background-color: color-mix(in srgb, var(--td-brand-color) 14%, transparent);
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);
  }
}

.intent-toggle-btn {
  max-width: 100%;
}

.intent-toggle-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.intent-toggle-dot {
  flex-shrink: 0;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.intent-active-desc {
  margin: 0;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  line-height: 1.5;
}

.system-prompt-textarea {
  width: 100%;
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-md);

  :deep(textarea) {
    resize: vertical !important;
    min-height: 200px;
  }
}

.placeholder-tags {
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: var(--app-text-sm);
  line-height: 1.4;
  overflow-x: auto;
  white-space: nowrap;
  padding-bottom: 4px;

  scrollbar-width: thin;

  &::-webkit-scrollbar {
    height: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(0, 0, 0, 0.1);
    border-radius: 2px;
  }

  .placeholder-label {
    color: var(--td-text-color-secondary);
    flex-shrink: 0;
  }

  .placeholder-hint {
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-xs);
    user-select: none;
    flex-shrink: 0;
  }

  .placeholder-tag {
    display: inline-flex;
    align-items: center;
    padding: 1px 5px;
    border-radius: 3px;
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-xs);
    color: var(--td-text-color-primary);
    background-color: var(--td-bg-color-secondarycontainer);
    cursor: pointer;
    transition: all var(--app-motion-base);
    user-select: none;
    border: 1px solid transparent;
    flex-shrink: 0;

    &:hover {
      color: var(--td-brand-color);
      background-color: var(--td-brand-color-light);
      border-color: var(--td-brand-color-focus);
    }

    &:active {
      background-color: var(--td-brand-color-focus);
    }
  }
}

.placeholder-popup-wrapper {
  position: fixed;
  z-index: 10001;
  pointer-events: auto;
}

.placeholder-popup {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  max-width: 320px;
  max-height: 240px;
  overflow-y: auto;
  padding: 4px;
}

.placeholder-item {
  padding: 6px 10px;
  cursor: pointer;
  transition: background-color var(--app-motion-fast);
  border-radius: var(--app-radius-xs);

  &:hover,
  &.active {
    background-color: var(--td-bg-color-container-hover);
  }

  .placeholder-name {
    margin-bottom: 2px;

    code {
      background: var(--td-bg-color-container-hover);
      padding: 2px 5px;
      border-radius: 3px;
      font-family: var(--app-font-family-mono);
      font-size: var(--app-text-xs);
      color: var(--td-brand-color);
    }
  }

  .placeholder-desc {
    font-size: var(--app-text-xs);
    color: var(--td-text-color-secondary);
  }
}

.builtin-agent-hint {
  display: inline-flex;
  align-items: center;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-2xl);
  line-height: 1;
  cursor: help;
  transition: color var(--app-motion-base);

  &:hover,
  &:focus-visible {
    color: var(--td-warning-color);
    outline: none;
  }

  &:focus-visible {
    border-radius: 2px;
    box-shadow: 0 0 0 2px var(--td-warning-color-focus);
  }
}

.builtin-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: var(--app-radius-xl);
  flex-shrink: 0;

  &.normal {
    background: linear-gradient(135deg, color-mix(in srgb, var(--td-brand-color) 15%, transparent) 0%, color-mix(in srgb, var(--td-brand-color) 8%, transparent) 100%);
    color: var(--td-brand-color-active);
  }

  &.agent {
    background: linear-gradient(135deg, rgba(124, 77, 255, 0.15) 0%, rgba(124, 77, 255, 0.08) 100%);
    color: var(--td-brand-color);
  }
}

.prompt-toggle {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;

  .prompt-toggle-label {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
  }
}

.prompt-disabled-hint {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-md);
  font-style: italic;
  padding: 12px 16px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--app-radius-sm);
}

.system-prompt-tabs {
  width: 100%;

  .prompt-variant-tabs {
    :deep(.t-tabs__nav) {
      margin-bottom: 12px;
    }
  }
}

.kb-option-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 0;
}

.kb-option-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: var(--app-radius-sm);
  font-size: var(--app-text-base);

  // Document KB
  &.doc-icon {
    background: rgba(16, 185, 129, 0.1);
    color: var(--td-success-color);
  }

  // FAQ KB
  &.faq-icon {
    background: rgba(0, 82, 217, 0.1);
    color: var(--td-brand-color);
  }
}

.kb-option-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--app-text-md);
  color: var(--td-text-color-primary);
}

.kb-option-org {
  flex-shrink: 0;
  font-size: var(--app-text-xs);
  color: var(--td-text-color-placeholder);
  background: var(--td-bg-color-secondarycontainer);
  padding: 1px 6px;
  border-radius: var(--app-radius-xs);
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kb-option-disabled-hint {
  flex-shrink: 0;
  font-size: var(--app-text-xs);
  color: var(--td-warning-color-6);
  background: var(--td-warning-color-1);
  padding: 1px 6px;
  border-radius: var(--app-radius-xs);
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-type-preset-desc {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

.agent-type-select {
  width: 100%;
  min-width: 240px;
  max-width: 360px;
}

.kb-option-count {
  flex-shrink: 0;
  font-size: var(--app-text-xs);
  color: var(--td-text-color-placeholder);
  background: var(--td-bg-color-secondarycontainer);
  padding: 1px 6px;
  border-radius: var(--app-radius-xs);
}

.kb-option-tag {
  flex-shrink: 0;
  font-size: var(--app-text-2xs);
  font-weight: 500;
  padding: 0 5px;
  border-radius: 3px;
  line-height: 18px;
}

.tag-rag {
  color: #165dff;
  background: rgba(22, 93, 255, 0.1);
}

.tag-wiki {
  color: #00b42a;
  background: rgba(0, 180, 42, 0.1);
}

</style>

<!-- Non-scoped styles: TDesign teleports the popup outside this component, so
     scoped selectors can't reach .agent-type-popup .t-select-option. -->
<style lang="less">
.reasoning-level-select-popup {
  padding: 4px;

  .t-select-option {
    height: auto !important;
    padding: 6px 10px;
    border-radius: 6px;
    margin: 2px 0;
    white-space: normal;
  }
}

.reasoning-level-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.35;
  min-width: 0;

  &__title {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }

  &__hint {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
    word-break: break-word;
  }
}

.agent-type-popup {
  .t-select-option {
    // The default option is one 32px line; we show two lines, so drop the fixed height and widen the padding
    height: auto !important;
    min-height: 48px;
    line-height: 1.4;
    padding: 8px 12px;
    white-space: normal;
  }
}

.sandbox-config-select-popup {
  .t-select-option {
    height: auto;
    padding: 6px 10px;
  }
}

.agent-type-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  width: 100%;
}

.agent-type-option-label {
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;
}

.agent-type-option-desc {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  line-height: 1.4;
  white-space: normal;
  overflow-wrap: break-word;
}
</style>
