import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

/** 旧书签 /quiz 兜底：回首页走「开始」手势，才能解锁 iOS 语音。 */
export function KidQuizRedirect() {
  const nav = useNavigate()

  useEffect(() => {
    nav('/', { replace: true })
  }, [nav])

  return (
    <div className="flex h-full items-center justify-center">
      <span className="animate-bob text-7xl">🌸</span>
    </div>
  )
}
