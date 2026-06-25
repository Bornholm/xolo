import type { NodeProps } from '@xyflow/react'
import { useState } from 'react'
import { useReactFlow } from '@xyflow/react'
import type { PluginNodeData, NodeTypeDescriptor } from '../types'
import { DeleteButton } from './DeleteButton'
import { PluginUIModal } from '../components/PluginUIModal'
import { InputPortRow, OutputPortRow } from './PortRow'
import { useReadonly } from '../ReadonlyContext'

interface PluginNodeProps extends NodeProps {
  data: PluginNodeData & { __descriptor?: NodeTypeDescriptor }
}

export function PluginNode({ id, data }: PluginNodeProps) {
  const readonly = useReadonly()
  const desc = data.__descriptor
  const cfg = data.config as { inputs?: Array<{ name: string; portType?: string }>; outputs?: Array<{ name: string; portType?: string }> } | undefined
  const inputPorts: Array<{ name: string; port_type: string }> = cfg?.inputs
    ? cfg.inputs.map(i => ({ name: i.name, port_type: i.portType ?? 'number' }))
    : (desc?.inputPorts ?? [])
  const outputPorts: Array<{ name: string; port_type: string }> = cfg?.outputs
    ? cfg.outputs.map(o => ({ name: o.name, port_type: o.portType ?? 'number' }))
    : (desc?.outputPorts ?? [])
  const hasPostResponse = desc?.capabilities?.includes('POST_RESPONSE')
  const [showModal, setShowModal] = useState(false)
  const { updateNodeData } = useReactFlow()

  const baseUrl = document.getElementById('pipeline-editor-root')?.dataset.apiBaseUrl ?? ''

  function handleModalClose(newConfig: Record<string, unknown> | null) {
    setShowModal(false)
    if (newConfig && Object.keys(newConfig).length > 0) {
      updateNodeData(id, { config: newConfig })
    }
  }

  return (
    <div className="pipeline-node pipeline-node--plugin">
      <DeleteButton nodeId={id} />

      <div className="pipeline-node__header">
        <span className="pipeline-node__icon">🔌</span>
        <span className="pipeline-node__label">{desc?.label ?? data.pluginName}</span>
        {hasPostResponse && <span title="Supporte la passe-retour" style={{ marginLeft: 4 }}>↩</span>}
      </div>

      {desc?.hasUI && !readonly && (
        <div className="pipeline-node__subheader">
          <button className="plugin-node__config-btn" onClick={() => setShowModal(true)}>
            ⚙ Configurer
          </button>
        </div>
      )}

      {(inputPorts.length > 0 || outputPorts.length > 0) && (
        <div className="pipeline-node__ports">
          <div className="pipeline-node__ports-col">
            {inputPorts.map(p => (
              <InputPortRow key={p.name} portId={p.name} portType={p.port_type} />
            ))}
          </div>
          <div className="pipeline-node__ports-col">
            {outputPorts.map(p => (
              <OutputPortRow key={p.name} portId={p.name} portType={p.port_type} />
            ))}
          </div>
        </div>
      )}

      {showModal && (
        <PluginUIModal
          pluginName={data.pluginName}
          nodeId={id}
          currentConfig={(data.config as Record<string, unknown>) ?? {}}
          baseUrl={baseUrl}
          onClose={handleModalClose}
        />
      )}
    </div>
  )
}
