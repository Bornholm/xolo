import { NodeCard } from './NodeCard'

export function GeneratorNode() {
  return (
    <NodeCard
      kind="generator"
      title="chat.completions"
      subtitle="requête entrante"
      outputs={[{ name: 'request', port_type: 'request' }]}
    />
  )
}
