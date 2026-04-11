import { useCallback, useState } from 'react'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: string
  message: string
  type: ToastType
  duration?: number
}

const DEFAULT_DURATION = 5000 // 5 seconds

export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = useCallback(
    (message: string, type: ToastType = 'info', duration = DEFAULT_DURATION) => {
      const id = Math.random().toString(36).substr(2, 9)
      const toast: Toast = { id, message, type, duration }

      setToasts((prev) => [...prev, toast])

      if (duration) {
        setTimeout(() => {
          removeToast(id)
        }, duration)
      }

      return id
    },
    []
  )

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const success = useCallback((message: string) => {
    return addToast(message, 'success')
  }, [addToast])

  const error = useCallback((message: string) => {
    return addToast(message, 'error', 7000)
  }, [addToast])

  const info = useCallback((message: string) => {
    return addToast(message, 'info')
  }, [addToast])

  const warning = useCallback((message: string) => {
    return addToast(message, 'warning')
  }, [addToast])

  return {
    toasts,
    addToast,
    removeToast,
    success,
    error,
    info,
    warning,
  }
}
