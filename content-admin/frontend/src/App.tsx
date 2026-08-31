import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './layout/AppShell'
import { ConfigPage } from './features/config/ConfigPage'
import { EnglishPage } from './features/english/EnglishPage'
import { LiteracyPage } from './features/literacy/LiteracyPage'
import { MathPage } from './features/math/MathPage'
import { PinyinPage } from './features/pinyin/PinyinPage'
import { QuestionsPage } from './features/qtask/QuestionsPage'
import { SciencePage } from './features/science/SciencePage'

export function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<Navigate to="/literacy" replace />} />
        <Route path="/literacy" element={<LiteracyPage />} />
        <Route path="/pinyin" element={<PinyinPage />} />
        <Route path="/math" element={<MathPage />} />
        <Route path="/english" element={<EnglishPage />} />
        <Route path="/science" element={<SciencePage />} />
        <Route path="/questions" element={<QuestionsPage />} />
        <Route path="/config/:section" element={<ConfigPage />} />
        <Route path="*" element={<Navigate to="/literacy" replace />} />
      </Routes>
    </AppShell>
  )
}
