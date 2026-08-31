import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { KpDetailDrawer } from '../mastery/KpDetailDrawer'

export function Shell() {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex-1 min-w-0">
        <TopBar />
        <main className="p-6 space-y-6"><Outlet /></main>
      </div>
      <KpDetailDrawer />
    </div>
  )
}
