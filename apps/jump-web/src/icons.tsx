type IconProps = { class?: string }

const S = {
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': '1.5',
  'stroke-linecap': 'round' as const,
  'stroke-linejoin': 'round' as const,
}

export function IconPlus({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M7 3v8M3 7h8" /></svg>
}

export function IconPlay({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" fill="currentColor"><path d="M4.5 3.2v7.6L10.5 7z" /></svg>
}

export function IconSettings({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M3 4h8M3 10h8" /><path d="M5 2.5v3M9 8.5v3" /></svg>
}

export function IconFolder({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M2 4.5h4l1 1H12v5.5H2z" /><path d="M2 4.5V3h3.5l1 1.5" /></svg>
}

export function IconCpu({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><rect x="4" y="4" width="6" height="6" rx="1" /><path d="M2 5h2M2 9h2M10 5h2M10 9h2M5 2v2M9 2v2M5 10v2M9 10v2" /></svg>
}

export function IconMemory({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M3 4h8v6H3z" /><path d="M4.5 4V2.5M7 4V2.5M9.5 4V2.5M4.5 11.5V10M7 11.5V10M9.5 11.5V10" /></svg>
}

export function IconBattery({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M2 5h8v4H2z" /><path d="M10 6.2h1.5v1.6H10" /><path d="M3.5 6.4v1.2" /></svg>
}

export function IconActivity({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M2 8h2l1.2-3 2 6 1.4-4H12" /></svg>
}

export function IconSun({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><circle cx="7" cy="7" r="2.2" /><path d="M7 1.8v1.1M7 11.1v1.1M1.8 7h1.1M11.1 7h1.1M3.3 3.3l.8.8M9.9 9.9l.8.8M10.7 3.3l-.8.8M4.1 9.9l-.8.8" /></svg>
}

export function IconHelp({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><circle cx="7" cy="7" r="5" /><path d="M5.6 5.4a1.5 1.5 0 0 1 2.8.7c0 1.3-1.4 1.3-1.4 2.3" /><path d="M7 10.7h.01" /></svg>
}

export function IconAlert({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M7 2.2 12 11H2z" /><path d="M7 5.4v2.5M7 10.1h.01" /></svg>
}

export function IconMoon({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M10.8 9.4A4.7 4.7 0 0 1 4.6 3.2a4.9 4.9 0 1 0 6.2 6.2z" /></svg>
}

export function IconRestart({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M11 6a4 4 0 1 1-1.2-2.9" /><path d="M11 2.5V6H7.5" /></svg>
}

export function IconTrash({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M3 4h8" /><path d="M5 4V2.8h4V4" /><path d="M4 4.8 4.5 12h5L10 4.8" /><path d="M6 6.5v3M8 6.5v3" /></svg>
}

export function IconCopy({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><rect x="4.5" y="3.5" width="6.5" height="7.5" rx="1" /><path d="M3 9.5V2.8h6" /></svg>
}

export function IconScreen({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><rect x="2.3" y="3" width="9.4" height="6.4" rx="1" /><path d="M5.3 11h3.4M7 9.4V11" /></svg>
}

export function IconX({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" {...S}><path d="M4 4l6 6M10 4l-6 6" /></svg>
}

export function IconDots({ class: className }: IconProps) {
  return <svg class={className} viewBox="0 0 14 14" aria-hidden="true" fill="currentColor"><circle cx="7" cy="3.5" r="1" /><circle cx="7" cy="7" r="1" /><circle cx="7" cy="10.5" r="1" /></svg>
}
