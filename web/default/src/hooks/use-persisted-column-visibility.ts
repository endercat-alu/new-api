/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useEffect, useRef, useState } from 'react'
import type { OnChangeFn, VisibilityState } from '@tanstack/react-table'

function readColumnVisibility(storageKey: string | undefined): VisibilityState {
  if (!storageKey || typeof window === 'undefined') return {}

  try {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) return {}

    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {}
    }

    return Object.entries(parsed).reduce<VisibilityState>(
      (visibility, [key, value]) => {
        if (typeof value === 'boolean') {
          visibility[key] = value
        }
        return visibility
      },
      {}
    )
  } catch {
    return {}
  }
}

function writeColumnVisibility(
  storageKey: string | undefined,
  visibility: VisibilityState
) {
  if (!storageKey || typeof window === 'undefined') return
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(visibility))
  } catch {
    /* ignore quota / private mode */
  }
}

/**
 * Column visibility state with optional localStorage persistence.
 * `initial` is the default when no stored value exists; stored keys overlay it.
 */
export function usePersistedColumnVisibility(
  storageKey: string | false | undefined,
  initial: VisibilityState = {}
): [VisibilityState, OnChangeFn<VisibilityState>] {
  const key = typeof storageKey === 'string' ? storageKey : undefined
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    () => ({
      ...initial,
      ...readColumnVisibility(key),
    })
  )
  const hydratedKeyRef = useRef(key)
  const skipNextPersistRef = useRef(false)

  useEffect(() => {
    if (key === hydratedKeyRef.current) return
    hydratedKeyRef.current = key
    skipNextPersistRef.current = true
    setColumnVisibility({
      ...initial,
      ...readColumnVisibility(key),
    })
    // Only rehydrate when the storage key changes; `initial` is first-paint defaults.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])

  useEffect(() => {
    if (skipNextPersistRef.current) {
      skipNextPersistRef.current = false
      return
    }
    writeColumnVisibility(key, columnVisibility)
  }, [columnVisibility, key])

  const onColumnVisibilityChange = useCallback<OnChangeFn<VisibilityState>>(
    (updater) => {
      setColumnVisibility((prev) =>
        typeof updater === 'function' ? updater(prev) : updater
      )
    },
    []
  )

  return [columnVisibility, onColumnVisibilityChange]
}
