# Postmortem — CI Backend Test JWT 만료 실패

**일시**: 2026-04-12
**작업**: Issue #8 PR CI 검증
**심각도**: 낮음 (프로덕션 영향 없음, CI 테스트 환경)

---

## 요약

Issue #8 PR의 CI에서 `TestAuthService_Refresh_Success`와 `TestAuthService_Logout_IncrementsTokenVersion`이 실패했다. 백엔드 코드를 변경하지 않았음에도 발생한 시간 의존 테스트 결함이다.

---

## 원인

- `newTestAuthService`는 `svc.now`를 `2026-03-30`으로 고정한다.
- refresh token의 만료 기간은 7일이므로, 발급된 토큰의 만료일은 `2026-04-06`이다.
- `parseTokenClaims`에서 `jwt.ParseWithClaims`는 기본적으로 `time.Now()`로 만료를 검증한다.
- 실제 날짜가 `2026-04-12`이므로 토큰이 이미 만료되어 파싱이 실패했다.
- 토큰 발급 시에는 고정 시각(`s.now`)을 사용하지만, 검증 시에는 실제 시각(`time.Now()`)을 사용하는 불일치가 근본 원인이다.

---

## 해결

`parseTokenClaims`의 `jwt.ParseWithClaims` 호출에 `jwt.WithTimeFunc(s.now)` 옵션을 추가하여, 토큰 발급과 검증 양쪽에서 동일한 시간 소스를 사용하도록 수정한다.

- 프로덕션: `s.now`가 `time.Now`이므로 동작 변화 없음
- 테스트: 고정 시각 기준으로 만료 검증이 수행되어 시간 경과와 무관하게 통과

---

## 교훈

시간 의존 로직을 테스트할 때는 발급(쓰기)과 검증(읽기) 양쪽 모두에 시간 함수를 주입해야 한다. 한쪽만 고정하면 시간이 지나면서 테스트가 깨진다.
