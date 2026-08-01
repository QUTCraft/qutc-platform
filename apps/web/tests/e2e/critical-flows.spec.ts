import { expect, test } from '@playwright/test'

test('public portal routes remain navigable', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toContainText('社团')
  for (const path of ['/posts', '/projects', '/resources', '/knowledge', '/apply']) {
    await page.goto(path)
    await expect(page.locator('body')).not.toContainText('页面不存在')
  }
})

test('owner can open dashboard and organization settings', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)
  await expect(page.getByRole('heading', { name: '工作台概览' })).toBeVisible()

  await page.goto('/admin/settings')
  await expect(page.getByRole('heading', { name: '组织公开资料' })).toBeVisible()
  await expect(page.getByLabel('组织全称')).toHaveValue('QUTCraft Commons')
})

test('content editor keeps the full-page markdown workspace scrollable', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
	await expect(page).toHaveURL(/\/admin$/)
  await page.goto('/admin/content/new')
  await expect(page.locator('textarea').first()).toBeVisible()
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(0)
})

test('activity planner opens as a structured three-step workspace', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)
  await page.goto('/admin/activity-planner')

  await expect(page.getByRole('heading', { name: 'AI 校园活动策划' })).toBeVisible()
  await expect(page.getByText('活动需求', { exact: true })).toBeVisible()
  await expect(page.getByText('选择依据', { exact: true })).toBeVisible()
  await expect(page.getByText('审查执行', { exact: true })).toBeVisible()
  await expect(page.getByLabel('活动名称')).toBeVisible()
})
