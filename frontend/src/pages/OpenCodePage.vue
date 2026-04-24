<template>
  <!-- ==================== OPENCODE TOOL ==================== -->
  <div v-loading="ocLoading || mainConfigLoading" class="opencode-page">

    <!-- Page-level secondary nav -->
    <div class="oc-secondary-nav">
      <div class="oc-secondary-nav-item" :class="{ active: ocActiveTab === 'opencode' }" @click="ocActiveTab = 'opencode'">
        <el-icon size="15"><Setting /></el-icon>
        <span>OpenCode</span>
      </div>
      <div class="oc-secondary-nav-item" :class="{ active: ocActiveTab === 'slim' }" @click="ocActiveTab = 'slim'">
        <el-icon size="15"><Operation /></el-icon>
        <span>oh-my-opencode-slim</span>
      </div>
    </div>

     <!-- ====== Tab 1: OpenCode ====== -->
    <div v-show="ocActiveTab === 'opencode'">
      <template v-if="mainConfig">
        <!-- Tertiary nav -->
        <div class="oc-tertiary-nav">
          <div class="oc-tertiary-nav-item" :class="{ active: ocOpenCodeSubTab === 'overview' }" @click="ocOpenCodeSubTab = 'overview'">
            <span>概览</span>
          </div>
          <div class="oc-tertiary-nav-item" :class="{ active: ocOpenCodeSubTab === 'model' }" @click="ocOpenCodeSubTab = 'model'">
            <span>模型设置</span>
          </div>
          <div class="oc-tertiary-nav-item" :class="{ active: ocOpenCodeSubTab === 'provider' }" @click="ocOpenCodeSubTab = 'provider'">
            <span>Agent</span>
          </div>
        </div>

        <!-- Sub-tab: Overview -->
        <div v-show="ocOpenCodeSubTab === 'overview'">
    <div>
      <!-- Stats chips -->
      <div class="oc-ov-chips">
        <div class="oc-ov-chip">
          <span class="oc-ov-chip-val oc-ov-chip-val--primary">{{ ocAvailableModels.length }}</span>
          <span class="oc-ov-chip-label">模型</span>
        </div>
        <div class="oc-ov-chip">
          <span class="oc-ov-chip-val oc-ov-chip-val--warning">{{ ocMCPs.length }}</span>
          <span class="oc-ov-chip-label">MCP</span>
        </div>
        <div class="oc-ov-chip">
          <span class="oc-ov-chip-val oc-ov-chip-val--success">{{ ocSkills.length }}</span>
          <span class="oc-ov-chip-label">Skills</span>
        </div>
        <div class="oc-ov-chip">
          <span class="oc-ov-chip-val">{{ mainProviders.length }}</span>
          <span class="oc-ov-chip-label">Providers</span>
        </div>
        <div class="oc-ov-chip">
          <span class="oc-ov-chip-val">{{ ocConfig?.presets ? Object.keys(ocConfig.presets).length : 0 }}</span>
          <span class="oc-ov-chip-label">预设</span>
        </div>
      </div>
      <!-- Config path -->
      <div class="oc-ov-paths">
        <div class="oc-ov-path-row">
          <span class="oc-ov-path-dot oc-ov-path-dot--primary"></span>
          <span class="oc-ov-path-label">opencode.json</span>
          <span class="oc-ov-path-value">{{ mainConfigPath || '~/.config/opencode/opencode.json' }}</span>
          <el-button size="small" text @click="OpenOpenCodeConfigDir" title="打开配置目录" class="oc-ov-path-btn">
            <el-icon :size="14"><FolderOpened /></el-icon>
          </el-button>
        </div>
      </div>
      <!-- MCP + Skills + Providers -->
      <div class="oc-ov-grid">
        <!-- Providers -->
        <div class="oc-ov-panel">
          <div class="oc-ov-panel-header">
            <div class="oc-ov-panel-title-group">
              <span class="oc-ov-panel-indicator oc-ov-panel-indicator--primary"></span>
              <h2 class="oc-ov-panel-title">Providers</h2>
            </div>
          </div>
          <div class="oc-integrate-list">
            <div v-for="prov in mainProviders" :key="prov.name" class="oc-integrate-item oc-ov-item">
              <div class="oc-integrate-item-left">
                <div class="oc-integrate-badge oc-ov-badge oc-ov-badge--provider">
                  <el-icon :size="14"><Connection /></el-icon>
                </div>
                <div class="oc-integrate-item-info">
                  <div class="oc-integrate-item-name oc-ov-item-name">{{ prov.name }}</div>
                </div>
              </div>
            </div>
          </div>
          <div class="oc-integrate-empty oc-ov-empty" v-if="!mainProviders.length">未配置 Provider</div>
        </div>
        <!-- MCP Servers -->
        <div class="oc-ov-panel">
          <div class="oc-ov-panel-header">
            <div class="oc-ov-panel-title-group">
              <span class="oc-ov-panel-indicator oc-ov-panel-indicator--warning"></span>
              <h2 class="oc-ov-panel-title">MCP Servers</h2>
            </div>
            <el-button v-if="!ocMCPSkillsRefreshing" size="small" text @click="handleRefreshMCPSkills" title="刷新">
              <el-icon><RefreshRight /></el-icon>
            </el-button>
          </div>
          <div class="oc-integrate-list" v-if="ocMCPs.length">
            <div v-for="mcp in ocMCPs" :key="mcp.name" class="oc-integrate-item oc-ov-item">
              <div class="oc-integrate-item-left">
                <div class="oc-integrate-badge oc-integrate-badge-mcp oc-ov-badge">
                  <el-icon :size="14"><Monitor v-if="mcp.type === 'local'" /><Link v-else /></el-icon>
                </div>
                <div class="oc-integrate-item-info">
                  <div class="oc-integrate-item-name oc-ov-item-name">{{ mcp.name }}</div>
                  <div class="oc-integrate-item-detail oc-ov-item-detail">{{ mcp.type === 'local' ? mcp.command : mcp.url }}</div>
                </div>
              </div>
              <el-tag size="small" :type="mcp.source === 'plugin' ? 'warning' : 'success'" effect="plain" class="oc-ov-tag">
                {{ mcp.source === 'plugin' ? '插件' : '配置' }}
              </el-tag>
            </div>
          </div>
          <div class="oc-integrate-empty oc-ov-empty" v-else>未配置 MCP Server</div>
        </div>
        <!-- Skills -->
        <div class="oc-ov-panel">
          <div class="oc-ov-panel-header">
            <div class="oc-ov-panel-title-group">
              <span class="oc-ov-panel-indicator oc-ov-panel-indicator--success"></span>
              <h2 class="oc-ov-panel-title">Skills</h2>
            </div>
            <el-button v-if="!ocMCPSkillsRefreshing" size="small" text @click="handleRefreshMCPSkills" title="刷新">
              <el-icon><RefreshRight /></el-icon>
            </el-button>
          </div>
          <div class="oc-integrate-list" v-if="ocSkills.length">
            <div v-for="skill in ocSkills" :key="skill.name" class="oc-integrate-item oc-ov-item">
              <div class="oc-integrate-item-left">
                <div class="oc-integrate-badge oc-integrate-badge-skill oc-ov-badge">
                  <el-icon :size="14"><MagicStick /></el-icon>
                </div>
                <div class="oc-integrate-item-info">
                  <div class="oc-integrate-item-name oc-ov-item-name">{{ skill.name }}</div>
                  <div class="oc-integrate-item-detail oc-ov-item-detail" v-if="skill.description">{{ skill.description }}</div>
                </div>
              </div>
              <el-tag size="small" :type="skill.source === 'plugin' ? 'warning' : skill.source === 'agent' ? undefined : 'success'" effect="plain" class="oc-ov-tag">
                {{ skill.source === 'plugin' ? '插件' : skill.source === 'agent' ? 'Agent' : '配置' }}
              </el-tag>
            </div>
          </div>
          <div class="oc-integrate-empty oc-ov-empty" v-else>未安装 Skill</div>
        </div>
      </div>
    </div>
        </div>

        <!-- Sub-tab: Model settings -->
        <div v-show="ocOpenCodeSubTab === 'model'">
          <div class="oc-section">
            <div class="content-header"><h2 class="content-title">模型设置</h2></div>
            <el-form label-position="top">
              <el-form-item label="主模型 (model)">
                <el-select
                  v-model="mainConfig.model"
                  filterable
                  allow-create
                  placeholder="选择或输入模型"
                  size="large"
                  style="width: 100%"
                  :loading="ocModelsLoading"
                  @change="onMainModelChange"
                >
                  <el-option v-for="m in ocAvailableModels" :key="m" :label="m" :value="m" />
                </el-select>
              </el-form-item>
              <el-form-item label="轻量模型 (small_model)">
                <el-select
                  v-model="mainConfig.small_model"
                  filterable
                  allow-create
                  placeholder="选择或输入模型"
                  size="large"
                  style="width: 100%"
                  :loading="ocModelsLoading"
                  @change="onMainSmallModelChange"
                >
                  <el-option v-for="m in ocAvailableModels" :key="m" :label="m" :value="m" />
                </el-select>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <!-- Sub-tab: Provider & Agent -->
        <div v-show="ocOpenCodeSubTab === 'provider'">

          <!-- Disabled Agents -->
          <div class="oc-section">
            <div class="content-header"><h2 class="content-title">禁用的 Agent</h2></div>
            <div class="oc-prompt-list">
              <div v-for="ag in mainAgentDisables" :key="ag.name" class="oc-prompt-item">
                <div class="oc-prompt-item-left">
                  <span class="oc-prompt-item-name">{{ ag.name }}</span>
                </div>
                <el-switch v-model="ag.disable" @change="mainConfigDirty = true" />
              </div>
            </div>
            <div class="oc-integrate-empty" v-if="!mainAgentDisables.length">无禁用的 Agent</div>
          </div>

          <!-- Save button for main config -->
          <div style="margin-top: 16px; display: flex; justify-content: flex-end">
            <el-button type="primary" size="large" @click="saveMainConfig" :disabled="!mainConfigDirty" style="min-width: 120px">
              <el-icon><Check /></el-icon><span>保存配置</span>
            </el-button>
          </div>
        </div>
       </template>

      <div class="empty-state" v-else-if="!mainConfigLoading">
        <div class="empty-icon"><el-icon size="48"><Document /></el-icon></div>
        <h3 class="empty-title">未找到配置文件</h3>
        <p class="empty-desc">opencode.json 不存在于默认路径</p>
      </div>
    </div>

    <!-- ====== Tab 2: oh-my-opencode-slim ====== -->
    <div v-show="ocActiveTab === 'slim'">
    <template v-if="ocConfig">
      <!-- Tertiary nav -->
      <div class="oc-tertiary-nav">
        <div class="oc-tertiary-nav-item" :class="{ active: slimSubTab === 'overview' }" @click="slimSubTab = 'overview'">
          <span>概览</span>
        </div>
        <div class="oc-tertiary-nav-item" :class="{ active: slimSubTab === 'agent' }" @click="slimSubTab = 'agent'">
          <span>Agent 配置</span>
        </div>
        <div class="oc-tertiary-nav-item" :class="{ active: slimSubTab === 'prompts' }" @click="slimSubTab = 'prompts'">
          <span>附加提示词</span>
        </div>
      </div>

      <!-- Sub-tab: Overview -->
      <div v-show="slimSubTab === 'overview'">
        <!-- Stats chips -->
        <div class="oc-ov-chips">
          <div class="oc-ov-chip">
            <span class="oc-ov-chip-val oc-ov-chip-val--primary">{{ ocAvailableModels.length }}</span>
            <span class="oc-ov-chip-label">模型</span>
          </div>
          <div class="oc-ov-chip">
            <span class="oc-ov-chip-val oc-ov-chip-val--warning">{{ ocMCPs.length }}</span>
            <span class="oc-ov-chip-label">MCP</span>
          </div>
          <div class="oc-ov-chip">
            <span class="oc-ov-chip-val oc-ov-chip-val--success">{{ ocSkills.length }}</span>
            <span class="oc-ov-chip-label">Skills</span>
          </div>
          <div class="oc-ov-chip">
            <span class="oc-ov-chip-val">{{ ocConfig?.presets ? Object.keys(ocConfig.presets).length : 0 }}</span>
            <span class="oc-ov-chip-label">预设</span>
          </div>
        </div>
        <!-- Config path -->
        <div class="oc-ov-paths">
          <div class="oc-ov-path-row">
            <span class="oc-ov-path-dot oc-ov-path-dot--primary"></span>
            <span class="oc-ov-path-label">oh-my-opencode-slim.json</span>
            <span class="oc-ov-path-value">{{ ocConfigPath || '~/.config/opencode/oh-my-opencode-slim.json' }}</span>
            <el-button size="small" text @click="OpenOpenCodeConfigDir" title="打开配置目录" class="oc-ov-path-btn">
              <el-icon :size="14"><FolderOpened /></el-icon>
            </el-button>
          </div>
        </div>
        <!-- Agent 配置 -->
        <div class="oc-ov-panel">
          <div class="oc-ov-panel-header">
            <div class="oc-ov-panel-title-group">
              <span class="oc-ov-panel-indicator oc-ov-panel-indicator--primary"></span>
              <h2 class="oc-ov-panel-title">Agent 配置 — {{ ocConfig.preset }}</h2>
            </div>
          </div>
          <div class="oc-integrate-list" v-if="ocConfig.presets?.[ocConfig.preset]">
            <div v-for="agentName in ocAgentNames" :key="agentName" class="oc-integrate-item oc-ov-item">
              <div class="oc-integrate-item-left">
                <div class="oc-integrate-badge oc-ov-badge" :style="{ background: ocAgentColors[agentName] }">
                  {{ ocAgentLabels[agentName].charAt(0) }}
                </div>
                <div class="oc-integrate-item-info">
                  <div class="oc-integrate-item-name oc-ov-item-name">{{ ocAgentLabels[agentName] }}</div>
                  <div class="oc-integrate-item-detail oc-ov-item-detail" v-if="ocGetActivePreset()?.[agentName]?.model">
                    {{ ocGetActivePreset()?.[agentName]?.model }}
                  </div>
                  <div class="oc-integrate-item-detail oc-ov-item-detail oc-agent-empty" v-else>未配置模型</div>
                  <div class="oc-integrate-item-detail oc-ov-item-detail oc-ov-prompt-preview" v-if="ocAppendPrompts[agentName]">
                    <span class="oc-ov-prompt-label">附加提示词</span>
                    {{ ocAppendPrompts[agentName] }}
                  </div>
                </div>
              </div>
              <div class="oc-ov-item-tags">
                <el-tag v-if="ocGetActivePreset()?.[agentName]?.variant" size="small" effect="plain" class="oc-ov-tag">{{ ocGetActivePreset()?.[agentName]?.variant }}</el-tag>
                <el-tag v-if="ocGetActivePreset()?.[agentName]?.skills?.length" size="small" effect="plain" type="success" class="oc-ov-tag">{{ ocGetActivePreset()?.[agentName]?.skills?.length }} Skills</el-tag>
                <el-tag v-if="ocGetActivePreset()?.[agentName]?.mcps?.length" size="small" effect="plain" type="warning" class="oc-ov-tag">{{ ocGetActivePreset()?.[agentName]?.mcps?.length }} MCPs</el-tag>
                <el-tag v-if="ocAppendPrompts[agentName]" size="small" effect="plain" type="info" class="oc-ov-tag">提示词</el-tag>
              </div>
            </div>
          </div>
            <div class="oc-integrate-empty oc-ov-empty" v-else>无活跃预设</div>
            </div>
      </div>

      <!-- Sub-tab: Agent 配置 -->
      <div v-show="slimSubTab === 'agent'">
        <!-- Preset selector -->
        <div class="oc-section">
          <div class="content-header">
            <h2 class="content-title">活跃预设</h2>
            <el-button size="small" text @click="ocOpenNewPreset">
              <el-icon><Plus /></el-icon><span>新建预设</span>
            </el-button>
          </div>
          <div class="oc-preset-list">
            <div
              v-for="(preset, name) in ocConfig.presets"
              :key="name"
              class="oc-preset-item"
              :class="{ active: ocConfig.preset === name }"
              @click="ocSwitchPreset(String(name))"
            >
              <div class="oc-preset-radio">
                <div class="oc-preset-dot" :class="{ active: ocConfig.preset === name }"></div>
              </div>
              <span class="oc-preset-name">{{ name }}</span>
              <span class="oc-preset-count">{{ Object.keys(preset).length }} 个 Agent</span>
              <el-icon
                size="14"
                class="oc-preset-rename"
                @click.stop="ocOpenRenamePreset(String(name))"
                title="重命名"
              >
                <Edit />
              </el-icon>
              <el-icon
                v-if="ocConfig.preset !== name"
                size="14"
                class="oc-preset-delete"
                @click.stop="ocDeletePreset(String(name))"
              >
                <Delete />
              </el-icon>
            </div>
          </div>
        </div>

        <!-- Agent cards -->
        <div class="oc-section">
          <div class="content-header">
            <h2 class="content-title">Agent 配置 — {{ ocConfig.preset }}</h2>
          </div>
          <div class="oc-agent-grid">
            <div
              v-for="agentName in ocAgentNames"
              :key="agentName"
              class="oc-agent-card"
            >
              <div class="oc-agent-header">
                <div class="oc-agent-badge" :style="{ background: ocAgentColors[agentName] }">
                  {{ ocAgentLabels[agentName].charAt(0) }}
                </div>
                <div class="oc-agent-info">
                  <div class="oc-agent-name">{{ ocAgentLabels[agentName] }}</div>
                  <div class="oc-agent-model" v-if="ocGetActivePreset()?.[agentName]?.model">
                    {{ ocGetActivePreset()?.[agentName]?.model }}
                  </div>
                  <div class="oc-agent-model oc-agent-empty" v-else>未配置</div>
                </div>
                <el-button text size="small" @click="ocOpenAgentEdit(agentName)" class="oc-agent-edit-btn">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </div>
              <div class="oc-agent-meta" v-if="ocGetActivePreset()?.[agentName]">
                <el-tag
                  v-if="ocGetActivePreset()?.[agentName]?.variant"
                  size="small"
                  effect="plain"
                  class="meta-tag"
                >
                  {{ ocGetActivePreset()?.[agentName]?.variant }}
                </el-tag>
                <span class="oc-agent-field">
                  <span class="oc-agent-field-label">Skills</span>
                  <span class="oc-agent-field-value" :class="{ 'oc-agent-empty': !ocGetActivePreset()?.[agentName]?.skills?.length }">
                    {{ ocGetActivePreset()?.[agentName]?.skills?.length ? ocGetActivePreset()?.[agentName]?.skills?.join(', ') : '未设置' }}
                  </span>
                </span>
                <span class="oc-agent-field">
                  <span class="oc-agent-field-label">MCPs</span>
                  <span class="oc-agent-field-value" :class="{ 'oc-agent-empty': !ocGetActivePreset()?.[agentName]?.mcps?.length }">
                    {{ ocGetActivePreset()?.[agentName]?.mcps?.length ? ocGetActivePreset()?.[agentName]?.mcps?.join(', ') : '未设置' }}
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Sub-tab: 附加提示词 -->
      <div v-show="slimSubTab === 'prompts'">
        <div class="oc-section">
          <div class="content-header">
            <h2 class="content-title">附加提示词</h2>
            <div style="display: flex; gap: 8px; align-items: center">
              <el-tag type="info" size="small" effect="plain" style="font-size: 11px">
                _append.md
              </el-tag>
              <el-button size="small" text type="warning" @click="ocSyncPromptsToFiles" title="同步持久存储到 .md 文件">
                <el-icon><RefreshRight /></el-icon><span>同步</span>
              </el-button>
              <el-button size="small" text @click="ocCheckDiff" title="手动检查存储与文件是否一致">
                <el-icon><CircleCheck /></el-icon><span>检查</span>
              </el-button>
              <el-button size="small" text @click="async () => { const dir = await GetAppendPromptStoreDir(); if (dir) OpenInExplorer(dir) }" title="打开备份目录">
                <el-icon><FolderOpened /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="oc-prompt-list" v-loading="ocAppendPromptsLoading">
            <div
              v-for="agentName in ocAgentNames"
              :key="'prompt-' + agentName"
              class="oc-prompt-item"
              @click="ocOpenPromptEdit(agentName)"
            >
              <div class="oc-prompt-item-left">
                <div class="oc-agent-badge" :style="{ background: ocAgentColors[agentName], width: '28px', height: '28px', fontSize: '12px', borderRadius: '7px' }">
                  {{ ocAgentLabels[agentName].charAt(0) }}
                </div>
                <div class="oc-prompt-item-info">
                  <div class="oc-prompt-item-name">{{ ocAgentLabels[agentName] }}</div>
                  <div class="oc-prompt-item-preview" v-if="ocAppendPrompts[agentName]">
                    {{ ocAppendPrompts[agentName].substring(0, 60) }}{{ ocAppendPrompts[agentName].length > 60 ? '...' : '' }}
                  </div>
                  <div class="oc-prompt-item-preview oc-prompt-item-empty" v-else>
                    未设置
                  </div>
                </div>
              </div>
              <el-icon size="14" class="oc-prompt-item-arrow"><ArrowRight /></el-icon>
            </div>
          </div>
          <div class="form-hint" style="margin-top: 8px">
            附加提示词会追加到 Agent 默认系统提示词末尾，不会覆盖原有内容。保存时同时写入 .md 文件和持久备份。
          </div>
        </div>
      </div>
  </template>

  <div class="empty-state" v-else-if="!ocLoading">
    <div class="empty-icon"><el-icon size="48"><Document /></el-icon></div>
    <h3 class="empty-title">未找到配置文件</h3>
    <p class="empty-desc">oh-my-opencode-slim.json 不存在于默认路径</p>
  </div>
    </div>
  </div>

  <!-- OpenCode Agent Edit Dialog -->
  <el-dialog
    v-model="ocAgentDialogVisible"
    :title="`编辑 ${ocAgentLabels[ocEditAgentName]}`"
    width="520px"
    align-center
  >
    <el-form label-position="top">
      <el-form-item label="模型 (Model)">
        <el-select
          v-model="ocEditForm.model"
          filterable
          placeholder="选择模型"
          size="large"
          style="width: 100%"
          :loading="ocModelsLoading"
        >
          <el-option
            v-for="m in ocAvailableModels"
            :key="m"
            :label="m"
            :value="m"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="Variant">
        <el-select v-model="ocEditForm.variant" placeholder="选择 Variant" size="large" clearable style="width: 100%">
          <el-option label="high" value="high" />
          <el-option label="medium" value="medium" />
          <el-option label="low" value="low" />
        </el-select>
      </el-form-item>
      <el-form-item label="Skills">
        <el-select
          v-model="ocEditForm.skills"
          multiple
          placeholder="选择 Skills"
          size="large"
          style="width: 100%"
        >
          <el-option label="* (全部)" value="*" />
          <el-option
            v-for="s in ocSkills"
            :key="s.name"
            :label="s.name"
            :value="s.name"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="MCPs">
        <el-select
          v-model="ocEditForm.mcps"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="选择或输入，* 全部，!name 排除"
          size="large"
          style="width: 100%"
        >
          <el-option label="* (全部)" value="*" />
          <el-option
            v-for="m in ocMCPs"
            :key="m.name"
            :label="m.name"
            :value="m.name"
          />
        </el-select>
        <div class="form-hint">* 表示全部，!name 表示排除</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="ocAgentDialogVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="ocSaveAgentEdit" style="flex: 1">保存</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- OpenCode New Preset Dialog -->
  <el-dialog v-model="ocNewPresetDialogVisible" title="新建预设" width="400px" align-center>
    <el-form label-position="top">
      <el-form-item label="预设名称" required>
        <el-input v-model="ocNewPresetName" placeholder="输入预设名称" size="large" @keyup.enter="ocCreatePreset" />
      </el-form-item>
    </el-form>
    <div class="dialog-desc">
      新预设将创建空的 Agent 配置，你可以逐个添加。
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="ocNewPresetDialogVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="ocCreatePreset" style="flex: 1">创建</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- OpenCode Rename Preset Dialog -->
  <el-dialog v-model="ocRenamePresetDialogVisible" title="重命名预设" width="400px" align-center>
    <el-form label-position="top">
      <el-form-item label="预设名称" required>
        <el-input v-model="ocRenameNewName" placeholder="输入预设名称" size="large" @keyup.enter="ocRenamePreset" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="ocRenamePresetDialogVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="ocRenamePreset" style="flex: 1">确认</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- OpenCode Append Prompt Dialog -->
  <el-dialog
    v-model="ocPromptDialogVisible"
    :title="`附加提示词 — ${ocAgentLabels[ocPromptAgentName]}`"
    width="600px"
    align-center
    :close-on-click-modal="false"
  >
    <div class="oc-prompt-path" v-if="ocPromptFilePath">
      <el-icon size="13"><Document /></el-icon>
      <span>{{ ocPromptFilePath }}</span>
    </div>
    <el-input
      v-model="ocPromptContent"
      type="textarea"
      :autosize="{ minRows: 8, maxRows: 20 }"
      placeholder="输入附加提示词，将追加到 Agent 默认系统提示词末尾..."
      @input="ocPromptDirty = true"
      style="margin-top: 12px"
    />
    <div class="form-hint" style="margin-top: 8px">
      支持 Markdown 格式。此内容会追加到 {{ ocAgentLabels[ocPromptAgentName] }} 的默认提示词末尾，不会覆盖原有内容。
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="ocPromptDialogVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="ocSavePrompt" style="flex: 1">保存</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- OpenCode Append Prompt Diff Alert -->
  <el-dialog
    v-model="ocPromptDiffVisible"
    title="检测到附加提示词差异"
    width="520px"
    align-center
    :close-on-click-modal="false"
  >
    <div class="dialog-desc" style="margin-bottom: 16px">
      以下 Agent 的 .md 文件与持久存储内容不一致，可能是外部手动修改了文件。
    </div>
    <div class="oc-diff-list">
      <div v-for="d in ocPromptDiffs" :key="d.agent" class="oc-diff-item">
        <div class="oc-diff-agent">
          <div class="oc-agent-badge" :style="{ background: ocAgentColors[d.agent], width: '24px', height: '24px', fontSize: '11px', borderRadius: '6px' }">
            {{ ocAgentLabels[d.agent]?.charAt(0) }}
          </div>
          <span>{{ ocAgentLabels[d.agent] }}</span>
        </div>
        <div class="oc-diff-detail">
          <div class="oc-diff-side">
            <span class="oc-diff-label">持久存储</span>
            <span class="oc-diff-value" :class="{ empty: !d.store }">{{ d.store ? d.store.substring(0, 50) + (d.store.length > 50 ? '...' : '') : '(空)' }}</span>
          </div>
          <span class="oc-diff-arrow">→</span>
          <div class="oc-diff-side">
            <span class="oc-diff-label">.md 文件</span>
            <span class="oc-diff-value" :class="{ empty: !d.file }">{{ d.file ? d.file.substring(0, 50) + (d.file.length > 50 ? '...' : '') : '(空)' }}</span>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="ocDismissDiff" style="flex: 1">忽略</el-button>
        <el-button size="large" type="warning" @click="ocImportPromptsFromFiles" style="flex: 1">以文件为准</el-button>
        <el-button type="primary" size="large" @click="ocSyncPromptsToFiles" style="flex: 1">以存储为准</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- OpenCode Preset Diff Alert -->
  <el-dialog
    v-model="ocPresetDiffVisible"
    title="检测到预设差异"
    width="520px"
    align-center
    :close-on-click-modal="false"
  >
    <div class="dialog-desc" style="margin-bottom: 16px">
      配置文件中的预设与持久存储不一致，可能是外部手动修改了配置文件。
    </div>
    <div class="oc-diff-detail" style="margin-bottom: 12px">
      <div class="oc-diff-side">
        <span class="oc-diff-label">持久存储活跃预设</span>
        <span class="oc-diff-value">{{ ocPresetDiff?.store_active || '(空)' }}</span>
      </div>
      <span class="oc-diff-arrow">→</span>
      <div class="oc-diff-side">
        <span class="oc-diff-label">配置文件活跃预设</span>
        <span class="oc-diff-value">{{ ocPresetDiff?.file_active || '(空)' }}</span>
      </div>
    </div>
    <div v-for="(d, i) in ocPresetDiff?.differences" :key="i" style="padding: 6px 0; font-size: 13px; color: var(--el-text-color-regular)">
      {{ d }}
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="ocDismissPresetDiff" style="flex: 1">忽略</el-button>
        <el-button size="large" type="warning" @click="ocImportPresetsFromConfig" style="flex: 1">以文件为准</el-button>
        <el-button type="primary" size="large" @click="ocSyncPresetsToConfig" style="flex: 1">以存储为准</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { GetAppendPromptStoreDir } from '../../wailsjs/go/main/App'
