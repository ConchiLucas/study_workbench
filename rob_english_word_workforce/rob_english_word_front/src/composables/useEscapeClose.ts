import { onBeforeUnmount, onMounted } from 'vue'

export function useEscapeClose(close: () => void) {
  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      close()
    }
  }

  onMounted(() => window.addEventListener('keydown', handleKeydown))
  onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))
}
