#!/usr/bin/env node
import { chromium } from 'playwright'
import { spawn } from 'node:child_process'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

const repoRoot = path.resolve(import.meta.dirname, '..')
const port = Number(process.env.JUMP_SCREENSHOT_PORT || 4174)
const host = '127.0.0.1'
const baseUrl = `http://${host}:${port}`
const outPath = path.resolve(process.env.JUMP_SCREENSHOT_OUT || 'artifacts/screenshots/jump-webui-zerobyte.png')
const deviceScaleFactor = Number(process.env.JUMP_SCREENSHOT_SCALE || 1.5)
const screenshotTerminalFontSize = Number(process.env.JUMP_SCREENSHOT_TERMINAL_FONT || 10)
const frameScreenshot = process.env.JUMP_SCREENSHOT_FRAME !== '0'

const routePath = '/jump/codex/sess-shot-attention?mock=1&screenshot=1'

function screenshotCss() {
  return `
    *, *::before, *::after {
      animation-delay: 0s !important;
      animation-duration: 1ms !important;
      transition-duration: 0ms !important;
      caret-color: transparent !important;
    }
    .xterm-cursor-layer { display: none !important; }
    ${frameScreenshot ? `
      html, body, #app {
        min-height: 100%;
      }
      body {
        margin: 0 !important;
        background:
          radial-gradient(circle at 24% 18%, rgba(255, 95, 38, 0.20), transparent 34%),
          radial-gradient(circle at 78% 4%, rgba(255, 181, 70, 0.12), transparent 30%),
          linear-gradient(135deg, #070606 0%, #0e0c0b 46%, #17100d 100%) !important;
      }
      #app {
        display: grid !important;
        place-items: end center !important;
        padding: 104px 72px 28px !important;
        box-sizing: border-box !important;
      }
      .app-layout {
        width: 1360px !important;
        height: 920px !important;
        max-width: calc(100vw - 144px) !important;
        max-height: calc(100vh - 132px) !important;
        border: 1px solid rgba(255, 117, 66, 0.28) !important;
        border-radius: 18px !important;
        overflow: hidden !important;
        box-shadow:
          0 44px 120px rgba(0, 0, 0, 0.74),
          0 18px 48px rgba(255, 95, 38, 0.10),
          inset 0 1px 0 rgba(255, 255, 255, 0.08) !important;
      }
      .terminal-container {
        height: 100% !important;
      }
    ` : ''}
  `
}

function startServer() {
  const child = spawn(
    'corepack',
    ['pnpm', '--dir', 'apps/jump-web', 'exec', 'vite', '--host', host, '--port', String(port), '--strictPort'],
    {
      cwd: repoRoot,
      env: { ...process.env, VITE_MOCK: '1' },
      stdio: ['ignore', 'pipe', 'pipe'],
    },
  )

  child.stdout.setEncoding('utf8')
  child.stderr.setEncoding('utf8')
  child.stdout.on('data', chunk => process.stdout.write(chunk))
  child.stderr.on('data', chunk => process.stderr.write(chunk))

  return child
}

async function waitForServer(timeoutMs = 20_000) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const res = await fetch(baseUrl)
      if (res.ok) return
    } catch {
      // keep polling
    }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  throw new Error(`Timed out waiting for ${baseUrl}`)
}

function mockApiPayload(url) {
  const path = new URL(url).pathname
  if (path === '/v1/host-metrics') {
    return {
      ok: true,
      data: {
        cpu_percent: 18,
        memory: { used_bytes: 11_700_000_000, total_bytes: 34_360_000_000, percent: 34.1 },
        battery: { percent: 86, state: 'charging' },
      },
    }
  }
  if (path === '/v1/session-metrics') return { ok: true, data: {} }
  return null
}


async function launchBrowser() {
  const channels = [process.env.JUMP_SCREENSHOT_BROWSER_CHANNEL, undefined, 'chrome', 'chromium', 'msedge'].filter(Boolean)
  let lastError
  for (const channel of channels) {
    try {
      return await chromium.launch(channel ? { channel } : {})
    } catch (err) {
      lastError = err
    }
  }
  throw lastError
}


async function main() {
  const server = startServer()
  let browser
  const apiRequests = []

  try {
    await waitForServer()
    await mkdir(path.dirname(outPath), { recursive: true })

    browser = await launchBrowser()
    const context = await browser.newContext({
      viewport: frameScreenshot ? { width: 1540, height: 1080 } : { width: 1360, height: 920 },
      deviceScaleFactor,
      colorScheme: 'dark',
    })

    await context.addInitScript(() => {
      localStorage.setItem('jump:appearance', JSON.stringify({ theme_id: 'zerobyte' }))
    })

    const page = await context.newPage()
    await page.route('**/v1/**', route => {
      const payload = mockApiPayload(route.request().url())
      if (payload) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(payload),
        })
      }
      apiRequests.push(route.request().url())
      return route.abort('blockedbyclient')
    })

    await page.goto(`${baseUrl}${routePath}`, { waitUntil: 'networkidle' })
    await page.waitForSelector('.app-layout.mock-mode, .mock-mode .app-layout', { timeout: 10_000 }).catch(async () => {
      await page.waitForSelector('.app-layout', { timeout: 10_000 })
    })
    await page.waitForSelector('.terminal-shell', { timeout: 10_000 })
    await page.waitForFunction(() => document.documentElement.dataset.theme === 'zerobyte')

    await page.addStyleTag({ content: screenshotCss() })
    await page.evaluate(fontSize => {
      const term = window.__jumpTerm
      if (term) term.options.fontSize = fontSize
      window.dispatchEvent(new Event('resize'))
    }, screenshotTerminalFontSize)
    await page.waitForFunction(() => {
      const term = window.__jumpTerm
      const shell = document.querySelector('.terminal-shell')
      const screen = document.querySelector('.terminal-container .xterm-screen')
      if (!term || !shell || !screen) return false
      return screen.getBoundingClientRect().height <= shell.getBoundingClientRect().height + 12
    }, null, { timeout: 5_000 })

    await page.screenshot({ path: outPath, fullPage: false, scale: 'device' })

    if (apiRequests.length > 0) {
      throw new Error(`Screenshot attempted live API requests:\n${apiRequests.join('\n')}`)
    }

    console.log(`Screenshot written: ${path.relative(repoRoot, outPath)} (${deviceScaleFactor}x)`)
  } finally {
    await browser?.close().catch(() => {})
    server.kill('SIGTERM')
    setTimeout(() => server.kill('SIGKILL'), 2_000).unref()
  }
}

main().catch(err => {
  console.error(err)
  process.exit(1)
})
