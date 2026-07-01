import type { NodeProps } from '@xyflow/react'
import { DeleteButton } from './DeleteButton'
import { InputPortRow, OutputPortRow } from './PortRow'

// In a middleware pipeline every model node is a passthrough: it wraps the model
// actually requested by the caller (enforced server-side). In other contexts
// (virtual models) the node resolves a model via its model_name input.
function isMiddlewareContext(): boolean {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.contextType === 'middleware'
}

export function ModelNode({ id }: NodeProps) {
  const middleware = isMiddlewareContext()

  return (
    <div className="pipeline-node pipeline-node--model">
      <DeleteButton nodeId={id} />
      <div className="pipeline-node__header">
        <span className="pipeline-node__icon">🤖</span>
        <span className="pipeline-node__label">{middleware ? 'Modèle demandé' : 'Modèle LLM'}</span>
      </div>
      {middleware && (
        <div className="pipeline-node__body">
          <span style={{ fontSize: 12, opacity: 0.7 }}>Applique le traitement au modèle appelé (passthrough)</span>
        </div>
      )}
      <div className="pipeline-node__ports">
        <div className="pipeline-node__ports-col">
          <InputPortRow portId="request" label="requête" portType="request" />
          {!middleware && (
            <InputPortRow portId="model_name" label="nom modèle" portType="string" />
          )}
        </div>
        <div className="pipeline-node__ports-col">
          <OutputPortRow portId="response" label="réponse" portType="response" />
        </div>
      </div>
    </div>
  )
}
