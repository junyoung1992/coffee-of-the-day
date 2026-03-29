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

/**
 * API 계층이 던지는 에러 객체의 최소 형태.
 * UI는 이 형태만 알면 상태 코드와 메시지를 공통 처리할 수 있다.
 */
export interface ApiErrorLike {
  message: string
  status?: number
  code?: string
}
