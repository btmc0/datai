/**
 * DATAI reactive state — Preact signals for server/key/template data.
 *
 * Follows the same pattern as store.ts: signals as sources of truth,
 * async loaders that fetch and update them.
 */

import { signal } from '@preact/signals'
import type { Server, SSHKey, PiTemplate, Conversation } from './datai-api'
import { listServers, listSSHKeys, listTemplates, listConversations } from './datai-api'

export { type Server, type SSHKey, type PiTemplate, type Conversation }

// ── Signals ─────────────────────────────────────────────────────────────────

export const servers = signal<Server[]>([])
export const sshKeys = signal<SSHKey[]>([])
export const templates = signal<PiTemplate[]>([])
export const conversations = signal<Conversation[]>([])

export const serversLoading = signal(false)
export const sshKeysLoading = signal(false)
export const templatesLoading = signal(false)
export const conversationsLoading = signal(false)

// ── Loaders ─────────────────────────────────────────────────────────────────

export async function loadServers(): Promise<void> {
  serversLoading.value = true
  try {
    servers.value = await listServers()
  } catch (err) {
    console.warn('Failed to load servers:', err)
  } finally {
    serversLoading.value = false
  }
}

export async function loadSSHKeys(): Promise<void> {
  sshKeysLoading.value = true
  try {
    sshKeys.value = await listSSHKeys()
  } catch (err) {
    console.warn('Failed to load SSH keys:', err)
  } finally {
    sshKeysLoading.value = false
  }
}

export async function loadTemplates(): Promise<void> {
  templatesLoading.value = true
  try {
    templates.value = await listTemplates()
  } catch (err) {
    console.warn('Failed to load templates:', err)
  } finally {
    templatesLoading.value = false
  }
}

export async function loadConversations(): Promise<void> {
  conversationsLoading.value = true
  try {
    conversations.value = await listConversations()
  } catch (err) {
    console.warn('Failed to load conversations:', err)
  } finally {
    conversationsLoading.value = false
  }
}

/** Load all DATAI data in parallel. Call once from app init. */
export function initDatAIStore(): void {
  loadServers()
  loadSSHKeys()
  loadTemplates()
  loadConversations()
}
