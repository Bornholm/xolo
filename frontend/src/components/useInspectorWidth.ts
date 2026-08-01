import { useCallback, useEffect, useRef, useState } from 'react'

/**
 * The inspector has two natural sizes, because it shows two kinds of thing.
 *
 * `compact` is the 290 px of the mockup: enough for a proxy name, a switch and a
 * list of ports. `wide` is for a node whose plugin brings its own configuration
 * document — a third-party page authored without knowing what it is embedded in,
 * which 290 px cannot hold.
 */
export type InspectorMode = 'compact' | 'wide'

export const INSPECTOR_DEFAULT_WIDTH = 290
const DEFAULT_WIDE_WIDTH = 560
const MIN_WIDTH = 260
/** Below this, the canvas stops being usable — the panel must leave it room. */
const MIN_CANVAS_WIDTH = 320

const STORAGE_KEY: Record<InspectorMode, string> = {
  compact: 'xolo.pipeline.inspectorWidth',
  wide: 'xolo.pipeline.inspectorWidthWide',
}

const DEFAULT_WIDTH: Record<InspectorMode, number> = {
  compact: INSPECTOR_DEFAULT_WIDTH,
  wide: DEFAULT_WIDE_WIDTH,
}

function clampWidth(value: number): number {
  const max = Math.max(MIN_WIDTH, window.innerWidth - MIN_CANVAS_WIDTH)
  return Math.min(Math.max(value, MIN_WIDTH), max)
}

function readStored(mode: InspectorMode): number {
  const stored = Number(localStorage.getItem(STORAGE_KEY[mode]))
  return Number.isFinite(stored) && stored >= MIN_WIDTH ? stored : DEFAULT_WIDTH[mode]
}

/**
 * useInspectorWidth sizes the panel for what it currently displays, and lets the
 * user override that by dragging its left edge.
 *
 * Both sizes are remembered separately. Keeping a single width would mean either
 * the panel stays too narrow for plugin UIs, or every node opens a panel wide
 * enough for the widest of them — and a single remembered value would be
 * overwritten every time the selection moved between the two kinds, which is
 * exactly the drag a user would have to redo.
 */
export function useInspectorWidth(mode: InspectorMode) {
  const [widths, setWidths] = useState<Record<InspectorMode, number>>(() => ({
    compact: readStored('compact'),
    wide: readStored('wide'),
  }))

  const dragging = useRef(false)
  const modeRef = useRef(mode)
  modeRef.current = mode

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY[mode], String(widths[mode]))
  }, [mode, widths])

  const onPointerDown = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    event.preventDefault()
    dragging.current = true
    event.currentTarget.setPointerCapture(event.pointerId)

    function onMove(e: PointerEvent) {
      if (!dragging.current) return
      // The panel is docked right, so its width is the distance from the pointer
      // to the right edge of the window. The drag adjusts the size of the mode
      // currently shown, leaving the other one alone.
      const next = clampWidth(window.innerWidth - e.clientX)
      setWidths(current => ({ ...current, [modeRef.current]: next }))
    }

    function onUp() {
      dragging.current = false
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }, [])

  return { width: clampWidth(widths[mode]), onPointerDown }
}
