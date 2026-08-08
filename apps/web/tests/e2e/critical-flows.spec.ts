import { expect, test, type Page } from '@playwright/test'

async function clickAdminNavigation(page: Page, label: string) {
  const menuButton = page.getByRole('button', { name: '打开后台导航' })
  if (await menuButton.isVisible()) await menuButton.click()
  await page.getByRole('link', { name: label, exact: true }).click()
}

async function clickPublicNavigation(page: Page, label: string) {
  const desktopLink = page.locator('.desktop-nav').getByRole('link', { name: label, exact: true })
  if (await desktopLink.isVisible()) {
    await desktopLink.click()
    return
  }
  await page.getByRole('button', { name: '打开导航' }).click()
  await page.locator('.mobile-nav').getByRole('link', { name: label, exact: true }).click()
}

test('stale dynamic imports trigger one controlled reload', async ({ page }) => {
  await page.goto('/')
  const initialTimeOrigin = await page.evaluate(() => performance.timeOrigin)

  const recoveryLoad = page.waitForEvent('load')
  await page.evaluate(() => {
    window.setTimeout(() => window.dispatchEvent(new Event('vite:preloadError', { cancelable: true })), 0)
  })
  await recoveryLoad

  const recoveredTimeOrigin = await page.evaluate(() => performance.timeOrigin)
  expect(recoveredTimeOrigin).not.toBe(initialTimeOrigin)
  await expect.poll(() => page.evaluate(() => Number(sessionStorage.getItem('qutc:stale-chunk-recovery') ?? 0))).toBeGreaterThan(0)
  await page.evaluate(() => window.dispatchEvent(new Event('vite:preloadError', { cancelable: true })))
  await page.waitForTimeout(250)
  expect(await page.evaluate(() => performance.timeOrigin)).toBe(recoveredTimeOrigin)
})

test('public portal routes remain navigable', async ({ page }) => {
  await page.addInitScript(() => {
    sessionStorage.setItem('qutc.portal.runtime_fallback', JSON.stringify({
      portal_id: 'legacy-custom-portal',
      version: '1.0.0',
      reason: 'entry_unavailable',
      occurred_at: new Date().toISOString(),
    }))
  })
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toContainText('社团')
  await expect(page.locator('body')).not.toContainText('自定义门户暂时不可用')
  await expect(page.getByRole('heading', { name: '最新动态与公告', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: '正在发生的项目', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: '共享资源', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: '公共知识库', exact: true })).toBeVisible()
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem('qutc.portal.runtime_fallback'))).toBeNull()
  for (const [label, path, heading] of [
    ['动态', '/posts', '社团动态'],
    ['项目', '/projects', '正在发生的项目'],
    ['资源', '/resources', '共享资源'],
    ['知识库', '/knowledge', '公共知识库'],
  ] as const) {
    await clickPublicNavigation(page, label)
    await expect(page).toHaveURL(new RegExp(`${path}$`))
    await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible()
    await expect(page.locator('body')).not.toContainText(/正在打开(?:管理)?页面/)
    await expect(page.locator('body')).not.toContainText('页面不存在')
  }

  const joinButton = page.getByRole('button', { name: '加入我们' })
  if (await joinButton.isVisible()) {
    await joinButton.click()
  } else {
    await clickPublicNavigation(page, '首页')
    await page.getByRole('button', { name: /申请加入服务器/ }).first().click()
  }
  await expect(page).toHaveURL(/\/apply$/)
  await expect(page.locator('body')).not.toContainText(/正在打开(?:管理)?页面/)
})

