/**
 * 커서 기반 페이지네이션 응답의 제네릭 래퍼.
 * API마다 items 타입이 다를 수 있어 제네릭으로 정의한다.
 * openapi.yml의 ListLogsResponse와 구조적으로 호환된다.
 */
export interface CursorPage<T> {
  items: T[]
  /** 다음 페이지 커서. has_next가 false이면 null */
  next_cursor: string | null
  has_next: boolean
}
