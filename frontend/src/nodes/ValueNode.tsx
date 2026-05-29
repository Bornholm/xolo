import type { NodeProps } from '@xyflow/react'
import { useState } from 'react'
import { useReactFlow } from '@xyflow/react'
import { DeleteButton } from './DeleteButton'
import { OutputPortRow } from './PortRow'

type PortType = 'string' | 'number' | 'boolean'

interface ValueNodeData {
  portType?: PortType
  value?: string
}

const TYPE_LABELS: Record<PortType, string> = {
  string: 'abc',
  number: '123',
  boolean: 'T/F',
}

const PORT_TYPE_COLOR: Record<PortType, string> = {
  string: '#10b981',
  number: '#3b82f6',
  boolean: '#8b5cf6',
}

export function ValueNode({ id, data }: NodeProps) {
  const nodeData = data as ValueNodeData
  const { updateNodeData } = useReactFlow()

  const [portType, setPortType] = useState<PortType>(nodeData.portType ?? 'string')
  const [value, setValue] = useState(nodeData.value ?? '')
  const [editing, setEditing] = useState(false)

  function save(newType: PortType, newValue: string) {
    setPortType(newType)
    setValue(newValue)
    updateNodeData(id, { portType: newType, value: newValue })
  }

  const color = PORT_TYPE_COLOR[portType]

  return (
    <div className="pipeline-node pipeline-node--value" style={{ borderColor: color }}>
      <DeleteButton nodeId={id} />

      <div className="pipeline-node__header">
        <span className="pipeline-node__icon">📌</span>
        <span className="pipeline-node__label">Valeur</span>
        <select
          className="value-node__type-select"
          value={portType}
          onChange={e => save(e.target.value as PortType, value)}
        >
          <option value="string">string</option>
          <option value="number">number</option>
          <option value="boolean">boolean</option>
        </select>
      </div>

      <div className="pipeline-node__body">
        {editing ? (
          portType === 'boolean' ? (
            <select
              autoFocus
              className="pipeline-node__input"
              value={value}
              onChange={e => save(portType, e.target.value)}
              onBlur={() => setEditing(false)}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </select>
          ) : (
            <input
              autoFocus
              className="pipeline-node__input"
              type={portType === 'number' ? 'number' : 'text'}
              value={value}
              onChange={e => setValue(e.target.value)}
              onBlur={() => { save(portType, value); setEditing(false) }}
              onKeyDown={e => e.key === 'Enter' && (save(portType, value), setEditing(false))}
              placeholder={portType === 'number' ? '0.7' : 'org/claude'}
            />
          )
        ) : (
          <span
            className="pipeline-node__value pipeline-node__value--code"
            onClick={() => setEditing(true)}
            style={{ color }}
          >
            {value !== '' ? value : <em style={{ opacity: 0.4 }}>cliquer pour saisir</em>}
          </span>
        )}
      </div>

      <div className="pipeline-node__ports">
        <div className="pipeline-node__ports-col" />
        <div className="pipeline-node__ports-col">
          <OutputPortRow portId="value" label={TYPE_LABELS[portType]} portType={portType} />
        </div>
      </div>
    </div>
  )
}
