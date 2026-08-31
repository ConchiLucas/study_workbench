import { useCallback, useRef } from 'react'

type Lang = 'zh-CN' | 'en-US'

/**
 * 读题用浏览器自带的 Web Speech API：iPad Safari 内置中英文语音，
 * 不需要后端 TTS，也不产生流量。
 *
 * iOS 有个硬性限制：首次 speak() 必须发生在用户手势的调用栈里，
 * 否则会被静默丢弃。所以封面页的"开始"按钮除了是入口，
 * 还承担了解锁语音的职责（见 prime）。
 */
export function useSpeech() {
  const primed = useRef(false)

  const speak = useCallback((text: string, lang: Lang = 'zh-CN') => {
    if (!text || typeof window === 'undefined' || !window.speechSynthesis) return
    // 打断上一句，否则连点会把两句叠在一起念。
    window.speechSynthesis.cancel()
    const u = new SpeechSynthesisUtterance(text)
    u.lang = lang
    u.rate = 0.85 // 放慢，5 岁孩子才听得清
    u.pitch = 1.05
    window.speechSynthesis.speak(u)
  }, [])

  /** 中英分两段念，中间留停顿——混在一句里 TTS 会把英文读成中文腔。 */
  const speakPair = useCallback((zh: string, en: string) => {
    if (typeof window === 'undefined' || !window.speechSynthesis) return
    window.speechSynthesis.cancel()
    const first = new SpeechSynthesisUtterance(zh)
    first.lang = 'zh-CN'
    first.rate = 0.85
    const second = new SpeechSynthesisUtterance(en)
    second.lang = 'en-US'
    second.rate = 0.8
    first.onend = () => window.setTimeout(() => window.speechSynthesis.speak(second), 220)
    window.speechSynthesis.speak(first)
  }, [])

  /** 在用户手势里调用一次，解锁 iOS 的语音权限。 */
  const prime = useCallback(() => {
    if (primed.current || typeof window === 'undefined' || !window.speechSynthesis) return
    primed.current = true
    const u = new SpeechSynthesisUtterance('')
    u.volume = 0
    window.speechSynthesis.speak(u)
  }, [])

  const stop = useCallback(() => {
    window.speechSynthesis?.cancel()
  }, [])

  return { speak, speakPair, prime, stop }
}
