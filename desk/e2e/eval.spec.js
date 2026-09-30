import { test, expect } from '@playwright/test'

// The core regression: an illegal operation must never leave a previously
// computed result on screen.
test('illegal operation clears the old result', async ({ page }) => {
  await page.goto('/')

  // 1. A valid computation first; its result and derivation steps appear.
  await page.fill('#expr-input', '2 m + 30 cm')
  await page.click('#eval-btn')
  await expect(page.getByTestId('result-panel')).toBeVisible()
  await expect(page.getByTestId('base-value')).toContainText('23/10 m')
  const stepsBefore = page.locator('.step')
  await expect(stepsBefore).toHaveCount(3)

  // 2. Edit into an illegal expression: absolute temperature multiplied.
  await page.fill('#expr-input', '20 C * 2')
  await page.click('#eval-btn')

  // 3. The old result must be gone; an error with a precise span shows instead.
  await expect(page.getByTestId('result-panel')).toHaveCount(0)
  await expect(page.getByTestId('error-panel')).toBeVisible()
  await expect(page.getByTestId('error-msg')).toContainText('absolute temperatures cannot be multiplied')
  await expect(page.locator('.hl-mark')).toHaveText('20 C * 2')
  await expect(page.locator('.step')).toHaveCount(0)

  // 4. Going back to a legal expression shows a fresh result — no merged state.
  await page.fill('#expr-input', '20 C + 5 dC in F')
  await page.click('#eval-btn')
  await expect(page.getByTestId('error-panel')).toHaveCount(0)
  await expect(page.getByTestId('result-panel')).toBeVisible()
  await expect(page.getByTestId('target-value')).toHaveText('77 F')

  // 5. Another illegal one (absolute + absolute) must clear again.
  await page.fill('#expr-input', '20 C + 30 C')
  await page.click('#eval-btn')
  await expect(page.getByTestId('result-panel')).toHaveCount(0)
  await expect(page.getByTestId('error-msg')).toContainText('cannot be added together')
})

test('step-by-step derivation shows type and dimension for every node', async ({ page }) => {
  await page.goto('/')
  await page.fill('#expr-input', '100 C in F')
  await page.click('#eval-btn')
  await expect(page.getByTestId('result-panel')).toBeVisible()
  await expect(page.getByTestId('target-value')).toHaveText('212 F')

  const steps = page.locator('.step')
  await expect(steps.nth(0)).toContainText('绝对')
  await expect(steps.nth(0)).toContainText('Θ')
  await expect(steps.nth(0)).toContainText('100 C')
  // the convert step mentions the target
  await expect(steps.last()).toContainText('换算')
})

test('dimension mismatch is rejected', async ({ page }) => {
  await page.goto('/')
  await page.fill('#expr-input', '1 m + 1 kg')
  await page.click('#eval-btn')
  await expect(page.getByTestId('error-panel')).toBeVisible()
  await expect(page.getByTestId('error-msg')).toContainText('dimensions')
  await expect(page.locator('.hl-mark')).toHaveText('1 m + 1 kg')
})