import {
  // State
  ocConfig,
  ocConfigPath,
  ocLoading,
  ocDirty,
  ocAvailableModels,
  ocModelsLoading,
  ocAgentDialogVisible,
  ocEditAgentName,
  ocEditPresetName,
  ocEditForm,
  ocNewPresetDialogVisible,
  ocNewPresetName,
  ocRenamePresetDialogVisible,
  ocRenameOldName,
  ocRenameNewName,
  ocPresetDiff,
  ocPresetDiffVisible,
  ocAppendPrompts,
  ocAppendPromptsLoading,
  ocPromptDialogVisible,
  ocPromptAgentName,
  ocPromptContent,
  ocPromptFilePath,
  ocPromptDirty,
  ocPromptDiffs,
  ocPromptDiffVisible,
  ocActiveTab,
  ocOpenCodeSubTab,
  slimSubTab,
  mainConfig,
  mainConfigPath,
  mainConfigLoading,
  mainConfigDirty,
  mainProviders,
  mainAgentDisables,
  ocMCPs,
  ocSkills,
  ocMCPSkillsRefreshing,
  ocAgentNames,
  ocAgentLabels,
  ocAgentColors,

  // Functions
  loadOpenCodeConfig,
  handleRefreshMCPSkills,
  saveOpenCodeConfig,
  ocSwitchPreset,
  ocOpenAgentEdit,
  ocSaveAgentEdit,
  ocOpenNewPreset,
  ocCreatePreset,
  ocDeletePreset,
  ocOpenRenamePreset,
  ocRenamePreset,
  ocGetActivePreset,
  ocSyncPresetsToConfig,
  ocImportPresetsFromConfig,
  ocDismissPresetDiff,
  ocOpenPromptEdit,
  ocSavePrompt,
  ocSyncPromptsToFiles,
  ocImportPromptsFromFiles,
  ocDismissDiff,
  ocCheckDiff,
  saveMainConfig,
  onMainModelChange,
  onMainSmallModelChange,
  OpenOpenCodeConfigDir,
  OpenInExplorer,
} from '../composables/useOpenCode'

