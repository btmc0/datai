import { useEffect, useState } from 'preact/hooks'

export interface HostMetrics {
  cpu_percent: number
  memory: {
    used_bytes: number
    total_bytes: number
    percent: number
  }
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function parseHostMetrics(value: unknown): HostMetrics | null {
  if (!value || typeof value !== 'object') return null
  const metrics = value as Partial<HostMetrics>
  const memory = metrics.memory
  if (!memory || typeof memory !== 'object') return null

  if (!isFiniteNumber(metrics.cpu_percent)
      || !isFiniteNumber(memory.used_bytes)
      || !isFiniteNumber(memory.total_bytes)
      || !isFiniteNumber(memory.percent)) {
    return null
  }

  return {
    cpu_percent: metrics.cpu_percent,
    memory: {
      used_bytes: memory.used_bytes,
      total_bytes: memory.total_bytes,
      percent: memory.percent,
    },
  }
}

export function formatPercent(value: number): string {
  return `${value.toFixed(1)}%`
}

export function formatBytes(bytes: number): string {
  const gib = bytes / 1024 / 1024 / 1024
  return `${gib.toFixed(gib >= 10 ? 0 : 1)} GiB`
}

export function useHostMetrics(): HostMetrics | null {
  const [metrics, setMetrics] = useState<HostMetrics | null>(null)

  useEffect(() => {
    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | null = null
    let controller: AbortController | null = null

    const schedule = () => {
      if (!cancelled) timer = setTimeout(refresh, 5000)
    }

    const refresh = async () => {
      controller = new AbortController()
      try {
        const resp = await fetch('/v1/host-metrics', { signal: controller.signal })
        if (!resp.ok) return

        const json = await resp.json()
        const next = json.ok ? parseHostMetrics(json.data) : null
        if (!cancelled && next) setMetrics(next)
      } catch {
        // Host metrics are informational; leave the UI unchanged on failure.
      } finally {
        controller = null
        schedule()
      }
    }

    void refresh()
    return () => {
      cancelled = true
      if (timer !== null) clearTimeout(timer)
      controller?.abort()
    }
  }, [])

  return metrics
}
