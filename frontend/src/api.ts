import type { VirtualModel, NodeTypeDescriptor, PipelineGraph, PipelineBundle } from './types'

function getBase(): string {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.apiBaseUrl ?? ''
}

function orgSlug(): string {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.orgSlug ?? ''
}

function vmId(): string | null {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.vmId ?? null
}

function contextType(): string {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.contextType ?? ''
}

function isPersonalContext(): boolean {
  return contextType() === 'personal'
}

function isMiddlewareContext(): boolean {
  return contextType() === 'middleware'
}

// entityBase returns the REST base path of the pipeline-bearing entity being
// edited (virtual model, personal virtual model or middleware).
function entityBase(): string {
  if (isPersonalContext()) return `/api/personal-models`
  if (isMiddlewareContext()) return `/api/orgs/${orgSlug()}/middlewares`
  return `/api/orgs/${orgSlug()}/virtual-models`
}

// nodeTypesBase returns the base path exposing the pipeline node-type catalog.
function nodeTypesBase(): string {
  if (isPersonalContext()) return `/api/personal-models/pipeline-node-types`
  return `/api/orgs/${orgSlug()}/pipeline-node-types`
}

export function isReadonly(): boolean {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.readonly === 'true'
}

export { vmId, orgSlug }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(getBase() + path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`API error ${res.status}: ${text}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export function fetchVirtualModel(id: string): Promise<VirtualModel> {
  return request(`${entityBase()}/${id}`)
}

export function updateVirtualModel(
  id: string,
  patch: { description?: string; graph?: PipelineGraph }
): Promise<VirtualModel> {
  return request(`${entityBase()}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(patch),
  })
}

export function fetchNodeTypes(): Promise<NodeTypeDescriptor[]> {
  return request(nodeTypesBase())
}

export function exportVirtualModelURL(id: string): string {
  return `${getBase()}${entityBase()}/${id}/export`
}

export function importVirtualModel(bundle: PipelineBundle): Promise<VirtualModel> {
  return request(`${entityBase()}/import`, {
    method: 'POST',
    body: JSON.stringify(bundle),
  })
}