// Initialize
loadOpenCodeConfig()
</script>

<style scoped>
/* Page container */
.opencode-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Tab content panels: fill remaining space and scroll with consistent padding */
.opencode-page > div:not(.oc-secondary-nav) {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
}

/* ===== OpenCode Config ===== */
.oc-section {
  margin-bottom: 16px;
}

/* Page-level secondary nav */
.oc-secondary-nav {
  display: flex;
  gap: 4px;
  padding: 0 0 8px 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 8px;
}

.oc-secondary-nav-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px 8px 0 0;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  transition: all 0.15s;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}

.oc-secondary-nav-item:hover {
  color: var(--text);
}

.oc-secondary-nav-item.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
}

.oc-tertiary-nav {
  display: flex;
  gap: 4px;
  padding: 0 0 8px 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 8px;
}

.oc-tertiary-nav-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px 8px 0 0;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
  transition: all 0.15s;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}

.oc-tertiary-nav-item:hover {
  color: var(--text);
}

.oc-tertiary-nav-item.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
}

/* Preset list */
.oc-preset-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.oc-preset-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
  border: 1px solid transparent;
}

.oc-preset-item:hover {
  background: var(--bg);
}

.oc-preset-item.active {
  background: var(--primary-bg);
  border-color: rgba(99, 102, 241, 0.2);
}

