import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ReactFlowProvider } from '@xyflow/react'
import { VirtualModelEditor } from './VirtualModelEditor'
import './style.css'

const root = document.getElementById('pipeline-editor-root')
if (root) {
  createRoot(root).render(
    <StrictMode>
      <ReactFlowProvider>
        <VirtualModelEditor />
      </ReactFlowProvider>
    </StrictMode>
  )
}
