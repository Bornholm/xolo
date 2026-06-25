import { createContext, useContext } from 'react'

export const ReadonlyContext = createContext(false)

export function useReadonly(): boolean {
  return useContext(ReadonlyContext)
}
