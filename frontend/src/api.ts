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

function isPersonalContext(): boolean {
  const root = document.getElementById('pipeline-editor-root')
  return root?.dataset.contextType === 'personal'
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
  if (isPersonalContext()) {
    return request(`/api/personal-models/${id}`)
  }
  return request(`/api/orgs/${orgSlug()}/virtual-models/${id}`)
}

export function updateVirtualModel(
  id: string,
  patch: { description?: string; graph?: PipelineGraph }
): Promise<VirtualModel> {
  if (isPersonalContext()) {
    return request(`/api/personal-models/${id}`, {
      method: 'PUT',
      body: JSON.stringify(patch),
    })
  }
  return request(`/api/orgs/${orgSlug()}/virtual-models/${id}`, {
    method: 'PUT',
    body: JSON.stringify(patch),
  })
}

export function fetchNodeTypes(): Promise<NodeTypeDescriptor[]> {
  if (isPersonalContext()) {
    return request(`/api/personal-models/pipeline-node-types`)
  }
  return request(`/api/orgs/${orgSlug()}/pipeline-node-types`)
}

export function exportVirtualModelURL(id: string): string {
  if (isPersonalContext()) {
    return `${getBase()}/api/personal-models/${id}/export`
  }
  return `${getBase()}/api/orgs/${orgSlug()}/virtual-models/${id}/export`
}

export function importVirtualModel(bundle: PipelineBundle): Promise<VirtualModel> {
  if (isPersonalContext()) {
    return request(`/api/personal-models/import`, {
      method: 'POST',
      body: JSON.stringify(bundle),
    })
  }
  return request(`/api/orgs/${orgSlug()}/virtual-models/import`, {
    method: 'POST',
    body: JSON.stringify(bundle),
  })
}