test('owner can open dashboard and organization settings', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)
  await expect(page.getByRole('heading', { name: '工作台概览' })).toBeVisible()

  if ((page.viewportSize()?.width ?? 0) > 900) {
    const rail = page.locator('.admin-rail')
    await expect.poll(async () => (await rail.boundingBox())?.width ?? 0).toBeLessThanOrEqual(216)
    await expect.poll(() => page.locator('.admin-nav').evaluate((element) => element.scrollHeight <= element.clientHeight + 1)).toBe(true)
    await expect.poll(() => page.locator('.rail-link').evaluateAll((links) => Math.max(...links.map((link) => link.getBoundingClientRect().height)))).toBeLessThanOrEqual(40)
  }

  await clickAdminNavigation(page, '内容管理')
  await expect(page).toHaveURL(/\/admin\/content$/)
  await expect(page.getByRole('heading', { name: '内容管理', exact: true })).toBeVisible()

  await clickAdminNavigation(page, '概览')
  await expect(page).toHaveURL(/\/admin$/)
  await expect(page.getByRole('heading', { name: '工作台概览' })).toBeVisible()

  await clickAdminNavigation(page, '系统设置')
  await expect(page).toHaveURL(/\/admin\/settings$/)
  await expect(page.getByRole('heading', { name: '组织公开资料' })).toBeVisible()
  await expect(page.getByLabel('组织全称')).toHaveValue('QUTCraft Commons')
  const characterCount = page.locator('.el-input__count').first()
  await expect(characterCount).toBeVisible()
  await expect.poll(() => characterCount.evaluate((element) => getComputedStyle(element).backgroundColor)).not.toBe('rgb(255, 255, 255)')

  await clickAdminNavigation(page, '智能体配置')
  await expect(page).toHaveURL(/\/admin\/ai$/)
  await expect(page.getByRole('heading', { level: 2, name: '智能体配置', exact: true })).toBeVisible()
  const numberControl = page.locator('.el-input-number__increase').first()
  await expect(numberControl).toBeVisible()
  await expect.poll(() => numberControl.evaluate((element) => getComputedStyle(element).backgroundColor)).not.toBe('rgb(255, 255, 255)')
})

test('account can switch organization and keep the selected session context', async ({ page }) => {
  await page.goto('/?organization=campus-commons')
  await page.goto('/login')
  await expect(page).toHaveURL(/\/login$/)
  await expect(page.locator('.login-hero')).toContainText('Campus Commons')
  await expect(page.locator('.login-hero')).not.toContainText('Minecraft')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)

  const organizationSwitcher = page.locator('.organization-switcher')
  await expect(page.getByRole('combobox', { name: '切换当前组织' })).toBeEnabled()
  await expect(organizationSwitcher).toContainText('QUTCraft')
  await organizationSwitcher.click()
  await Promise.all([
    page.waitForEvent('load'),
    page.getByRole('option', { name: /校园协作中心/ }).click(),
  ])

  await expect(page).toHaveURL(/\/admin$/)
  await expect(page.locator('.organization-switcher')).toContainText('校园协作中心')
  await expect(page.locator('.account-meta')).toContainText('管理员')
  await expect(page.locator('.admin-brand')).toContainText('校园协作中心')
  await expect(page.locator('.metric-grid')).toContainText('进行中项目')
  await expect(page.getByRole('heading', { name: 'AI 活动运营' })).toBeVisible()
  await page.reload()
  await expect(page.locator('.organization-switcher')).toContainText('校园协作中心')

  const menuButton = page.getByRole('button', { name: '打开后台导航' })
  if (await menuButton.isVisible()) await menuButton.click()
  await page.getByRole('link', { name: '返回公开门户', exact: true }).click()
  await expect(page).toHaveURL(/\/?\?organization=campus-commons$/)
  await expect(page.locator('.hero-brand-pill')).toContainText('Commons 官方门户')
  await expect(page.locator('.app-header .brand')).toContainText('Campus Commons')
  await expect(page.getByRole('complementary', { name: '组织公开概览' })).toBeVisible()
  await expect(page.locator('body')).not.toContainText('Minecraft')
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

test('owner can create, revisit, and revoke a pending invitation', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)
  await page.goto('/admin/users')

  const email = `pending-${Date.now()}@example.com`
  await page.getByRole('button', { name: '邀请成员' }).click()
  await page.getByPlaceholder('member@example.com').fill(email)
  await page.getByRole('button', { name: '创建邀请' }).click()
  await expect(page.getByText('邀请链接已创建', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '关闭', exact: true }).click()

  const invitationRow = page.getByRole('row').filter({ hasText: email })
  await expect(invitationRow).toBeVisible()
  await invitationRow.getByRole('button', { name: '撤销' }).click()
  await page.getByRole('button', { name: '确认撤销' }).click()
  await expect(invitationRow).toHaveCount(0)
  await expect(page.getByText('当前没有待处理邀请')).toBeVisible()
})

