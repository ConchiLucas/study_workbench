import { useCallback, useRef } from 'react'

/**
 * 答对/答错的音效用 WebAudio 合成，不加载音频文件。
 *
 * 两个原因：一是省掉资源请求，家里断网也照样响；
 * 二是合成音能精确控制时长——音效必须在 400ms 内结束，
 * 拖长了会挡住下一题。
 *
 * 按决策只做对错短音效，不做背景音乐（会干扰专注）。
 */
export function useSound() {
  const ctxRef = useRef<AudioContext | null>(null)

  /** iOS 要求 AudioContext 在用户手势里创建/恢复。 */
  const unlock = useCallback(() => {
    if (!ctxRef.current) {
      const Ctor = window.AudioContext ?? (window as unknown as {
        webkitAudioContext?: typeof AudioContext
      }).webkitAudioContext
      if (!Ctor) return
      ctxRef.current = new Ctor()
    }
    if (ctxRef.current.state === 'suspended') void ctxRef.current.resume()
  }, [])

  const tone = useCallback((freq: number, start: number, dur: number, gain: number) => {
    const ctx = ctxRef.current
    if (!ctx) return
    const osc = ctx.createOscillator()
    const vol = ctx.createGain()
    osc.type = 'triangle'
    osc.frequency.value = freq
    // 淡入淡出避免爆音
    vol.gain.setValueAtTime(0, ctx.currentTime + start)
    vol.gain.linearRampToValueAtTime(gain, ctx.currentTime + start + 0.02)
    vol.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + start + dur)
    osc.connect(vol).connect(ctx.destination)
    osc.start(ctx.currentTime + start)
    osc.stop(ctx.currentTime + start + dur + 0.02)
  }, [])

  /** 上扬三音，像"答对啦" */
  const playRight = useCallback(() => {
    unlock()
    tone(660, 0, 0.12, 0.16)
    tone(880, 0.1, 0.14, 0.16)
    tone(1180, 0.21, 0.18, 0.14)
  }, [tone, unlock])

  /** 柔和的低音，是"再想想"而不是"你错了" */
  const playWrong = useCallback(() => {
    unlock()
    tone(300, 0, 0.16, 0.12)
    tone(232, 0.13, 0.22, 0.1)
  }, [tone, unlock])

  /** 结算页的小欢呼 */
  const playCheer = useCallback(() => {
    unlock()
    const notes = [523, 659, 784, 1047]
    notes.forEach((f, i) => tone(f, i * 0.11, 0.24, 0.14))
  }, [tone, unlock])

  return { unlock, playRight, playWrong, playCheer }
}
