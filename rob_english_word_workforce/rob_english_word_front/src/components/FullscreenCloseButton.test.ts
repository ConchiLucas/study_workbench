import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FullscreenCloseButton from './FullscreenCloseButton.vue'

describe('FullscreenCloseButton', () => {
  it('renders an accessible close button and emits close', async () => {
    const wrapper = mount(FullscreenCloseButton)
    const button = wrapper.get('button')

    expect(button.attributes('type')).toBe('button')
    expect(button.attributes('aria-label')).toBe('关闭')
    expect(button.text()).toBe('×')

    await button.trigger('click')

    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('does not emit close while disabled', async () => {
    const wrapper = mount(FullscreenCloseButton, {
      props: { disabled: true }
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('does not bubble the native click to an overlay', async () => {
    const wrapper = mount({
      components: { FullscreenCloseButton },
      template: '<div class="overlay" @click="overlayClicks++"><FullscreenCloseButton @close="closeClicks++" /></div>',
      data: () => ({ overlayClicks: 0, closeClicks: 0 })
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.vm.closeClicks).toBe(1)
    expect(wrapper.vm.overlayClicks).toBe(0)
  })
})
