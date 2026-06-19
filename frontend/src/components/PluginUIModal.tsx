import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { orgSlug } from '../api'

interface PluginUIModalProps {
  pluginName: string
  nodeId: string
  currentConfig: Record<string, unknown>
  baseUrl: string
  onClose: (newConfig: Record<string, unknown> | null) => void
}

function isPersonalContext(): boolean {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.contextType === 'personal'
}

async function seedConfig(orgSlugVal: string, pluginName: string, config: Record<string, unknown>, base: string) {
  const url = isPersonalContext()
    ? `${base}/api/personal-plugin-ui-config?plugin=${encodeURIComponent(pluginName)}`
    : `${base}/api/orgs/${orgSlugVal}/plugin-ui-config?plugin=${encodeURIComponent(pluginName)}`
  await fetch(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ configJson: JSON.stringify(config) }),
  })
}

async function readConfig(orgSlugVal: string, pluginName: string, base: string): Promise<Record<string, unknown>> {
  const url = isPersonalContext()
    ? `${base}/api/personal-plugin-ui-config?plugin=${encodeURIComponent(pluginName)}`
    : `${base}/api/orgs/${orgSlugVal}/plugin-ui-config?plugin=${encodeURIComponent(pluginName)}`
  const res = await fetch(url)
  if (!res.ok) return {}
  const data = await res.json() as { configJson: string }
  try { return JSON.parse(data.configJson) } catch { return {} }
}

export function PluginUIModal({ pluginName, nodeId, currentConfig, baseUrl, onClose }: PluginUIModalProps) {
  const [ready, setReady] = useState(false)
  const [closing, setClosing] = useState(false)
  const slug = orgSlug()

  const uiURL = (isPersonalContext()
    ? `${baseUrl}/profile/plugins/${encodeURIComponent(pluginName)}/ui/`
    : `${baseUrl}/orgs/${slug}/plugins/${encodeURIComponent(pluginName)}/ui/`
  ) + `?nodeId=${encodeURIComponent(nodeId)}`

  useEffect(() => {
    seedConfig(slug, pluginName, currentConfig, baseUrl)
      .then(() => setReady(true))
      .catch(() => setReady(true)) // open anyway on error
  }, [])

  async function handleClose() {
    setClosing(true)
    const newConfig = await readConfig(slug, pluginName, baseUrl)
    onClose(newConfig)
  }

  const modal = (
    <div className="plugin-modal-overlay" onClick={e => { if (e.target === e.currentTarget) handleClose() }}>
      <div className="plugin-modal">
        <div className="plugin-modal__header">
          <span className="plugin-modal__title">Configuration — {pluginName}</span>
          <button className="plugin-modal__close" onClick={handleClose} disabled={closing}>
            {closing ? '…' : '✕'}
          </button>
        </div>
        <div className="plugin-modal__body">
          {ready ? (
            <iframe
              src={uiURL}
              className="plugin-modal__iframe"
              title={`Configuration ${pluginName}`}
            />
          ) : (
            <div className="plugin-modal__loading">Chargement…</div>
          )}
        </div>
      </div>
    </div>
  )

  const mountNode = document.getElementById('pipeline-editor-root') ?? document.body
  return createPortal(modal, mountNode)
}
