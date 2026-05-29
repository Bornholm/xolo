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
  return request(`/api/orgs/${orgSlug()}/virtual-models/${id}`)
}

export function updateVirtualModel(
  id: string,
  patch: { description?: string; graph?: PipelineGraph }
): Promise<VirtualModel> {
  return request(`/api/orgs/${orgSlug()}/virtual-models/${id}`, {
    method: 'PUT',
    body: JSON.stringify(patch),
  })
}

export function fetchNodeTypes(): Promise<NodeTypeDescriptor[]> {
  return request(`/api/orgs/${orgSlug()}/pipeline-node-types`)
}

export function exportVirtualModelURL(id: string): string {
  return `${getBase()}/api/orgs/${orgSlug()}/virtual-models/${id}/export`
}

export function importVirtualModel(bundle: PipelineBundle): Promise<VirtualModel> {
  return request(`/api/orgs/${orgSlug()}/virtual-models/import`, {
    method: 'POST',
    body: JSON.stringify(bundle),
  })
}
