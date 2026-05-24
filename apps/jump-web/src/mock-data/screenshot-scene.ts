import { type MockSession, ago } from './types'
import type { ProjectItem } from '../types'

const RST = '\x1b[0m'
const DIM = '\x1b[2m'
const BOLD = '\x1b[1m'
const GREEN = '\x1b[32m'
const CYAN = '\x1b[36m'
const ORANGE = '\x1b[38;2;255;95;38m'
const GRAY = '\x1b[90m'
const C1 = '\x1b[0;48;2;43;48;53m'
const C2 = '\x1b[0;1;2;48;2;43;48;53m'
const C3 = '\x1b[0;1;38;2;95;100;106m'
const C4 = '\x1b[0;1;38;2;211;216;222m'
const C5 = '\x1b[0;1;38;2;194;199;205m'
const C6 = '\x1b[0;1;38;2;150;155;161m'
const C7 = '\x1b[0;1;38;2;51;56;62m'
const C8 = '\x1b[0;1;38;2;34;39;45m'
const C9 = '\x1b[0;1;48;2;43;48;53m'
const C10 = '\x1b[0;2;48;2;43;48;53m'

export const SCREENSHOT_PROJECTS: ProjectItem[] = [
  { slug: 'jump', match: [{ path: '~/workspaces/jump' }, { remote: 'github.com/acme/jump' }] },
  { slug: 'agent-lab', match: [{ path: '~/workspaces/agent-lab' }, { remote: 'github.com/acme/agent-lab' }] },
]

