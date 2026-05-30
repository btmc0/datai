/**
 * Pi Config management page — edit system prompt, skills, project instructions,
 * apply templates, and sync configs to remote servers.
 */

import { useCallback, useEffect, useState } from 'preact/hooks'
import {
  listPiConfigs, savePiConfig, syncPiConfigs,
  listTemplates, applyPiTemplate, getServer,
  type PiConfig, type PiTemplate,
  type Server,
} from './datai-api'

// ── Types ──

type SyncState = 'idle' | 'syncing' | 'synced' | 'error'

interface SkillState {
  name: string
  enabled: boolean
  id?: string
  synced: boolean
}

// ── Component ──

export function PiConfigPage({ serverId }: { serverId: string }) {
  const [server, setServer] = useState<Server | null>(null)
  const [configs, setConfigs] = useState<PiConfig[]>([])
  const [templates, setTemplates] = useState<PiTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Editable state
  const [systemPrompt, setSystemPrompt] = useState('')
  const [systemPromptDirty, setSystemPromptDirty] = useState(false)
  const [projectSystem, setProjectSystem] = useState('')
  const [projectSystemDirty, setProjectSystemDirty] = useState(false)
  const [skills, setSkills] = useState<SkillState[]>([])
  const [skillsDirty, setSkillsDirty] = useState(false)

  // Sync states
  const [promptSync, setPromptSync] = useState<SyncState>('idle')
  const [projectSync, setProjectSync] = useState<SyncState>('idle')
  const [skillsSync, setSkillsSync] = useState<SyncState>('idle')
  const [globalSync, setGlobalSync] = useState<SyncState>('idle')

  // Template
  const [selectedTemplate, setSelectedTemplate] = useState('')
  const [applyingTemplate, setApplyingTemplate] = useState(false)

  // ── Load data ──

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [srv, cfgs, tmpls] = await Promise.all([
        getServer(serverId),
        listPiConfigs(serverId),
        listTemplates(),
      ])
      setServer(srv)
      setConfigs(cfgs)
      setTemplates(tmpls)
      applyConfigsToState(cfgs)
    } catch (err: any) {
      setError(err.message ?? 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [serverId])

  useEffect(() => { void loadData() }, [loadData])

  function applyConfigsToState(cfgs: PiConfig[]) {
    const sp = cfgs.find(c => c.config_type === 'system_prompt')
    setSystemPrompt(sp?.content ?? '')
    setSystemPromptDirty(false)

    const ps = cfgs.find(c => c.config_type === 'project_system')
    setProjectSystem(ps?.content ?? '')
    setProjectSystemDirty(false)

    const skillConfigs = cfgs.filter(c => c.config_type === 'skill')
    const defaultSkills = ['srcwalk', 'code-review', 'git-tools', 'testing', 'docker', 'debugging']
    const savedSkillNames = new Set(skillConfigs.map(s => s.name))
    const allSkillNames = [...new Set([...savedSkillNames, ...defaultSkills])]

    setSkills(allSkillNames.map(name => {
      const saved = skillConfigs.find(s => s.name === name)
      return {
        name,
        enabled: saved ? saved.content === 'enabled' : false,
        id: saved?.id,
        synced: saved?.synced_at != null,
      }
    }))
    setSkillsDirty(false)
  }

  // ── Sync status helpers ──

  function configSyncStatus(configType: string): 'synced' | 'pending' | 'new' {
    const cfg = configs.find(c => c.config_type === configType)
    if (!cfg) return 'new'
    return cfg.synced_at != null ? 'synced' : 'pending'
  }

  // ── Save handlers ──

  const saveSystemPrompt = useCallback(async () => {
    setPromptSync('syncing')
    try {
      await savePiConfig(serverId, {
        config_type: 'system_prompt',
        name: 'system-prompt',
        content: systemPrompt,
        remote_path: '~/.config/pi/system-prompt.md',
      })
      setSystemPromptDirty(false)
      setPromptSync('synced')
      const cfgs = await listPiConfigs(serverId)
      setConfigs(cfgs)
    } catch {
      setPromptSync('error')
    }
  }, [serverId, systemPrompt])

  const saveProjectSystem = useCallback(async () => {
    setProjectSync('syncing')
    try {
      await savePiConfig(serverId, {
        config_type: 'project_system',
        name: 'project-system',
        content: projectSystem,
        remote_path: '.pi/system-prompt.md',
      })
      setProjectSystemDirty(false)
      setProjectSync('synced')
      const cfgs = await listPiConfigs(serverId)
      setConfigs(cfgs)
    } catch {
      setProjectSync('error')
    }
  }, [serverId, projectSystem])

  const saveSkills = useCallback(async () => {
    setSkillsSync('syncing')
    try {
      for (const skill of skills) {
        await savePiConfig(serverId, {
          config_type: 'skill',
          name: skill.name,
          content: skill.enabled ? 'enabled' : 'disabled',
        })
      }
      setSkillsDirty(false)
      setSkillsSync('synced')
      const cfgs = await listPiConfigs(serverId)
      setConfigs(cfgs)
      applyConfigsToState(cfgs)
    } catch {
      setSkillsSync('error')
    }
  }, [serverId, skills])

  // ── Sync all ──

  const handleSyncAll = useCallback(async () => {
    setGlobalSync('syncing')
    try {
      await syncPiConfigs(serverId)
      setGlobalSync('synced')
      const cfgs = await listPiConfigs(serverId)
      setConfigs(cfgs)
      applyConfigsToState(cfgs)
    } catch {
      setGlobalSync('error')
    }
  }, [serverId])

  // ── Apply template ──

  const handleApplyTemplate = useCallback(async () => {
    if (!selectedTemplate) return
    setApplyingTemplate(true)
    try {
      await applyPiTemplate(serverId, selectedTemplate)
      await loadData()
    } catch (err: any) {
      setError(err.message ?? 'Failed to apply template')
    } finally {
      setApplyingTemplate(false)
    }
  }, [serverId, selectedTemplate, loadData])

  // ── Skill toggle ──

  const toggleSkill = useCallback((name: string) => {
    setSkills(prev => prev.map(s =>
      s.name === name ? { ...s, enabled: !s.enabled } : s
    ))
    setSkillsDirty(true)
  }, [])

  // ── Render ──

  if (loading) {
    return (
      <div class="datai-page">
        <div class="datai-page-content pi-config-loading">Loading Pi configuration...</div>
      </div>
    )
  }

  if (error && !server) {
    return (
      <div class="datai-page">
        <div class="datai-page-content pi-config-error">{error}</div>
      </div>
    )
  }

  return (
    <div class="datai-page">
      <div class="datai-page-header">
        <h1>Pi Config — {server?.name ?? serverId}</h1>
        {server?.pi_installed && (
          <span class="pi-config-version">Pi {server.pi_version || 'installed'}</span>
        )}
        {server && !server.pi_installed && (
          <span class="pi-config-not-installed">Pi not installed</span>
        )}
      </div>

      {error && <div class="pi-config-alert pi-config-alert-error">{error}</div>}

      <div class="datai-page-content pi-config-content">
        {/* ── Template picker ── */}
        <section class="pi-config-section">
          <div class="pi-config-section-label">Template</div>
          <div class="pi-config-template-row">
            <select
              class="pi-config-select"
              value={selectedTemplate}
              onChange={e => setSelectedTemplate((e.target as HTMLSelectElement).value)}
            >
              <option value="">— Select a template —</option>
              {templates.map(t => (
                <option key={t.id} value={t.id}>
                  {t.name}{t.is_builtin ? ' (builtin)' : ''}
                </option>
              ))}
            </select>
            <button
              class="pi-config-btn"
              disabled={!selectedTemplate || applyingTemplate}
              onClick={() => void handleApplyTemplate()}
            >
              {applyingTemplate ? 'Applying...' : 'Apply Template'}
            </button>
          </div>
          {selectedTemplate && (
            <div class="pi-config-template-desc">
              {templates.find(t => t.id === selectedTemplate)?.description ?? ''}
            </div>
          )}
        </section>

        {/* ── System Prompt ── */}
        <section class="pi-config-section">
          <div class="pi-config-section-header">
            <div class="pi-config-section-label">System Prompt</div>
            <SyncBadge status={systemPromptDirty ? 'pending' : configSyncStatus('system_prompt')} />
          </div>
          <textarea
            class="pi-config-textarea"
            rows={10}
            value={systemPrompt}
            onInput={e => {
              setSystemPrompt((e.target as HTMLTextAreaElement).value)
              setSystemPromptDirty(true)
              setPromptSync('idle')
            }}
            placeholder="Enter the system prompt for Pi agent..."
          />
          <div class="pi-config-section-footer">
            <span class="pi-config-path">~/.config/pi/system-prompt.md</span>
            <button
              class="pi-config-btn pi-config-btn-save"
              disabled={!systemPromptDirty || promptSync === 'syncing'}
              onClick={() => void saveSystemPrompt()}
            >
              {promptSync === 'syncing' ? 'Saving...' : 'Save'}
            </button>
          </div>
        </section>

        {/* ── Skills ── */}
        <section class="pi-config-section">
          <div class="pi-config-section-header">
            <div class="pi-config-section-label">Skills</div>
            <SyncBadge status={skillsDirty ? 'pending' : configSyncStatus('skill')} />
          </div>
          <div class="pi-config-skills-grid">
            {skills.map(skill => (
              <label key={skill.name} class="pi-config-skill-item">
                <input
                  type="checkbox"
                  checked={skill.enabled}
                  onChange={() => toggleSkill(skill.name)}
                />
                <span>{skill.name}</span>
              </label>
            ))}
          </div>
          <div class="pi-config-section-footer">
            <span />
            <button
              class="pi-config-btn pi-config-btn-save"
              disabled={!skillsDirty || skillsSync === 'syncing'}
              onClick={() => void saveSkills()}
            >
              {skillsSync === 'syncing' ? 'Saving...' : 'Save Skills'}
            </button>
          </div>
        </section>

        {/* ── Project system.md ── */}
        <section class="pi-config-section">
          <div class="pi-config-section-header">
            <div class="pi-config-section-label">Project Instructions</div>
            <SyncBadge status={projectSystemDirty ? 'pending' : configSyncStatus('project_system')} />
          </div>
          <textarea
            class="pi-config-textarea"
            rows={8}
            value={projectSystem}
            onInput={e => {
              setProjectSystem((e.target as HTMLTextAreaElement).value)
              setProjectSystemDirty(true)
              setProjectSync('idle')
            }}
            placeholder="Project-level instructions (saved to .pi/system-prompt.md)..."
          />
          <div class="pi-config-section-footer">
            <span class="pi-config-path">.pi/system-prompt.md</span>
            <button
              class="pi-config-btn pi-config-btn-save"
              disabled={!projectSystemDirty || projectSync === 'syncing'}
              onClick={() => void saveProjectSystem()}
            >
              {projectSync === 'syncing' ? 'Saving...' : 'Save'}
            </button>
          </div>
        </section>

        {/* ── Sync All ── */}
        <section class="pi-config-section pi-config-sync-section">
          <button
            class={`pi-config-btn pi-config-btn-sync${globalSync === 'syncing' ? ' syncing' : ''}`}
            disabled={globalSync === 'syncing'}
            onClick={() => void handleSyncAll()}
          >
            {globalSync === 'syncing' ? 'Syncing to Server...' : 'Sync All to Server'}
          </button>
          {globalSync === 'synced' && <span class="pi-config-sync-ok">All configs synced</span>}
          {globalSync === 'error' && <span class="pi-config-sync-fail">Sync failed</span>}
        </section>
      </div>
    </div>
  )
}

// ── Sync Badge ──

function SyncBadge({ status }: { status: 'synced' | 'pending' | 'new' }) {
  const labels: Record<string, string> = {
    synced: 'synced',
    pending: 'pending',
    new: 'new',
  }
  return (
    <span class={`pi-config-sync-badge pi-config-sync-${status}`}>
      {labels[status]}
    </span>
  )
}

export default PiConfigPage
