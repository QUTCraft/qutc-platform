import { useRouter } from 'vue-router'

export function usePageTransition() {
  const router = useRouter()

  const navigateToApply = (e?: MouseEvent) => {
    const x = e && e.clientX ? e.clientX : window.innerWidth / 2
    const y = e && e.clientY ? e.clientY : window.innerHeight / 2

    // Dynamically calculate distance to farthest screen corner to guarantee 100% full coverage
    const distTopLeft = Math.hypot(x, y)
    const distTopRight = Math.hypot(window.innerWidth - x, y)
    const distBottomLeft = Math.hypot(x, window.innerHeight - y)
    const distBottomRight = Math.hypot(window.innerWidth - x, window.innerHeight - y)
    const maxDist = Math.max(distTopLeft, distTopRight, distBottomLeft, distBottomRight)
    const scaleFactor = Math.ceil((maxDist / 20) * 1.3)

    const curtain = document.createElement('div')
    curtain.style.cssText = `
      position: fixed;
      left: ${x}px;
      top: ${y}px;
      width: 40px;
      height: 40px;
      margin-left: -20px;
      margin-top: -20px;
      border-radius: 50%;
      background-color: #0b0b0d;
      z-index: 99999;
      pointer-events: none;
      transition: transform 750ms cubic-bezier(0.16, 1, 0.3, 1), opacity 300ms ease 750ms;
      transform: scale(0);
      box-shadow: 0 0 40px rgba(0,0,0,0.8);
    `
    document.body.appendChild(curtain)

    requestAnimationFrame(() => {
      curtain.style.transform = `scale(${scaleFactor})`
    })

    setTimeout(() => {
      router.push('/apply')
    }, 650)

    setTimeout(() => {
      curtain.style.opacity = '0'
      setTimeout(() => curtain.remove(), 400)
    }, 950)
  }

  return {
    navigateToApply,
  }
}
