<template>
  <div class="general-settings">
    <div class="section-header">
      <h2>{{ 'General Settings' }}</h2>
      <p class="section-description">{{ 'Configure appearance and other basic options' }}</p>
    </div>

    <div class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Theme' }}</label>
          <p class="desc">{{ 'Choose the display theme for the interface, supports automatic switching with system settings' }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localTheme"
            style="width: 280px;"
            :placeholder="'Select theme'"
            @change="handleThemeChange"
          >
            <t-option value="light" :label="'Light'">{{ 'Light' }}</t-option>
            <t-option value="dark" :label="'Dark'">{{ 'Dark' }}</t-option>
            <t-option value="system" :label="'Follow System'">{{ 'Follow System' }}</t-option>
          </t-select>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Interface Font' }}</label>
          <p class="desc">{{ 'Font used for menus, body text, buttons, and most UI text' }}</p>
        </div>
        <div class="setting-control setting-control--stacked">
          <t-select
            v-model="localSansFont"
            style="width: 280px;"
            :placeholder="'Select font'"
            @change="handleSansFontChange"
          >
            <t-option
              v-for="opt in sansFontOptions"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            >
              <span :style="{ fontFamily: opt.preview }">{{ opt.label }}</span>
            </t-option>
          </t-select>
          <div class="font-preview" :style="{ fontFamily: currentSansStack }">
            {{ 'The quick brown fox jumps — Aa Gg Oo 0123' }}
          </div>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Code Font' }}</label>
          <p class="desc">{{ 'Used for code blocks, terminal commands, API keys, file paths and other technical text. Every character has the same width so you can tell 0 from O and 1 from l.' }}</p>
        </div>
        <div class="setting-control setting-control--stacked">
          <t-select
            v-model="localMonoFont"
            style="width: 280px;"
            :placeholder="'Select font'"
            @change="handleMonoFontChange"
          >
            <t-option
              v-for="opt in monoFontOptions"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            >
              <span :style="{ fontFamily: opt.preview }">{{ opt.label }}</span>
            </t-option>
          </t-select>
          <div class="font-preview font-preview--mono" :style="{ fontFamily: currentMonoStack }">
            {{ 'const msg = \'Hello\'; // 0O1l' }}
          </div>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Font Size' }}</label>
          <p class="desc">{{ 'Scales the entire interface (text, icons, spacing) and applies immediately' }}</p>
        </div>
        <div class="setting-control">
          <t-radio-group
            v-model="localFontSize"
            @change="handleFontSizeChange"
          >
            <t-radio-button value="small">{{ 'Small' }}</t-radio-button>
            <t-radio-button value="normal">{{ 'Normal' }}</t-radio-button>
            <t-radio-button value="large">{{ 'Large' }}</t-radio-button>
          </t-radio-group>
        </div>
      </div>

      <div class="setting-row" v-if="authStore.isLiteMode">
        <div class="setting-info">
          <label>{{ 'Auto Download Updates' }}</label>
          <p class="desc">{{ 'When enabled, automatically check and download the latest version in the background.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="isAutoCheckUpdateEnabled"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useTheme, type ThemeMode } from '@/composables/useTheme'
import {
  useFont,
  SANS_STACKS,
  MONO_STACKS,
  visibleSansKeys,
  visibleMonoKeys,
  type FontKey,
  type MonoFontKey,
  type FontSizeKey,
} from '@/composables/useFont'

const FONT_SANS_LABELS: Record<string, string> = {
  system: 'System Default',
  pingfang: 'PingFang SC',
  georgia: 'Georgia (Serif)',
  yahei: 'Microsoft YaHei',
  times: 'Times New Roman (Serif)',
  'noto-cjk': 'Noto Sans CJK',
  'dejavu-serif': 'DejaVu Serif (Serif)',
  'sans-serif': 'Generic Sans-Serif',
}

const FONT_MONO_LABELS: Record<string, string> = {
  system: 'System Default',
  menlo: 'Menlo',
  monaco: 'Monaco',
  consolas: 'Consolas',
  cascadia: 'Cascadia Code',
  'dejavu-mono': 'DejaVu Sans Mono',
  'liberation-mono': 'Liberation Mono',
  monospace: 'Generic Monospace',
}

const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const { currentTheme, setTheme } = useTheme()
const {
  currentSans,
  currentMono,
  currentSize,
  setSansFont,
  setMonoFont,
  setFontSize,
} = useFont()

const localTheme = ref<ThemeMode>(currentTheme.value)
const localSansFont = ref<FontKey>(currentSans.value)
const localMonoFont = ref<MonoFontKey>(currentMono.value)
const localFontSize = ref<FontSizeKey>(currentSize.value)

// Keep the form in sync if preferences change externally (e.g. on user switch).
watch(currentTheme, (val) => { localTheme.value = val })
watch(currentSans, (val) => { localSansFont.value = val })
watch(currentMono, (val) => { localMonoFont.value = val })
watch(currentSize, (val) => { localFontSize.value = val })

const sansFontOptions = computed<{ value: FontKey; label: string; preview: string }[]>(() =>
  visibleSansKeys().map((key) => ({
    value: key,
    label: (FONT_SANS_LABELS[key] ?? ''),
    preview: SANS_STACKS[key],
  })),
)

const monoFontOptions = computed<{ value: MonoFontKey; label: string; preview: string }[]>(() =>
  visibleMonoKeys().map((key) => ({
    value: key,
    label: (FONT_MONO_LABELS[key] ?? ''),
    preview: MONO_STACKS[key],
  })),
)

// Live preview stacks, driven by the local form refs so the preview row
// updates immediately on selection — even before handleSansFontChange
// commits the choice to the global store and writes the CSS variable.
const currentSansStack = computed(() => SANS_STACKS[localSansFont.value] ?? SANS_STACKS.system)
const currentMonoStack = computed(() => MONO_STACKS[localMonoFont.value] ?? MONO_STACKS.system)

const isAutoCheckUpdateEnabled = computed({
  get: () => settingsStore.isAutoCheckUpdateEnabled,
  set: (val) => {
    settingsStore.toggleAutoCheckUpdate(val)
    if (val) {
      // @ts-ignore
      if (window.go && window.go.main && window.go.main.App && window.go.main.App.AutoCheckForUpdates) {
        // @ts-ignore
        window.go.main.App.AutoCheckForUpdates()
      }
    }
  }
})

const handleThemeChange = (val: ThemeMode) => {
  if (!setTheme(val)) {
    // Setter rejected the value (validation guard); roll the form back to
    // the canonical state so the UI doesn't drift.
    localTheme.value = currentTheme.value
    return
  }
}

const handleSansFontChange = (val: FontKey) => {
  if (!setSansFont(val)) {
    localSansFont.value = currentSans.value
    return
  }
}

const handleMonoFontChange = (val: MonoFontKey) => {
  if (!setMonoFont(val)) {
    localMonoFont.value = currentMono.value
    return
  }
}

const handleFontSizeChange = (val: FontSizeKey) => {
  if (!setFontSize(val)) {
    localFontSize.value = currentSize.value
    return
  }
}
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.general-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  .setting-row();
}

.setting-info {
  .setting-info();
}

.setting-control {
  .setting-control();
}

// When a font picker is rendered, stack the select on top of a live
// preview line so the user can verify their choice without hunting for
// an API Info page or a code block.
.setting-control--stacked {
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.font-preview {
  width: 280px;
  padding: 8px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  line-height: 1.4;
  text-align: left;
  box-sizing: border-box;

  &--mono {
    // Harden the preview against wrap-around for long monospace samples.
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
</style>
