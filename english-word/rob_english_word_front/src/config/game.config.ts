/**
 * 游戏配置参数
 * 修改此文件即可调整游戏轮次、时长等参数
 */

export const GAME_CONFIG = {
  /**
   * 每轮游戏单词数量（抢词阶段最多抢多少个单词）
   * 默认: 8个单词
   * 总时长计算公式: WORDS_PER_ROUND * QUESTION_TIME_PER_WORD
   * 例如: 8个单词 × 4秒 = 32秒答题时间
   */
  WORDS_PER_ROUND: 5,

  /**
   * 抢词阶段总时长（秒）
   * 默认: 6秒
   */
  GRAB_PHASE_DURATION: 6,

  /**
   * 每道题的倒计时时间（秒）
   * 默认: 4秒
   * 包含: 3秒答题时间 + 1秒显示正确答案
   */
  QUESTION_TIME_PER_WORD: 4,

  /**
   * 显示正确答案的时长（秒）
   * 默认: 1秒（包含在 QUESTION_TIME_PER_WORD 中）
   */
  SHOW_ANSWER_DURATION: 1,

  /**
   * 获取答题阶段总时长（秒）
   * 计算公式: 单词数量 × 每题时长
   */
  get ANSWER_PHASE_TOTAL_TIME(): number {
    return this.WORDS_PER_ROUND * this.QUESTION_TIME_PER_WORD
  },

  /**
   * 获取实际答题时间（不含显示答案时间）
   * 计算公式: 单词数量 × (每题时长 - 显示答案时长)
   */
  get ACTUAL_ANSWER_TIME(): number {
    return this.WORDS_PER_ROUND * (this.QUESTION_TIME_PER_WORD - this.SHOW_ANSWER_DURATION)
  }
}

/**
 * 便捷导出单个配置项
 */
export const {
  WORDS_PER_ROUND,
  GRAB_PHASE_DURATION,
  QUESTION_TIME_PER_WORD,
  SHOW_ANSWER_DURATION,
  ANSWER_PHASE_TOTAL_TIME,
  ACTUAL_ANSWER_TIME
} = GAME_CONFIG

export default GAME_CONFIG