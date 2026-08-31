import { Suspense, lazy, type ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Shell } from './components/layout/Shell'

const Overview = lazy(() => import('./pages/Overview').then((m) => ({ default: m.Overview })))
const SubjectDetail = lazy(() => import('./pages/SubjectDetail').then((m) => ({ default: m.SubjectDetail })))
const Attention = lazy(() => import('./pages/Attention').then((m) => ({ default: m.Attention })))
const TasksPage = lazy(() => import('./pages/TasksPage').then((m) => ({ default: m.TasksPage })))
const TaskDetailPage = lazy(() => import('./pages/TaskDetailPage').then((m) => ({ default: m.TaskDetailPage })))
const Rewards = lazy(() => import('./pages/Rewards').then((m) => ({ default: m.Rewards })))

const qc = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000 } } })

function Lazy({ children }: { children: ReactNode }) {
  return <Suspense fallback={<div className="text-sm text-slate-400">加载中…</div>}>{children}</Suspense>
}

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        <Routes>
          <Route element={<Shell />}>
            <Route path="/" element={<Lazy><Overview /></Lazy>} />
            <Route path="/subjects/:code" element={<Lazy><SubjectDetail /></Lazy>} />
            <Route path="/attention" element={<Lazy><Attention /></Lazy>} />
            <Route path="/tasks" element={<Lazy><TasksPage /></Lazy>} />
            <Route path="/tasks/:planId" element={<Lazy><TaskDetailPage /></Lazy>} />
            <Route path="/rewards" element={<Lazy><Rewards /></Lazy>} />
            <Route path="/calendar" element={<Navigate to="/tasks" replace />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
