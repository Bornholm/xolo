import type { ReactNode } from 'react'
import type { PipelineNodeType } from '../types'
import { KIND_LABEL, NodeKindIcon } from './kind'
import { InputPortRow, OutputPortRow } from './PortRow'

export interface CardPort {
  name: string
  port_type: string
  label?: string
}

interface NodeCardProps {
  kind: PipelineNodeType
  /** The node's own name — a model proxy name, a plugin name, "requête"… */
  title: string
  /** The grey line under the title: a version, a role, a value type. */
  subtitle?: string
  inputs?: CardPort[]
  outputs?: CardPort[]
  /** Extra content between the title and the ports (the value editor). */
  children?: ReactNode
}

/**
 * NodeCard is the single shape every node on the canvas takes: a kind band in
 * small caps, the node's own name in monospace, an optional grey subtitle, then
 * the ports.
 *
 * The five node components used to each lay this out themselves, which is how
 * they ended up with different paddings and three different notions of what the
 * header holds. The kind band is what the mockup leads with — a reader scans the
 * canvas for "which of these is the model", not for a name they chose.
 */
export function NodeCard({ kind, title, subtitle, inputs = [], outputs = [], children }: NodeCardProps) {
  return (
    <div className={`pipeline-node pipeline-node--${kind}`}>
      <div className="pipeline-node__kind">
        <NodeKindIcon kind={kind} />
        {KIND_LABEL[kind]}
      </div>

      <div className="pipeline-node__identity">
        <div className="pipeline-node__title">{title}</div>
        {subtitle && <div className="pipeline-node__subtitle">{subtitle}</div>}
      </div>

      {children}

      {(inputs.length > 0 || outputs.length > 0) && (
        <div className="pipeline-node__ports">
          <div className="pipeline-node__ports-col">
            {inputs.map(p => (
              <InputPortRow key={p.name} portId={p.name} label={p.label} portType={p.port_type} />
            ))}
          </div>
          <div className="pipeline-node__ports-col">
            {outputs.map(p => (
              <OutputPortRow key={p.name} portId={p.name} label={p.label} portType={p.port_type} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