.oc-preset-radio {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.oc-preset-dot {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid var(--border);
  transition: all 0.15s;
}

.oc-preset-dot.active {
  border-color: var(--primary);
  background: var(--primary);
  box-shadow: inset 0 0 0 3px #fff;
}

.oc-preset-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  flex-shrink: 0;
}

.oc-preset-count {
  font-size: 12px;
  color: var(--text-muted);
  flex: 1;
}

.oc-preset-delete {
  display: none;
  color: var(--text-muted);
  flex-shrink: 0;
}

.oc-preset-item:hover .oc-preset-delete {
  display: block;
}

.oc-preset-delete:hover {
  color: var(--danger) !important;
}

.oc-preset-rename {
  display: none;
  color: var(--text-muted);
  flex-shrink: 0;
}

.oc-preset-item:hover .oc-preset-rename {
  display: block;
}

.oc-preset-rename:hover {
  color: var(--primary) !important;
}

/* Agent grid */
.oc-agent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 10px;
}

.oc-agent-card {
  background: var(--bg-card);
  border-radius: var(--radius);
  border: 1px solid var(--border);
  padding: 14px;
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.oc-agent-card:hover {
  box-shadow: var(--shadow-md);
  border-color: rgba(99, 102, 241, 0.2);
  transform: translateY(-1px);
}

.oc-agent-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.oc-agent-badge {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 15px;
  flex-shrink: 0;
}

.oc-agent-info {
  flex: 1;
  min-width: 0;
}

.oc-agent-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

.oc-agent-model {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  margin-top: 2px;
}

.oc-agent-empty {
  color: var(--text-muted);
  font-style: italic;
}

.oc-agent-edit-btn {
  flex-shrink: 0;
}

.oc-agent-meta {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-left: 50px;
  align-items: center;
}

.oc-agent-field {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  line-height: 1;
}

.oc-agent-field-label {
  color: var(--text-muted);
  flex-shrink: 0;
}

.oc-agent-field-value {
  color: var(--el-text-color-regular);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* MCP & Skills Integration List */
.oc-integrate-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.oc-integrate-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  transition: background 0.15s;
}

.oc-integrate-item:hover {
  background: var(--el-fill-color-light);
}

.oc-integrate-item-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.oc-integrate-badge {
  width: 22px;
  height: 22px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.oc-integrate-badge-mcp {
  background: #f59e0b;
}

.oc-integrate-badge-skill {
  background: #10b981;
}

.oc-integrate-item-info {
  min-width: 0;
  flex: 1;
  overflow: hidden;
}

.oc-integrate-item-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.oc-integrate-item-detail {
  font-size: 10px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
  margin-top: 2px;
}

.oc-integrate-empty {
  font-size: 13px;
  color: var(--text-muted);
  text-align: center;
  padding: 12px;
}

/* Overview chips */
.oc-ov-chips { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.oc-ov-chip { display: flex; align-items: center; gap: 6px; padding: 6px 12px; background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-sm); transition: all 0.15s; }
.oc-ov-chip:hover { border-color: rgba(99, 102, 241, 0.3); box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04); }
.oc-ov-chip-val { font-size: 18px; font-weight: 700; color: var(--text); line-height: 1; }
.oc-ov-chip-val--primary { color: var(--primary); }
.oc-ov-chip-val--warning { color: var(--warning); }
.oc-ov-chip-val--success { color: var(--success); }
.oc-ov-chip-label { font-size: 11px; color: var(--text-muted); line-height: 1; }

/* Config paths */
.oc-ov-paths { display: flex; flex-direction: column; gap: 4px; padding: 10px 14px; background: var(--bg); border-radius: var(--radius-sm); border: 1px solid var(--border); margin-bottom: 16px; }
.oc-ov-path-row { display: flex; align-items: center; gap: 8px; min-width: 0; }
.oc-ov-path-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; background: var(--text-muted); }
.oc-ov-path-dot--primary { background: var(--primary); }
.oc-ov-path-label { font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
.oc-ov-path-value { font-size: 11px; font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace; color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 0; }
.oc-ov-path-btn { flex-shrink: 0; color: var(--text-muted); margin-left: 4px; }

/* Overview grid */
.oc-ov-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 10px; }

