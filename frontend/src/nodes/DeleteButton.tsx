import { useReactFlow } from '@xyflow/react'
import { useReadonly } from '../ReadonlyContext'

interface DeleteButtonProps {
  nodeId: string
}

export function DeleteButton({ nodeId }: DeleteButtonProps) {
  const { deleteElements } = useReactFlow()
  const readonly = useReadonly()

  if (readonly) return null

  function handleDelete(e: React.MouseEvent) {
    e.stopPropagation()
    deleteElements({ nodes: [{ id: nodeId }] })
  }

  return (
    <button
      className="pipeline-node__delete"
      onClick={handleDelete}
      title="Supprimer ce nœud"
    >
      ✕
    </button>
  )
}
