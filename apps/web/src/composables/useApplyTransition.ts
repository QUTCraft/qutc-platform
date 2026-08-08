import { useRouter } from 'vue-router'

const CURTAIN_SIZE = 40
const ROUTE_DELAY = 650
const CLEANUP_DELAY = 1_350

/**
 * Navigate to the application page with a local radial black-mask reveal.
 *
 * This is intentionally scoped to the application CTA. It does not wrap
 * RouterView, so normal route changes keep their stable, content-preserving
 * behavior.
 */
export function useApplyTransition() {
  const router = useRouter()

  const navigateToApply = (event?: MouseEvent) => {
    const x = event?.clientX || window.innerWidth / 2
    const y = event?.clientY || window.innerHeight / 2
    const farthestCorner = Math.max(
      Math.hypot(x, y),
      Math.hypot(window.innerWidth - x, y),
      Math.hypot(x, window.innerHeight - y),
      Math.hypot(window.innerWidth - x, window.innerHeight - y),
    )
    const scaleFactor = Math.ceil((farthestCorner / (CURTAIN_SIZE / 2)) * 1.3)

    const curtain = document.createElement('div')
    curtain.className = 'apply-transition-curtain'
    curtain.style.left = `${x}px`
    curtain.style.top = `${y}px`
    curtain.style.setProperty('--apply-transition-scale', String(scaleFactor))
    document.body.appendChild(curtain)

    requestAnimationFrame(() => {
      curtain.classList.add('is-expanding')
    })

    window.setTimeout(() => {
      void router.push({ name: 'apply' })
    }, ROUTE_DELAY)

    window.setTimeout(() => {
      curtain.classList.add('is-fading')
      window.setTimeout(() => curtain.remove(), 400)
    }, CLEANUP_DELAY)
  }

  return { navigateToApply }
}
