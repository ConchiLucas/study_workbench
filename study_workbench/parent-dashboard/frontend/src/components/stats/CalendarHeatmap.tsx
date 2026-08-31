import ReactECharts from 'echarts-for-react'
import { useCalendar } from '../../api/dashboard'
import { useChildStore } from '../../store/childStore'

export function CalendarHeatmap({ months = 3 }: { months?: number }) {
  const childId = useChildStore((s) => s.childId)
  const { data } = useCalendar(childId, months)
  if (!data) return null

  const end = new Date()
  const start = new Date(end)
  start.setMonth(start.getMonth() - months)
  const fmt = (d: Date) => d.toISOString().slice(0, 10)

  const option = {
    tooltip: {
      formatter: (p: { value: [string, number] }) => `${p.value[0]}<br/>练习 ${p.value[1]} 分钟`,
    },
    visualMap: {
      min: 0, max: 30, type: 'piecewise', orient: 'horizontal', left: 'center', bottom: 0,
      pieces: [
        { min: 1, max: 5, label: '1-5分', color: '#FFE4EE' },
        { min: 6, max: 15, label: '6-15分', color: '#FBA9C5' },
        { min: 16, max: 30, label: '16-30分', color: '#F472A6' },
        { min: 31, label: '30分+', color: '#DB2777' },
      ],
      textStyle: { fontSize: 10 },
    },
    calendar: {
      top: 30, left: 40, right: 20, cellSize: ['auto', 16], range: [fmt(start), fmt(end)],
      splitLine: { show: false }, itemStyle: { borderWidth: 3, borderColor: '#fff', color: '#F5F5F7' },
      yearLabel: { show: false }, dayLabel: { nameMap: 'ZH', fontSize: 10 },
      monthLabel: { nameMap: 'ZH', fontSize: 10 },
    },
    series: {
      type: 'heatmap', coordinateSystem: 'calendar',
      data: data.map((d) => [d.date, d.practice_min]),
    },
  }

  return (
    <div className="rounded-xl2 bg-white p-4 shadow-sm">
      <div className="mb-2 font-semibold text-slate-700">学习日历</div>
      <ReactECharts option={option} style={{ height: 220 }} />
    </div>
  )
}
