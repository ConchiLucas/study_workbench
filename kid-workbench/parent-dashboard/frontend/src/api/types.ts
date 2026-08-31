import type { MasteryStatus } from '../theme'

export interface StatusCounts {
  not_started: number; learning: number; shaky: number;
  mastered: number; review_due: number;
}

export interface Overview {
  child: { id: number; name: string; grade: string; flowers: number }
  total_kp: number
  counts: StatusCounts
  week_delta: { mastered: number; practice_min: number }
  streak_days: number
  today: { practice_min: number; attempts: number; tasks_done: number; tasks_total: number }
}

export interface SubjectSummary {
  code: string; name: string; icon: string; total: number;
  counts: StatusCounts; week_new: number; progress: number;
}

export interface MatrixSkill {
  code: string
  status: MasteryStatus
  accuracy: number
  attempts: number
}

export interface MatrixPoint {
  id: number; title: string; status: MasteryStatus;
  accuracy: number; attempts: number; due_at: string | null;
  skills?: MatrixSkill[];
}

export interface MatrixModule {
  code: string; name: string; total: number; mastered: number; points: MatrixPoint[];
}

export interface Matrix { subject: SubjectSummary; modules: MatrixModule[] }

export interface AttentionItem {
  kp_id: number; title: string; subject_code: string; subject_name: string;
  module_name: string; status: MasteryStatus; accuracy: number;
  wrong_count: number; attempts: number; due_at: string | null;
}

export interface HistoryItem {
  at: string; is_correct: boolean; cost_ms: number; source: string;
  skill_code?: string;
}

export interface KpDetail {
  kp_id: number; title: string; payload: string; difficulty: number;
  subject_code: string; subject_name: string; module_name: string;
  status: MasteryStatus; attempts: number; correct: number; accuracy: number;
  streak: number; best_streak: number;
  due_at: string | null; mastered_at: string | null; history: HistoryItem[];
  skills?: MatrixSkill[];
}

export interface TrendPoint {
  date: string; practice_min: number; attempts: number;
  correct: number; newly_mastered: number; cumulative_mastered: number;
}

export interface CalendarDay {
  date: string; practice_min: number; attempts: number;
  mastered: number; checked_in: boolean;
}
