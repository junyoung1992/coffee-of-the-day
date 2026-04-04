// 로컬 시간 기준으로 날짜를 YYYY-MM-DD 형식 문자열로 변환한다.
// toISOString()은 UTC 기준이므로 KST 자정~오전에 어제 날짜를 반환하는 문제가 있다.
export function formatLocalDate(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

// 오늘 기준 30일 전 (로컬 시간 기준)
export function getDefaultDateFrom(): string {
  const now = new Date()
  now.setDate(now.getDate() - 30)
  return formatLocalDate(now)
}

// 오늘 (로컬 시간 기준)
export function getDefaultDateTo(): string {
  return formatLocalDate(new Date())
}
