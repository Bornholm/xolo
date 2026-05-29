import { Handle, Position } from '@xyflow/react'

export type PortType = 'request' | 'response' | 'string' | 'number' | 'boolean'

export const PORT_COLOR: Record<string, string> = {
  request: '#f59e0b',
  response: '#ef4444',
  string: '#10b981',
  number: '#3b82f6',
  boolean: '#8b5cf6',
}

// Handle style: positioned at the node border (left: 0 / right: 0 of the row,
// which extends edge-to-edge via negative margin on the parent).
const inputHandleStyle = (color: string): React.CSSProperties => ({
  position: 'absolute',
  left: 0,
  top: '50%',
  transform: 'translate(-50%, -50%)',
  background: color,
})

const outputHandleStyle = (color: string): React.CSSProperties => ({
  position: 'absolute',
  right: 0,
  top: '50%',
  transform: 'translate(50%, -50%)',
  background: color,
})

interface InputPortRowProps {
  portId: string
  label?: string
  portType: string
}

export function InputPortRow({ portId, label, portType }: InputPortRowProps) {
  const color = PORT_COLOR[portType] ?? '#6b7280'
  return (
    <div className="pipeline-node__port-row pipeline-node__port-row--input">
      <Handle
        type="target"
        position={Position.Left}
        id={portId}
        style={inputHandleStyle(color)}
      />
      <span>{label ?? portId}</span>
    </div>
  )
}

interface OutputPortRowProps {
  portId: string
  label?: string
  portType: string
}

export function OutputPortRow({ portId, label, portType }: OutputPortRowProps) {
  const color = PORT_COLOR[portType] ?? '#6b7280'
  return (
    <div className="pipeline-node__port-row pipeline-node__port-row--output">
      <span>{label ?? portId}</span>
      <Handle
        type="source"
        position={Position.Right}
        id={portId}
        style={outputHandleStyle(color)}
      />
    </div>
  )
}
