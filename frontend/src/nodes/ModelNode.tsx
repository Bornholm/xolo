import type { NodeProps } from '@xyflow/react'
import { DeleteButton } from './DeleteButton'
import { InputPortRow, OutputPortRow } from './PortRow'

export function ModelNode({ id }: NodeProps) {
  return (
    <div className="pipeline-node pipeline-node--model">
      <DeleteButton nodeId={id} />
      <div className="pipeline-node__header">
        <span className="pipeline-node__icon">🤖</span>
        <span className="pipeline-node__label">Modèle LLM</span>
      </div>
      <div className="pipeline-node__ports">
        <div className="pipeline-node__ports-col">
          <InputPortRow portId="request"    label="requête"    portType="request" />
          <InputPortRow portId="model_name" label="nom modèle" portType="string"  />
        </div>
        <div className="pipeline-node__ports-col">
          <OutputPortRow portId="response" label="réponse" portType="response" />
        </div>
      </div>
    </div>
  )
}
