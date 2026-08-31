import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { KidDone } from './pages/KidDone'
import { KidHome } from './pages/KidHome'
import { KidQuiz } from './pages/KidQuiz'
import { KidQuizRedirect } from './pages/KidQuizRedirect'

const qc = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000 } } })

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        <Routes>
          <Route element={<Shell />}>
            <Route index element={<KidHome />} />
            <Route path="task/:planId" element={<KidQuiz />} />
            <Route path="task/:planId/done" element={<KidDone />} />
            <Route path="quiz" element={<KidQuizRedirect />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