/* Overview panel */
.oc-ov-panel { background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); padding: 14px; display: flex; flex-direction: column; gap: 8px; }
.oc-ov-panel-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.oc-ov-panel-title-group { display: flex; align-items: center; gap: 8px; }
.oc-ov-panel-indicator { width: 3px; height: 16px; border-radius: 2px; }
.oc-ov-panel-indicator--primary { background: var(--primary); }
.oc-ov-panel-indicator--warning { background: var(--warning); }
.oc-ov-panel-indicator--success { background: var(--success); }
.oc-ov-panel-title { font-size: 14px; font-weight: 600; color: var(--text); margin: 0; }

/* Overview item overrides */
.oc-ov-tag { font-size: 10px; }
.oc-ov-item-tags { display: flex; gap: 4px; flex-wrap: wrap; flex-shrink: 0; }
.oc-ov-prompt-preview { margin-top: 3px; padding: 2px 6px 2px 8px; background: var(--primary-bg); border-left: 2px solid var(--primary); border-radius: 0 4px 4px 0; font-style: normal; color: var(--text-muted); }
.oc-ov-prompt-label { font-style: normal; font-size: 10px; font-weight: 700; color: var(--primary); opacity: 0.8; margin-right: 4px; }
.oc-ov-badge--provider { background: var(--primary); }

