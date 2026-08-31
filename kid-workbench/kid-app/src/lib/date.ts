/**
 * 本地时区的 YYYY-MM-DD。
 *
 * 不能用 toISOString().slice(0,10)：那是 UTC 日期，东八区晚上 8 点之后
 * 会算成前一天，和后端按本地日期存的 plan_date 对不上。
 */
export function localDate(d: Date = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
