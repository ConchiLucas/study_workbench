<template>
  <button
    class="fullscreen-close-button"
    type="button"
    :aria-label="label"
    :title="label"
    :disabled="disabled"
    @click.stop="emit('close')"
  >
    <span aria-hidden="true">×</span>
  </button>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  disabled?: boolean
  label?: string
}>(), {
  disabled: false,
  label: '关闭'
})

const emit = defineEmits<{
  (event: 'close'): void
}>()
</script>

<style scoped>
.fullscreen-close-button {
  position: fixed;
  top: 24px;
  right: 28px;
  z-index: 3000;
  display: grid;
  width: 46px;
  height: 46px;
  padding: 0;
  place-items: center;
  color: rgba(229, 231, 235, 0.74);
  font: 300 34px/1 Arial, sans-serif;
  background: rgba(15, 18, 28, 0.58);
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 50%;
  cursor: pointer;
  backdrop-filter: blur(10px);
  transition: color 0.18s ease, background 0.18s ease, border-color 0.18s ease, transform 0.18s ease;
}

.fullscreen-close-button:hover:not(:disabled) {
  color: #ffffff;
  background: rgba(51, 65, 85, 0.72);
  border-color: rgba(167, 139, 250, 0.55);
  transform: scale(1.04);
}

.fullscreen-close-button:focus-visible {
  outline: 3px solid rgba(139, 92, 246, 0.5);
  outline-offset: 3px;
}

.fullscreen-close-button:disabled {
  opacity: 0.36;
  cursor: not-allowed;
}

@media (max-width: 640px) {
  .fullscreen-close-button {
    top: 14px;
    right: 14px;
    width: 44px;
    height: 44px;
    font-size: 31px;
  }
}
</style>