/* Form hint */
.form-hint {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 6px;
}

/* Append Prompt List */
.oc-prompt-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.oc-prompt-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
  border: 1px solid transparent;
}

.oc-prompt-item:hover {
  background: var(--bg);
  border-color: var(--border);
}

.oc-prompt-item-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.oc-prompt-item-info {
  flex: 1;
  min-width: 0;
}

.oc-prompt-item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}

.oc-prompt-item-preview {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.oc-prompt-item-empty {
  color: var(--text-muted);
  font-style: italic;
}

.oc-prompt-item-arrow {
  color: var(--text-muted);
  flex-shrink: 0;
}

.oc-prompt-path {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

/* Prompt Diff Alert */
.oc-diff-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 360px;
  overflow-y: auto;
}

.oc-diff-item {
  background: var(--bg);
  border-radius: var(--radius-sm);
  padding: 12px;
  border: 1px solid var(--border);
}

.oc-diff-agent {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

.oc-diff-detail {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.oc-diff-side {
  flex: 1;
  min-width: 0;
}

.oc-diff-label {
  font-size: 11px;
  color: var(--text-muted);
  display: block;
  margin-bottom: 2px;
}

.oc-diff-value {
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-all;
}

.oc-diff-value.empty {
  color: var(--text-muted);
  font-style: italic;
}

.oc-diff-arrow {
  color: var(--text-muted);
  font-size: 12px;
  flex-shrink: 0;
  margin-top: 16px;
}

/* Dialog desc */
.dialog-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--primary-bg);
  border-radius: 8px;
  border-left: 3px solid var(--primary);
}

/* Content header */
.content-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.content-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
}

/* Empty state */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
}

.empty-icon {
  width: 88px;
  height: 88px;
  border-radius: 22px;
  background: var(--primary-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  margin-bottom: 20px;
}

.empty-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 6px;
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-bottom: 20px;
}

/* Meta tag */
.meta-tag {
  border: none !important;
  font-size: 11px !important;
  display: flex;
  align-items: center;
  gap: 3px;
}
</style>
