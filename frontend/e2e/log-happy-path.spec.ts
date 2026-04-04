import { expect, test } from '@playwright/test'

// E2E 테스트는 매 실행마다 새 DB를 사용하므로 beforeAll에서 사용자를 1회 등록한다.
// beforeEach에서 매번 재가입을 시도하면 두 번째 테스트부터 이메일 중복 오류가 발생한다.
const E2E_EMAIL = 'e2e@example.com'
const E2E_PASSWORD = 'e2epassword123'
const E2E_USERNAME = 'e2euser'

test.beforeAll(async ({ browser }) => {
  const page = await browser.newPage()
  await page.goto('/register')
  await page.getByLabel('Email').fill(E2E_EMAIL)
  await page.getByLabel('Password').fill(E2E_PASSWORD)
  await page.getByLabel('Username').fill(E2E_USERNAME)
  await page.getByRole('button', { name: '회원가입' }).click()
  await expect(page).toHaveURL('/')
  await page.close()
})

test.beforeEach(async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Email').fill(E2E_EMAIL)
  await page.getByLabel('Password').fill(E2E_PASSWORD)
  await page.getByRole('button', { name: '로그인' }).click()
  await expect(page).toHaveURL('/')
})

test('brew happy-path CRUD flow works end-to-end', async ({ page }) => {
  const beanName = `E2E Brew Bean ${Date.now()}`
  const updatedBeanName = `${beanName} Updated`

  await page.getByRole('link', { name: '기록 추가' }).click()
  await expect(page).toHaveURL('/logs/new')

  await page.getByRole('button', { name: /Brew log/ }).click()
  await page.getByLabel('Recorded at').fill('2026-03-29T09:30')
  await page.getByLabel('Bean name').fill(beanName)
  await page.getByRole('button', { name: 'AeroPress 압력 + 침지', exact: true }).click()

  // 선택 영역을 펼쳐야 Companions, Memo 등에 접근할 수 있다
  await page.getByRole('button', { name: '더 기록하기' }).click()
  await page.getByLabel('Companions').fill('민수, 지연')
  await page.getByLabel('Memo').fill('E2E로 생성한 브루 로그입니다.')
  await page.getByLabel('Brew device').fill('AeroPress Go')
  await page.getByLabel('Grind size').fill('18 clicks')
  await page.getByLabel(/Coffee \(g\)/).fill('18')
  await page.getByLabel(/Water \(ml\)/).fill('270')
  await page.getByLabel(/Water temperature \(°C\)/).fill('92')
  await page.getByLabel(/Brew time \(sec\)/).fill('110')
  await page.getByLabel('Tasting tags').fill('berry, cacao')
  await page.getByLabel('Tasting note').fill('붉은 과일과 카카오가 이어집니다.')
  await page.getByLabel('Impressions').fill('단맛이 잘 살아난 컵.')
  await page.getByLabel('Brew step 1').fill('30초 뜸들이기 후 천천히 프레스')
  await page.getByRole('button', { name: '기록 추가' }).click()

  await expect(page).toHaveURL(/\/logs\/.+$/)
  await expect(page.getByRole('heading', { name: beanName })).toBeVisible()
  await expect(page.getByText('AeroPress Go')).toBeVisible()
  await expect(page.getByText('30초 뜸들이기 후 천천히 프레스')).toBeVisible()

  await page.getByRole('link', { name: '목록으로' }).click()
  await expect(page).toHaveURL('/')

  await page.getByRole('button', { name: '브루' }).click()
  await expect(page).toHaveURL(/log_type=brew/)
  await expect(page.getByRole('link', { name: new RegExp(beanName) })).toBeVisible()

  await page.getByRole('link', { name: new RegExp(beanName) }).click()
  await page.getByRole('link', { name: '수정' }).click()
  await expect(page).toHaveURL(/\/edit$/)

  await page.getByLabel('Bean name').fill(updatedBeanName)
  await page.getByLabel('Brew device').fill('AeroPress Clear')
  await page.getByLabel('Impressions').fill('수정 후에도 단맛과 질감이 안정적입니다.')
  await page.getByRole('button', { name: '변경 저장' }).click()

  await expect(page).toHaveURL(/\/logs\/.+$/)
  await expect(page.getByRole('heading', { name: updatedBeanName })).toBeVisible()
  await expect(page.getByText('AeroPress Clear')).toBeVisible()
  await expect(page.getByText('수정 후에도 단맛과 질감이 안정적입니다.')).toBeVisible()

  page.once('dialog', (dialog) => dialog.accept())
  await page.getByRole('button', { name: '삭제' }).click()

  await expect(page).toHaveURL('/')
  await expect(page.getByRole('link', { name: new RegExp(updatedBeanName) })).toHaveCount(0)
})

test('clone from detail page creates a new log with pre-filled data', async ({ page }) => {
  const originalBean = `E2E Clone Source ${Date.now()}`

  // 1. 원본 로그 생성
  await page.getByRole('link', { name: '기록 추가' }).click()
  await page.getByRole('button', { name: /Brew log/ }).click()
  await page.getByLabel(/Recorded at/).fill('2026-04-01T10:00')
  await page.getByLabel('Bean name').fill(originalBean)
  await page.getByRole('button', { name: 'AeroPress 압력 + 침지', exact: true }).click()
  await page.getByRole('button', { name: '기록 추가' }).click()
  await expect(page).toHaveURL(/\/logs\/.+$/)
  await expect(page.getByRole('heading', { name: originalBean })).toBeVisible()

  // 2. 상세 화면에서 "다시 쓰기" 클릭
  await page.getByRole('button', { name: '복제' }).click()
  await expect(page).toHaveURL('/logs/new')

  // 3. 복제 폼 검증 — 원본 필드 유지, 리셋 필드 초기화
  await expect(page.getByText('기록 복제')).toBeVisible()
  const beanInput = page.getByLabel('Bean name')
  await expect(beanInput).toHaveValue(originalBean)
  // recordedAt은 원본(2026-04-01)이 아닌 오늘 날짜여야 한다
  const recordedAtValue = await page.getByLabel(/Recorded at/).inputValue()
  expect(recordedAtValue).not.toContain('2026-04-01')

  // 4. 저장하여 새 로그 생성
  await page.getByRole('button', { name: '기록 추가' }).click()
  await expect(page).toHaveURL(/\/logs\/.+$/)
  await expect(page.getByRole('heading', { name: originalBean })).toBeVisible()

  // 5. 목록에서 원본과 복제본 모두 존재하는지 확인
  await page.getByRole('link', { name: '목록으로' }).click()
  await page.getByRole('button', { name: '브루' }).click()
  const cards = page.getByRole('link', { name: new RegExp(originalBean) })
  await expect(cards).toHaveCount(2)

  // 정리: 두 로그 모두 삭제
  for (let i = 0; i < 2; i++) {
    await cards.first().click()
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: '삭제' }).click()
    await expect(page).toHaveURL('/')
    if (i === 0) await page.getByRole('button', { name: '브루' }).click()
  }
})

test('logout redirects to login page', async ({ page }) => {
  await page.getByRole('button', { name: '로그아웃' }).click()
  await expect(page).toHaveURL('/login')

  // 로그아웃 후 홈에 접근하면 로그인 페이지로 리다이렉트된다
  await page.goto('/')
  await expect(page).toHaveURL('/login')
})
