/** 商家端通用格式化工具 */

/** 分 -> 元（保留 2 位小数字符串） */
export const fenToYuan = (value: any): string =>
  (Number(value || 0) / 100).toFixed(2)

/** 分 -> 元（数字） */
export const fenToYuanNumber = (value: any): number =>
  Number((Number(value || 0) / 100).toFixed(2))

/** 毫秒时间戳 -> MM-DD HH:mm */
export const formatDateTime = (ms: any): string => {
  if (!ms) return '-'
  const d = new Date(Number(ms))
  const pad = (n: number) => (n < 10 ? '0' + n : String(n))
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 毫秒时间戳 -> YYYY-MM-DD */
export const formatDate = (ms: any): string => {
  if (!ms) return '-'
  const d = new Date(Number(ms))
  const pad = (n: number) => (n < 10 ? '0' + n : String(n))
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 剩余时间文案 */
export const formatRemain = (endTime: any): string => {
  const left = Number(endTime) - Date.now()
  if (left <= 0) return '已结束'
  const h = Math.floor(left / 3600000)
  const m = Math.floor((left % 3600000) / 60000)
  return h > 0 ? `${h}小时${m}分` : `${m}分钟`
}

/** 安全 JSON 解析 */
export const parseJson = <T = any>(raw: any, fallback: T): T => {
  if (!raw) return fallback
  try {
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

/** 浏览器下载 CSV（带 BOM，Excel 中文不乱码） */
export const downloadCsv = (filename: string, rows: (string | number)[][]) => {
  const content = '\ufeff' + rows.map((row) => row.map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`).join(',')).join('\n')
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
