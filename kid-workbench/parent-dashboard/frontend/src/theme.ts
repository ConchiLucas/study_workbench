export type MasteryStatus =
  | 'not_started' | 'learning' | 'shaky' | 'mastered' | 'review_due';

export const STATUS_STYLE: Record<MasteryStatus, {
  label: string; bg: string; text: string; ring: string; desc: string;
}> = {
  mastered:    { label: '已掌握', bg: '#86EFAC', text: '#166534', ring: '#22C55E', desc: '连续答对，稳了' },
  review_due:  { label: '待复习', bg: '#93C5FD', text: '#1E40AF', ring: '#3B82F6', desc: '曾掌握，该复习了' },
  learning:    { label: '学习中', bg: '#FDE68A', text: '#92400E', ring: '#F59E0B', desc: '练过但还不稳' },
  shaky:       { label: '需巩固', bg: '#FCA5A5', text: '#991B1B', ring: '#EF4444', desc: '反复出错，重点关注' },
  not_started: { label: '未开始', bg: '#EEEEF2', text: '#9CA3AF', ring: '#D1D5DB', desc: '还没练过' },
}

export const STATUS_ORDER: MasteryStatus[] =
  ['mastered', 'review_due', 'learning', 'shaky', 'not_started']
