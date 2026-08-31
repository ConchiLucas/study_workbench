import { Outlet } from 'react-router-dom'

export function Shell() {
  return (
    <div className="kid-root kid-paper font-kid text-candy-ink">
      <Outlet />
    </div>
  )
}
