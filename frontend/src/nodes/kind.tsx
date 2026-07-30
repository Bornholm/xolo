import type { PipelineNodeType } from '../types'

/**
 * The editor names a node's kind in three places — the palette, the card on the
 * canvas and the inspector — and the mockup shows the same glyph and the same
 * uppercase label in all three. Defining them once is what keeps those three
 * views from drifting apart.
 */

/** Uppercase kind label of the mockup ("GENERATOR", "PLUGIN"…). */
export const KIND_LABEL: Record<PipelineNodeType, string> = {
  generator: 'generator',
  sink: 'sink',
  model: 'model',
  value: 'value',
  plugin: 'plugin',
}

/** One-line description under a built-in kind in the palette. */
export const KIND_DESCRIPTION: Record<PipelineNodeType, string> = {
  generator: 'source : émet la requête',
  sink: 'sortie : reçoit la réponse',
  model: 'appelle un modèle réel',
  value: 'valeur statique typée',
  plugin: 'traitement intercalé',
}

/**
 * Kind glyphs, as inline SVG rather than emoji: emoji render in the system font
 * at a size and weight nothing else in the design system uses, and they carry
 * their own colour, so they cannot follow the node's tint.
 */
export function NodeKindIcon({ kind }: { kind: PipelineNodeType }) {
  const common = {
    width: 12,
    height: 12,
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 2,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
  }

  switch (kind) {
    case 'generator':
      return (
        <svg {...common} aria-hidden="true">
          <rect x="2" y="4" width="20" height="16" rx="2" />
          <path d="M2 10h20" />
        </svg>
      )
    case 'sink':
      return (
        <svg {...common} aria-hidden="true">
          <path d="M12 2 2 7l10 5 10-5-10-5Z" />
          <path d="m2 17 10 5 10-5M2 12l10 5 10-5" />
        </svg>
      )
    case 'model':
      return (
        <svg {...common} aria-hidden="true">
          <path d="M12 5a3 3 0 1 0-5.997.125A4 4 0 0 0 5 13a4 4 0 0 0 .8 6.4A3 3 0 0 0 12 19Z" />
          <path d="M12 5a3 3 0 1 1 5.997.125A4 4 0 0 1 19 13a4 4 0 0 1-.8 6.4A3 3 0 0 1 12 19Z" />
        </svg>
      )
    case 'value':
      return (
        <svg {...common} aria-hidden="true">
          <circle cx="12" cy="12" r="3" />
          <path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
        </svg>
      )
    case 'plugin':
      return (
        <svg {...common} aria-hidden="true">
          <path d="M12 22v-5M9 8V2M15 8V2M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z" />
        </svg>
      )
  }
}
