import { expect, test } from '@playwright/test'

test.describe('automatic seasonal day/night theme', () => {
  test.use({ timezoneId: 'Asia/Hong_Kong' })

  test('refreshes at time and month boundaries without reloading', async ({ page }) => {
    await page.clock.install({ time: new Date('2026-08-31T18:59:30+08:00') })
    await page.goto('/')

    const root = page.locator('html')
    await expect(root).toHaveAttribute('data-monet-season', 'summer')
    await expect(root).toHaveAttribute('data-monet-time', 'day')
    await expect(root).toHaveAttribute('data-theme', 'light')

    await page.clock.fastForward(5 * 60 * 60 * 1000 + 2 * 60 * 1000)
    await expect(root).toHaveAttribute('data-monet-season', 'autumn')
    await expect(root).toHaveAttribute('data-monet-time', 'night')
    await expect(root).toHaveAttribute('data-theme', 'dark')

    await page.clock.fastForward(7 * 60 * 60 * 1000)
    await expect(root).toHaveAttribute('data-monet-season', 'autumn')
    await expect(root).toHaveAttribute('data-monet-time', 'day')
    await expect(root).toHaveAttribute('data-theme', 'light')
  })
})
