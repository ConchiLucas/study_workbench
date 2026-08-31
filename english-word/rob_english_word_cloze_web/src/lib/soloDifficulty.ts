import type { ClozePracticePreference } from "../types/cloze";

export type SoloDifficultyParentKey =
  | "primary"
  | "junior"
  | "senior"
  | "college"
  | "entrance"
  | "business_abroad"
  | "professional"
  | "advanced_exam";

export interface SoloDifficultyChildOption {
  key: string;
  title: string;
  avgDifficulty: number;
}

export interface SoloDifficultyParentOption {
  key: SoloDifficultyParentKey;
  title: string;
  children: SoloDifficultyChildOption[];
}

export interface SelectedSoloDifficulty {
  key: string;
  parentKey: SoloDifficultyParentKey;
  title: string;
  avgDifficulty?: number;
}

export const SOLO_DIFFICULTY_OPTIONS: SoloDifficultyParentOption[] = [
  { key: "primary", title: "小学英语", children: [
    { key: "primary_3_1", title: "3年级上册", avgDifficulty: 71 },
    { key: "primary_3_2", title: "3年级下册", avgDifficulty: 72 },
    { key: "primary_4_1", title: "4年级上册", avgDifficulty: 73 },
    { key: "primary_4_2", title: "4年级下册", avgDifficulty: 74 },
    { key: "primary_5_1", title: "5年级上册", avgDifficulty: 79 },
    { key: "primary_5_2", title: "5年级下册", avgDifficulty: 78 },
    { key: "primary_6_1", title: "6年级上册", avgDifficulty: 82 },
    { key: "primary_6_2", title: "6年级下册", avgDifficulty: 81 },
  ] },
  { key: "junior", title: "初中英语", children: [
    { key: "junior_7_1", title: "7年级上册", avgDifficulty: 122 },
    { key: "junior_7_2", title: "7年级下册", avgDifficulty: 130 },
    { key: "junior_8_1", title: "8年级上册", avgDifficulty: 137 },
    { key: "junior_8_2", title: "8年级下册", avgDifficulty: 140 },
    { key: "junior_9_1", title: "9年级全册", avgDifficulty: 147 },
  ] },
  { key: "senior", title: "高中英语", children: Array.from({ length: 11 }, (_, index) => ({
    key: `senior_${index + 1}`, title: `高中第${index + 1}册`, avgDifficulty: 246 + index * 5,
  })) },
  { key: "college", title: "大学英语", children: [
    { key: "college_cet4", title: "四级", avgDifficulty: 340 },
    { key: "college_cet6", title: "六级", avgDifficulty: 450 },
  ] },
  { key: "entrance", title: "升学考试英语", children: [
    { key: "entrance_kaoyan", title: "考研英语", avgDifficulty: 442 },
  ] },
  { key: "business_abroad", title: "商务与出国英语", children: [
    { key: "business_bec", title: "商务英语 BEC", avgDifficulty: 573 },
    { key: "business_ielts", title: "雅思", avgDifficulty: 573 },
    { key: "business_toefl", title: "托福", avgDifficulty: 640 },
    { key: "business_gmat", title: "GMAT", avgDifficulty: 693 },
  ] },
  { key: "professional", title: "专业英语", children: [
    { key: "professional_tem4", title: "专四", avgDifficulty: 501 },
    { key: "professional_tem8", title: "专八", avgDifficulty: 541 },
  ] },
  { key: "advanced_exam", title: "高阶考试英语", children: [
    { key: "advanced_gre", title: "GRE", avgDifficulty: 732 },
    { key: "advanced_sat", title: "SAT", avgDifficulty: 747 },
  ] },
];

export const DEFAULT_SOLO_DIFFICULTY: SelectedSoloDifficulty = {
  key: "junior",
  parentKey: "junior",
  title: "初中英语",
};

export function normalizeSoloDifficulty(preference?: Partial<ClozePracticePreference> | null): SelectedSoloDifficulty {
  const parent = SOLO_DIFFICULTY_OPTIONS.find((option) => option.key === preference?.soloDifficultyGroup);
  if (!parent) return DEFAULT_SOLO_DIFFICULTY;
  if (preference?.soloDifficultyLevel === parent.key) {
    return { key: parent.key, parentKey: parent.key, title: parent.title };
  }
  const child = parent.children.find((option) => option.key === preference?.soloDifficultyLevel);
  return child
    ? { key: child.key, parentKey: parent.key, title: child.title, avgDifficulty: child.avgDifficulty }
    : DEFAULT_SOLO_DIFFICULTY;
}

export function selectedSoloDifficultyText(selected: SelectedSoloDifficulty): string {
  const parent = SOLO_DIFFICULTY_OPTIONS.find((option) => option.key === selected.parentKey);
  return selected.key === selected.parentKey
    ? selected.title
    : `${parent?.title || ""}${parent ? " · " : ""}${selected.title}`;
}
