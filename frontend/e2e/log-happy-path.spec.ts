import { expect, test } from '@playwright/test'

test('brew happy-path CRUD flow works end-to-end', async ({ page }) => {
  const beanName = `E2E Brew Bean ${Date.now()}`
  const updatedBeanName = `${beanName} Updated`

  await page.goto('/')

  await page.getByRole('link', { name: '오늘의 기록 추가' }).click()
  await expect(page).toHaveURL('/logs/new')

  await page.getByRole('button', { name: /Brew log/ }).click()
  await page.getByLabel('Recorded at').fill('2026-03-29T09:30')
  await page.getByLabel('Companions').fill('민수, 지연')
  await page.getByLabel('Memo').fill('E2E로 생성한 브루 로그입니다.')
  await page.getByLabel('Bean name').fill(beanName)
  await page.getByRole('button', { name: 'AeroPress 압력 + 침지', exact: true }).click()
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
