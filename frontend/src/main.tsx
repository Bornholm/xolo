import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ReactFlowProvider } from '@xyflow/react'
import { VirtualModelEditor } from './VirtualModelEditor'
import './style.css'

const container = document.getElementById('pipeline-editor-root')
if (container) {
  const root = createRoot(container)
  root.render(
    <StrictMode>
      <ReactFlowProvider>
        <VirtualModelEditor />
      </ReactFlowProvider>
    </StrictMode>
  )

  // L'éditeur vit dans #content, la seule région qu'échange hx-boost, et son
  // <script> est écrit là aussi. Quitter l'écran arrache donc le conteneur sans
  // que React ne l'apprenne : la racine reste montée, avec les écouteurs que
  // React Flow pose sur la fenêtre. Le bundle est un IIFE, réexécuté à chaque
  // insertion du <script> par l'échange, si bien que chaque retour sur l'écran
  // en empile une de plus. On démonte donc à la main, avant l'échange qui
  // détruit le conteneur.
  const dispose = () => {
    document.removeEventListener('htmx:beforeSwap', onBeforeSwap)
    document.removeEventListener('htmx:historyRestore', onHistoryRestore)
    root.unmount()
  }

  const onBeforeSwap = (event: Event) => {
    const target = (event as CustomEvent<{ target?: Node }>).detail?.target
    if (target && target.contains(container)) dispose()
  }

  // La restauration d'historique ne passe par aucun échange : HTMX réécrit le
  // `innerHTML` de <body>, le conteneur est déjà détaché quand l'événement part.
  const onHistoryRestore = () => {
    if (!container.isConnected) dispose()
  }

  document.addEventListener('htmx:beforeSwap', onBeforeSwap)
  document.addEventListener('htmx:historyRestore', onHistoryRestore)
}
