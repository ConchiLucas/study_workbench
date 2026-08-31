import type { PropsWithChildren } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import {
  Bot,
  Braces,
  Cloud,
  Database,
  Image,
  Mic,
  RadioTower,
  Terminal,
  Video,
} from 'lucide-react'

const configNav = [
  { to: '/config/databases', label: '数据库配置', icon: Database },
  { to: '/config/ai', label: 'AI 配置', icon: Bot },
  { to: '/config/local-cli', label: '本地 CLI 配置', icon: Terminal },
  { to: '/config/object-storage', label: 'MinIO 配置', icon: Cloud },
  { to: '/config/image-models', label: '图片模型配置', icon: Image },
  { to: '/config/video-models', label: '视频模型配置', icon: Video },
  { to: '/config/voice-models', label: 'TTS 语音配置', icon: Mic },
  { to: '/config/runtime', label: 'Runtime Contract', icon: Braces },
]

export function AppShell({ children }: PropsWithChildren) {
  const { pathname } = useLocation()
  const onConfig = pathname.startsWith('/config')

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand-mark" aria-hidden="true">
          <RadioTower size={18} />
        </div>
        <div className="brand-block">
          <div className="brand-name">Study Content Admin</div>
          <div className="brand-sub">学习内容后台</div>
        </div>
        <span className="environment-label">INTERNAL</span>

        <nav className="top-primary-nav" aria-label="主导航">
          <NavLink
            to="/literacy"
            className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}
          >
            识字
          </NavLink>
          <NavLink
            to="/pinyin"
            className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}
          >
            拼音
          </NavLink>
          <NavLink
            to="/math"
            className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}
          >
            算术
          </NavLink>
          <NavLink
            to="/english"
            className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}
          >
            英语
          </NavLink>
          <NavLink
            to="/science"
            className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}
          >
            科普
          </NavLink>
          <NavLink
            to="/questions"
            className={({ isActive }) => `top-nav-link${isActive ? ' active' : ''}`}
          >
            题目管理
          </NavLink>
          <NavLink
            to="/config/databases"
            className={() => `top-nav-link${onConfig ? ' active' : ''}`}
          >
            配置管理
          </NavLink>
        </nav>
      </header>

      <div className={`workspace${onConfig ? ' has-rail' : ' full'}`}>
        {onConfig ? (
          <aside className="navigation-rail">
            <div className="navigation-heading">配置管理</div>
            <nav aria-label="配置导航">
              {configNav.map(({ to, label, icon: Icon }) => (
                <NavLink
                  key={to}
                  to={to}
                  className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                >
                  <Icon size={18} />
                  <span>{label}</span>
                </NavLink>
              ))}
            </nav>
            <div className="plaintext-boundary">
              <strong>PLAINTEXT BOUNDARY</strong>
              配置凭据明文展示，仅限可信内网。修改请到 Shared Config Center。
            </div>
          </aside>
        ) : null}
        <main className="application-main">{children}</main>
      </div>
    </div>
  )
}
