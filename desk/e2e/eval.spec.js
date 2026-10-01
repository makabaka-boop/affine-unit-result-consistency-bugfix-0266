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

// Editing re-evaluates on its own; the old result disappears the moment the
// input diverges from what is displayed.
test('edits re-evaluate automatically and clear the old result immediately', async ({ page }) => {
  await page.goto('/')
  // the initial expression is evaluated on load
  await expect(page.getByTestId('base-value')).toContainText('23/10 m')

  // editing clears the old result right away…
  await page.fill('#expr-input', '1 kg + 500 g')
  await expect(page.getByTestId('result-panel')).toHaveCount(0)

  // …and a fresh result appears without any button click
  await expect(page.getByTestId('result-panel')).toBeVisible()
  await expect(page.getByTestId('base-value')).toContainText('3/2 kg')

  // changing the target unit re-evaluates too
  await page.selectOption('#target-select', 'g')
  await expect(page.getByTestId('target-value')).toHaveText('1500 g')
})

// A slower earlier request must never overwrite the state of a newer edit.
test('a slower earlier response cannot overwrite a newer result', async ({ page }) => {
  await page.route('**/api/eval', async route => {
    const expr = route.request().postDataJSON().expression
    if (expr.includes('20 C')) {
      // hold the temperature request back so it resolves out of order
      await new Promise(r => setTimeout(r, 1200))
    }
    try {
      await route.fallback()
    } catch {
      // the client may have aborted the superseded request
    }
  })
  await page.goto('/')
  await expect(page.getByTestId('base-value')).toContainText('23/10 m')

  // trigger a slow evaluation…
  await page.fill('#expr-input', '20 C + 5 dC in F')
  await page.waitForRequest(
    req => req.url().includes('/api/eval') && (req.postData() || '').includes('20 C'),
  )

  // …then edit again before the slow response comes back
  await page.fill('#expr-input', '2 m * 3 m')
  await expect(page.getByTestId('base-value')).toContainText('6 m²')

  // even after the stale response arrives, the newer result must stay
  await page.waitForTimeout(1800)
  await expect(page.getByTestId('base-value')).toContainText('6 m²')
  await expect(page.getByTestId('error-panel')).toHaveCount(0)
})
