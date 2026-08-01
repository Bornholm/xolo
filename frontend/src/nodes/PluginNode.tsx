import type { NodeProps } from '@xyflow/react'
import type { PluginNodeData, NodeTypeDescriptor } from '../types'
import { NodeCard, type CardPort } from './NodeCard'
import { pluginPorts } from './ports'

interface PluginNodeProps extends NodeProps {
  data: PluginNodeData & { __descriptor?: NodeTypeDescriptor }
}

export function PluginNode({ data }: PluginNodeProps) {
  const desc = data.__descriptor
  const { inputs, outputs } = pluginPorts(data, desc)

  return (
    <NodeCard
      kind="plugin"
      title={data.pluginName}
      subtitle={desc?.version ? `v${desc.version}` : ''}
      inputs={inputs as CardPort[]}
      outputs={outputs as CardPort[]}
    />
  )
}
