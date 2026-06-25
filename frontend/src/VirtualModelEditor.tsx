import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type Connection,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { GeneratorNode } from './nodes/GeneratorNode'
import { SinkNode } from './nodes/SinkNode'
import { ModelNode } from './nodes/ModelNode'
import { ValueNode } from './nodes/ValueNode'
import { PluginNode } from './nodes/PluginNode'
import { NodePalette } from './components/NodePalette'
import { fetchVirtualModel, fetchNodeTypes, updateVirtualModel, exportVirtualModelURL, vmId, orgSlug, isReadonly } from './api'
import type { NodeTypeDescriptor, PipelineGraph, PipelineNode, PipelineEdge, PipelineBundle, PluginNodeData } from './types'
import { ReadonlyContext } from './ReadonlyContext'

const nodeTypes = {
  generator: GeneratorNode,
  sink: SinkNode,
  model: ModelNode,
  value: ValueNode,
  plugin: PluginNode,
}

let idCounter = 1
function nextId() {
  return `node-${Date.now()}-${idCounter++}`
}

function graphToFlow(graph: PipelineGraph | undefined, descriptors: NodeTypeDescriptor[]): { nodes: Node[]; edges: Edge[] } {
  if (!graph) {
    return {
      nodes: [
        { id: 'gen', type: 'generator', position: { x: 80, y: 200 }, data: {}, deletable: false },
        { id: 'sink', type: 'sink', position: { x: 600, y: 200 }, data: {}, deletable: false },
      ],
      edges: [],
    }
  }

  const descMap = new Map(descriptors.map(d => [d.pluginName ?? d.type, d]))

  const nodes: Node[] = graph.nodes.map(n => ({
    id: n.id,
    type: n.type,
    position: n.position,
    deletable: n.type !== 'generator' && n.type !== 'sink',
    data: n.type === 'plugin'
      ? { ...(n.data ?? {}), __descriptor: descMap.get((n.data as PluginNodeData)?.pluginName) }
      : (n.data ?? {}),
  }))

  const edges: Edge[] = graph.edges.map(e => ({
    id: e.id,
    source: e.source,
    sourceHandle: e.sourcePort,
    target: e.target,
    targetHandle: e.targetPort,
  }))

  return { nodes, edges }
}

function flowToGraph(nodes: Node[], edges: Edge[]): PipelineGraph {
  const pNodes: PipelineNode[] = nodes.map(n => ({
    id: n.id,
    type: n.type as PipelineNode['type'],
    position: n.position,
    data: n.type === 'plugin'
      ? (({ __descriptor: _d, ...rest }) => rest)(n.data as Record<string, unknown>)
      : n.data,
  }))

  const pEdges: PipelineEdge[] = edges.map(e => ({
    id: e.id,
    source: e.source,
    sourcePort: e.sourceHandle ?? '',
    target: e.target,
    targetPort: e.targetHandle ?? '',
  }))

  return { nodes: pNodes, edges: pEdges }
}

