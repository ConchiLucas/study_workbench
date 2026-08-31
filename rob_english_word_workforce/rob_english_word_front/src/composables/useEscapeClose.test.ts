import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useEscapeClose } from './useEscapeClose'

function mountEscapeListener(close: () => void) {
  return mount(defineComponent({
    setup() {
      useEscapeClose(close)
      return () => h('div')
    }
  }))
}

describe('useEscapeClose', () => {
  it('calls close only when Escape is pressed', () => {
    const close = vi.fn()
    const wrapper = mountEscapeListener(close)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))

    expect(close).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('removes the listener when the component unmounts', () => {
    const close = vi.fn()
    const wrapper = mountEscapeListener(close)
    wrapper.unmount()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))

    expect(close).not.toHaveBeenCalled()
  })
})
