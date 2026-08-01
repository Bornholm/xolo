import { NodeCard } from './NodeCard'

export function SinkNode() {
  return (
    <NodeCard
      kind="sink"
      title="réponse"
      subtitle="sortie du pipeline"
      inputs={[{ name: 'response', port_type: 'response' }]}
    />
  )
}