export function VirtualModelEditor() {
  const id = vmId()
  const slug = orgSlug()
  const readonly = isReadonly()

  const [vmName, setVmName] = useState('')
  const [vmDescription, setVmDescription] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [descriptors, setDescriptors] = useState<NodeTypeDescriptor[]>([])
  const importInputRef = useRef<HTMLInputElement>(null)

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

  const descriptorsRef = useRef<NodeTypeDescriptor[]>([])
  descriptorsRef.current = descriptors

  useEffect(() => {
    async function load() {
      try {
        const [nts] = await Promise.all([fetchNodeTypes()])
        setDescriptors(nts)

        if (id) {
          const vm = await fetchVirtualModel(id)
          setVmName(vm.name)
          setVmDescription(vm.description ?? '')
          const { nodes: n, edges: e } = graphToFlow(vm.graph, nts)
          setNodes(n)
          setEdges(e)
        }
      } catch (err) {
        setError(String(err))
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [id, setNodes, setEdges])

  const onConnect = useCallback(
    (connection: Connection) => {
      setEdges(eds => addEdge(connection, eds))
    },
    [setEdges]
  )

  function addNode(desc: NodeTypeDescriptor) {
    const newId = nextId()
    const newNode: Node = {
      id: newId,
      type: desc.type,
      position: { x: 300 + Math.random() * 100, y: 200 + Math.random() * 100 },
      deletable: true,
      data: desc.type === 'plugin'
        ? { pluginName: desc.pluginName, __descriptor: desc }
        : desc.type === 'expr-router'
        ? { expression: '' }
        : {},
    }
    setNodes(nds => [...nds, newNode])
  }

  async function save() {
    if (!id) return
    setSaving(true)
    setError(null)
    try {
      const graph = flowToGraph(nodes, edges)
      await updateVirtualModel(id, { graph })
    } catch (err) {
      setError(String(err))
    } finally {
      setSaving(false)
    }
  }

  function exportPipeline() {
    if (!id) return
    const graph = flowToGraph(nodes, edges)
    const bundle: PipelineBundle = {
      version: '1',
      exportedAt: new Date().toISOString(),
      name: vmName,
      description: vmDescription,
      graph,
    }
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `pipeline-${vmName.replace(/[/\\:*?"<>|]/g, '-')}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  function triggerImport() {
    importInputRef.current?.click()
  }

  async function handleImportFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setError(null)
    try {
      const text = await file.text()
      const bundle = JSON.parse(text) as PipelineBundle
      if (!bundle.graph) {
        setError('Le fichier ne contient pas de graphe de pipeline.')
        return
      }
      const { nodes: n, edges: ed } = graphToFlow(bundle.graph, descriptorsRef.current)
      setNodes(n)
      setEdges(ed)
    } catch {
      setError('Impossible de lire le fichier : format invalide.')
    } finally {
      // Reset so the same file can be re-imported if needed.
      e.target.value = ''
    }
  }

  const paletteTypes = useMemo(
    () => descriptors.filter(d => d.type !== 'generator' && d.type !== 'sink'),
    [descriptors]
  )

  if (loading) return <div className="pipeline-loading">Chargement…</div>

  return (
    <ReadonlyContext.Provider value={readonly}>
      <div className="pipeline-editor">
        <header className="pipeline-editor__toolbar">
          <span className="pipeline-editor__vm-name">{vmName}</span>
          {error && <span className="pipeline-editor__error">{error}</span>}
          {!readonly && (
            <>
              <input
                ref={importInputRef}
                type="file"
                accept=".json,application/json"
                style={{ display: 'none' }}
                onChange={handleImportFile}
              />
              <button
                className="pipeline-editor__import-btn"
                onClick={triggerImport}
                title="Charger un pipeline depuis un fichier JSON"
              >
                Importer
              </button>
            </>
          )}
          <button
            className="pipeline-editor__export-btn"
            onClick={exportPipeline}
            disabled={!id}
            title="Télécharger le pipeline courant (incluant la configuration de chaque nœud)"
          >
            Exporter
          </button>
          {!readonly && (
            <button
              className="pipeline-editor__save-btn"
              onClick={save}
              disabled={saving || !id}
            >
              {saving ? 'Enregistrement…' : 'Enregistrer'}
            </button>
          )}
        </header>
        <div className="pipeline-editor__body">
          {!readonly && <NodePalette nodeTypes={paletteTypes} onAddNode={addNode} />}
          <div className="pipeline-editor__canvas">
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={readonly ? undefined : onConnect}
              deleteKeyCode={readonly ? null : ['Backspace', 'Delete']}
              nodesDraggable={!readonly}
              nodesConnectable={!readonly}
              edgesUpdatable={!readonly}
              fitView
            >
              <Background />
              <Controls />
              <MiniMap />
            </ReactFlow>
          </div>
        </div>
      </div>
    </ReadonlyContext.Provider>
  )
}