export const SCREENSHOT_SESSIONS: MockSession[] = [
  {
    id: 'sess-shot-attention',
    slug: 'attention-state',
    created_at: ago(3),
    command: ['codex'],
    cwd: '/home/demo/workspaces/jump',
    workspace_root: '/home/demo/workspaces/jump',
    remotes: { origin: 'github.com/acme/jump' },
    kind: 'codex',
    alive: true,
    pid: 4102,
    exit_code: null,
    started_at: ago(3),
    exited_at: null,
    title: 'attention state redesign',
    subtitle: 'backend lifecycle + web dots',
    status: { label: '', working: true },
    unread: false,
    socket_path: '/tmp/jump-sessions/screenshot-attention.sock',
    cursorX: 2,
    cursorY: 52,
    terminalAnchor: 'bottom',
    terminal: `
${DIM}╭───────────────────────────────────────╮${RST}
${DIM}│ directory: ${RST}~/workspaces/jump          ${DIM}│${RST}
${DIM}│ branch:    ${RST}fix/attention-state-redesign${DIM} │${RST}
${DIM}╰───────────────────────────────────────╯${RST}

${C1}                                                                                                                                      ${RST}
${C2}› ${C1}nhìn rộng ra, edit by design: fix attention dots, notify policy, and the screenshot scene                              ${RST}
${C1}                                                                                                                                      ${RST}

${DIM}• ${RST}I’m going to map the attention lifecycle before changing code, then patch the shared seam instead of chasing one surface.
  ${CYAN}runner/file events${RST} → ${CYAN}session store${RST} → ${CYAN}notification suppression${RST} → ${CYAN}web dot rendering${RST}

${DIM}• ${BOLD}Explored${RST}
  ${DIM}├ ${CYAN}Search ${RST}attention|unread|notify ${DIM}in ${RST}jumpd${DIM}, ${RST}apps/jump-web${DIM}, ${RST}docs/stories${RST}
  ${DIM}├ ${CYAN}Read   ${RST}store.go${DIM}, ${RST}sessionmeta.go${DIM}, ${RST}notifications.ts${DIM}, ${RST}attention.ts${RST}
  ${DIM}└ ${CYAN}Read   ${RST}terminal.tsx${DIM}, ${RST}screenshot-scene.ts${DIM}, ${RST}screenshot-webui.mjs${RST}

${GRAY}──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────${RST}

${DIM}• ${RST}Invariant found:
  ${GREEN}✓${RST} unread means there is user-visible work since the last focus/read boundary
  ${GREEN}✓${RST} stopped sessions keep attention until the user acknowledges them
  ${GREEN}✓${RST} notification history is pruned without losing unread state
  ${GREEN}✓${RST} every dot surface reads the same normalized attention state

${DIM}• ${BOLD}Changed${RST}
  ${DIM}├ ${CYAN}jumpd       ${RST}centralized attention transitions and pi/sessionmeta stop handling
  ${DIM}├ ${CYAN}web         ${RST}shared dot policy for sidebar, folders, home cards, and project session cards
  ${DIM}├ ${CYAN}notify      ${RST}suppression matrix now preserves histories without duplicating noise
  ${DIM}└ ${CYAN}screenshot  ${RST}frame CSS, resize observer, and 1.5x capture for crisp terminal output

${GRAY}──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────${RST}

${DIM}• ${RST}Verification matrix
  ${GREEN}✓${RST} attention dot surfaces
  ${GREEN}✓${RST} attention policy cases
  ${GREEN}✓${RST} pi stop unread transition
  ${GREEN}✓${RST} notification history pruning
  ${GREEN}✓${RST} zerobyte screenshot frame fit
  ${ORANGE}⠋${RST} regenerate Codex TUI screenshot scene at 1.5x

${DIM}• ${RST}Next check: open the artifact and make sure the terminal reads like an active Codex session, not a tiny transcript footer.

${GRAY}───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────${RST}

${C3}• ${C4}Wo${C5}r${C6}k${C3}i${C7}n${C8}g ${DIM}(11s • esc to interrupt)${RST}

${C1}                                                                                                                           ${RST}
${C9}›${C1} ${C10}Use /skills to list available skills${C1}                                                                                     ${RST}
${C1}                                                                                                                           ${RST}
  ${DIM}gpt-5.4 default · 72% left · ~/workspaces/jump${RST}`,
  },
  {
    id: 'sess-shot-release',
    slug: 'release-notes',
    created_at: ago(14),
    command: ['pi'],
    cwd: '/home/demo/workspaces/jump',
    workspace_root: '/home/demo/workspaces/jump',
    remotes: { origin: 'github.com/acme/jump' },
    kind: 'pi',
    alive: true,
    pid: 3920,
    exit_code: null,
    started_at: ago(14),
    exited_at: null,
    title: 'release notes ready',
    subtitle: 'v1.14.1 patch summary',
    status: null,
    unread: true,
    socket_path: '/tmp/jump-sessions/screenshot-release.sock',
    terminal: `${GREEN}✓${RST} Release notes generated for v1.14.1\n${DIM}Waiting for review…${RST}`,
  },
  {
    id: 'sess-shot-logs',
    slug: 'preview-logs',
    created_at: ago(22),
    command: ['shell'],
    cwd: '/home/demo/workspaces/jump',
    workspace_root: '/home/demo/workspaces/jump',
    remotes: { origin: 'github.com/acme/jump' },
    kind: 'shell',
    alive: true,
    pid: 2711,
    exit_code: null,
    started_at: ago(22),
    exited_at: null,
    title: 'preview logs',
    subtitle: 'local web build',
    status: null,
    unread: false,
    mockActive: true,
    socket_path: '/tmp/jump-sessions/screenshot-logs.sock',
    terminal: `${DIM}$${RST} pnpm --filter @jump/web dev\n${GREEN}ready${RST} http://127.0.0.1:5173`,
  },
  {
    id: 'sess-shot-agent-lab',
    slug: 'agent-routing',
    created_at: ago(28),
    command: ['codex'],
    cwd: '/home/demo/workspaces/agent-lab',
    workspace_root: '/home/demo/workspaces/agent-lab',
    remotes: { origin: 'github.com/acme/agent-lab' },
    kind: 'codex',
    alive: true,
    pid: 1804,
    exit_code: null,
    started_at: ago(28),
    exited_at: null,
    title: 'agent routing review',
    subtitle: 'remote worker policy',
    status: { label: '', working: true },
    unread: false,
    socket_path: '/tmp/jump-sessions/screenshot-agent.sock',
    terminal: `${DIM}Reviewing routing policy…${RST}`,
  },
]
