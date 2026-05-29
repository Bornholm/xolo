import { InputPortRow } from './PortRow'

export function SinkNode() {
  return (
    <div className="pipeline-node pipeline-node--sink">
      <div className="pipeline-node__header">
        <span className="pipeline-node__icon">⏹</span>
        <span className="pipeline-node__label">Réponse</span>
      </div>
      <div className="pipeline-node__ports">
        <div className="pipeline-node__ports-col">
          <InputPortRow portId="response" label="réponse" portType="response" />
        </div>
        <div className="pipeline-node__ports-col" />
      </div>
    </div>
  )
}
