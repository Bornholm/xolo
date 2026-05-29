import { OutputPortRow } from './PortRow'

export function GeneratorNode() {
  return (
    <div className="pipeline-node pipeline-node--generator">
      <div className="pipeline-node__header">
        <span className="pipeline-node__icon">▶</span>
        <span className="pipeline-node__label">Requête</span>
      </div>
      <div className="pipeline-node__ports">
        <div className="pipeline-node__ports-col" />
        <div className="pipeline-node__ports-col">
          <OutputPortRow portId="request" label="requête" portType="request" />
        </div>
      </div>
    </div>
  )
}
