// Mirrors Go model.PipelineNodeType
export type PipelineNodeType =
  | 'generator'
  | 'sink'
  | 'model'
  | 'plugin'

// Port type mirrors model.PortType
export type PortType = 'request' | 'response' | 'number' | 'string' | 'boolean'

export interface PortDescriptor {
  name: string
  port_type: PortType
  required?: boolean
}

export interface NodeTypeDescriptor {
  type: PipelineNodeType
  pluginName?: string
  label: string
  description: string
  inputPorts: PortDescriptor[]
  outputPorts: PortDescriptor[]
  configSchema?: string
  hasUI?: boolean
}

// Pipeline graph structures (mirrors Go model.PipelineGraph)
export interface PipelineGraph {
  nodes: PipelineNode[]
  edges: PipelineEdge[]
}

export interface PipelineNode {
  id: string
  type: PipelineNodeType
  position: { x: number; y: number }
  data?: Record<string, unknown>
}

export interface PipelineEdge {
  id: string
  source: string
  sourcePort: string
  target: string
  targetPort: string
}

export interface VirtualModel {
  id: string
  orgId: string
  name: string
  description: string
  graph?: PipelineGraph
  createdAt: string
  updatedAt: string
}

// Mirrors Go model.PipelineBundle
export interface PipelineBundle {
  version: string
  exportedAt: string
  name: string
  description: string
  graph?: PipelineGraph
}

// Node data payloads
export interface PluginNodeData {
  pluginName: string
  config?: Record<string, unknown>
}

export interface ModelNodeData {
  proxyName?: string
}