test('batch invitations preserve per-item success and failure results', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)
  await page.goto('/admin/users')

  const email = `batch-${Date.now()}@example.com`
  await page.getByRole('button', { name: '批量邀请' }).click()
  const dialog = page.getByRole('dialog', { name: '批量邀请成员' })
  await dialog.getByRole('textbox').fill(`${email}\nnot-an-email`)
  await expect(dialog.getByText('已识别 2 个不重复邮箱，单次最多 20 个。')).toBeVisible()
  await dialog.getByRole('button', { name: '开始批量邀请' }).click()

  await expect(page.getByText('处理完成：1 条成功，1 条失败', { exact: true })).toBeVisible()
  const resultDialog = page.getByRole('dialog', { name: '批量邀请结果' })
  await expect(resultDialog.getByRole('row').filter({ hasText: email })).toContainText('已创建')
  await expect(resultDialog.getByRole('row').filter({ hasText: 'not-an-email' })).toContainText('邮箱、角色或邀请有效期不符合要求。')
  await resultDialog.getByRole('button', { name: '关闭', exact: true }).click()
  await expect(resultDialog).toBeHidden()
  await expect(page.locator('.invitation-panel').getByRole('row').filter({ hasText: email })).toBeVisible()
})

test('activity planner opens as a structured three-step workspace', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: /登录工作台/ }).click()
  await expect(page).toHaveURL(/\/admin$/)
  await page.goto('/admin/activity-planner')

  await expect(page.getByRole('heading', { name: 'AI 校园活动策划' })).toBeVisible()
  await expect(page.getByRole('link', { name: '历史方案', exact: true })).toHaveCount(0)
  await expect(page.locator('.review-queue-panel').getByRole('button', { name: '历史方案', exact: true })).toBeVisible()
  await expect(page.getByText('活动需求', { exact: true })).toBeVisible()
  await expect(page.getByText('选择依据', { exact: true })).toBeVisible()
  await expect(page.getByText('审查执行', { exact: true })).toBeVisible()
  await expect(page.getByLabel('活动名称')).toBeVisible()
  await expect(page.getByRole('heading', { name: '活动方案质量证据' })).toBeVisible()
  await expect(page.getByText('尚无人工评分。生成方案后由真实评审完成五维评价，系统不会用自动分数冒充人工结论。')).toBeVisible()
  await expect(page.getByRole('heading', { name: '待评分方案已全部完成' })).toBeVisible()

  await page.getByRole('button', { name: '历史方案' }).click()
  await expect(page.getByRole('heading', { name: '活动策划历史' })).toBeVisible()
  await expect(page.getByText('尚无活动策划记录')).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByLabel('活动名称').fill('校园开源体验日')
  await page.getByLabel('目标受众').fill('全校学生')
  await page.getByLabel('活动目标').fill('验证带引用策划、历史恢复与人工质量评分闭环')
  await page.getByRole('button', { name: '下一步：选择活动依据' }).click()
  await page.getByPlaceholder('搜索活动规范、场地要求或历史活动').fill('门户')
  await page.getByRole('button', { name: '检索' }).click()
  await page.getByRole('button', { name: /自定义门户接入约定/ }).click()
  await page.getByRole('button', { name: '生成活动方案' }).click()
  await expect(page.getByText('人工质量评分', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: '1 个方案等待你的评分' })).toBeVisible()
  await expect(page.getByText('评分只用于评估方案质量，不会触发项目、内容或审批操作。')).toBeVisible()
  const scoreRows = page.locator('.score-row')
  await expect(scoreRows).toHaveCount(5)
  for (let index = 0; index < 5; index += 1) await scoreRows.nth(index).locator('.el-rate__item').nth(3).click()
  await page.getByRole('button', { name: '保存评分' }).click()
  await expect(page.getByText(/汇总当前组织 1 次人工评价，覆盖 1 个方案/)).toBeVisible()
  await expect(page.getByRole('heading', { name: '待评分方案已全部完成' })).toBeVisible()
})
