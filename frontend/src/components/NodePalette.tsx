import type { NodeTypeDescriptor } from '../types'

interface NodePaletteProps {
  nodeTypes: NodeTypeDescriptor[]
  onAddNode: (descriptor: NodeTypeDescriptor) => void
}

const TYPE_ICONS: Record<string, string> = {
  generator: '▶',
  sink: '⏹',
  model: '🤖',
  'expr-router': '⚡',
  plugin: '🔌',
}

export function NodePalette({ nodeTypes, onAddNode }: NodePaletteProps) {
  return (
    <aside className="pipeline-palette">
      <h3 className="pipeline-palette__title">Nœuds disponibles</h3>
      <ul className="pipeline-palette__list">
        {nodeTypes.map(nd => (
          <li
            key={nd.pluginName ?? nd.type}
            className="pipeline-palette__item"
            onClick={() => onAddNode(nd)}
            title={nd.description}
          >
            <span className="pipeline-palette__icon">{TYPE_ICONS[nd.type] ?? '●'}</span>
            <span className="pipeline-palette__label">{nd.label}</span>
          </li>
        ))}
      </ul>
    </aside>
  )
}
