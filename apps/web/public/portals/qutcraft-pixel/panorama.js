/* ============================================================================
 * panorama.js — 首页全景画布（像素山水 · 方形日月 · 昼夜星空 · 路过的鸡）
 * 低分辨率离屏渲染 + CSS 放大（image-rendering: pixelated），颜色实时跟随
 * PXTheme 的四季 × 昼夜调色板；慢速平移致敬某个沙盒游戏的标题界面。
 * ========================================================================== */
(function () {
  'use strict'

  function hexToRgb(hex) {
    var v = hex.replace('#', '')
    return { r: parseInt(v.slice(0, 2), 16), g: parseInt(v.slice(2, 4), 16), b: parseInt(v.slice(4, 6), 16) }
  }

  function mix(a, b, t) {
    var A = hexToRgb(a), B = hexToRgb(b)
    var r = Math.round(A.r + (B.r - A.r) * t)
    var g = Math.round(A.g + (B.g - A.g) * t)
    var bl = Math.round(A.b + (B.b - A.b) * t)
    return 'rgb(' + r + ',' + g + ',' + bl + ')'
  }

  // 整数坐标伪随机（山脊方块化、星星布局都用它，保证同一帧内稳定）
  function hash(n) {
    var x = Math.sin(n * 127.1 + 311.7) * 43758.5453
    return x - Math.floor(x)
  }

  function start(canvas) {
    var ctx = canvas.getContext('2d')
    var raf = 0
    var running = true
    var W = 0, H = 0
    var stars = []
    var pan = 0
    var last = 0
    var chickenX = -30

    var reduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches

    function resize() {
      var rect = canvas.getBoundingClientRect()
      var scale = 4 // 一个逻辑像素 = 4 个屏幕像素，保证“像素块”是正方形
      W = Math.max(160, Math.round(rect.width / scale))
      H = Math.max(90, Math.round(rect.height / scale))
      canvas.width = W
      canvas.height = H
      stars = []
      for (var i = 0; i < 46; i++) {
        stars.push({ x: Math.floor(hash(i) * W), y: Math.floor(hash(i + 100) * H * 0.55), p: hash(i + 200) * 6.28 })
      }
    }

    function draw(t) {
      var theme = window.PXTheme.get()
      var p = theme.palette
      var night = theme.time === 'night'
      var easter = theme.tweaks.easter

      var groundTop = H - Math.max(18, Math.round(H * 0.18))

      // ---- 天空：12 条横向色带（条带化 = 像素质感，拒绝平滑渐变） ----
      var skyTop = night ? mix(p.surfaceContainerLowest, '#000000', 0.35) : mix(p.primary, '#ffffff', 0.72)
      var skyLow = night ? p.surfaceContainerHigh : mix(p.primaryContainer, '#ffffff', 0.1)
      var bands = 12
      for (var i = 0; i < bands; i++) {
        ctx.fillStyle = mixColor(skyTop, skyLow, i / (bands - 1))
        ctx.fillRect(0, Math.floor((groundTop * i) / bands), W, Math.ceil(groundTop / bands) + 1)
      }

      // ---- 方形日月（懂的人自然懂） ----
      if (night) {
        stars.forEach(function (s) {
          var a = 0.35 + 0.65 * Math.abs(Math.sin(t * 0.9 + s.p))
          ctx.globalAlpha = a
          ctx.fillStyle = '#e8ecff'
          ctx.fillRect(s.x, s.y, 1, 1)
        })
        ctx.globalAlpha = 1
        ctx.fillStyle = '#e6e6f2'
        ctx.fillRect(W - 34, 12, 12, 12) // 方月亮
        ctx.fillStyle = mix('#e6e6f2', '#9a9ab0', 0.5)
        ctx.fillRect(W - 30, 16, 2, 2)
        ctx.fillRect(W - 26, 20, 3, 2) // 环形山
      } else {
        ctx.fillStyle = '#fff3b0'
        ctx.fillRect(W - 36, 10, 14, 14) // 方太阳
        ctx.fillStyle = '#fff9d8'
        ctx.fillRect(W - 34, 12, 10, 10)
      }

      // ---- 两层方块山脊（8px / 4px 步进，远景更慢、更朦胧；棕土色山体系） ----
      var skyRef = night ? p.surfaceContainerHigh : mix(p.primaryContainer, '#ffffff', 0.1)
      ridge(pan * 0.25, groundTop - Math.round(H * 0.22), Math.round(H * 0.11), 8, mixColor(p.tertiary, skyRef, night ? 0.72 : 0.55))
      ridge(pan * 0.6, groundTop - Math.round(H * 0.1), Math.round(H * 0.09), 4, mixColor(p.tertiary, skyRef, night ? 0.45 : 0.22))

      // ---- 地面：草皮 + 泥土（固定 MC 配色，夜里罩一层夜色） ----
      ctx.fillStyle = '#6f4a2c'
      ctx.fillRect(0, groundTop, W, H - groundTop)
      ctx.fillStyle = '#5e3f24'
      for (var dx = 0; dx < W; dx += 3) {
        if (hash(dx * 7) > 0.6) ctx.fillRect(dx, groundTop + 5 + Math.floor(hash(dx) * (H - groundTop - 8)), 2, 2)
      }
      ctx.fillStyle = '#69a83b'
      ctx.fillRect(0, groundTop, W, 3)
      ctx.fillStyle = '#7fbf4d'
      for (var gx = 0; gx < W; gx += 2) {
        if (hash(gx * 3) > 0.5) ctx.fillRect(gx, groundTop, 1, 2)
      }
      if (night) {
        ctx.fillStyle = 'rgba(6, 8, 26, 0.5)'
        ctx.fillRect(0, groundTop, W, H - groundTop)
      }

      // ---- 氛围生物：一只偶尔路过的鸡（彩蛋密度 ≥ 标准） ----
      if (easter !== 'subtle') {
        chickenX += 0.18
        if (chickenX > W + 16) chickenX = -30 - hash(Math.floor(t)) * 120
        drawChicken(Math.floor(chickenX), groundTop - 8 + (Math.abs(Math.sin(t * 6)) > 0.85 ? -1 : 0), night)
      }
    }

    function mixColor(a, b, t) {
      // a/b 可能是 hex 或 rgb() 字符串；统一转 rgb 数组
      var A = parseColor(a), B = parseColor(b)
      return 'rgb(' + Math.round(A[0] + (B[0] - A[0]) * t) + ',' + Math.round(A[1] + (B[1] - A[1]) * t) + ',' + Math.round(A[2] + (B[2] - A[2]) * t) + ')'
    }

    function parseColor(c) {
      if (c.charAt(0) === '#') {
        var v = hexToRgb(c)
        return [v.r, v.g, v.b]
      }
      var m = c.match(/\d+/g)
      return [+m[0], +m[1], +m[2]]
    }

    function ridge(offset, base, amp, step, color) {
      ctx.fillStyle = color
      var off = Math.floor(offset)
      for (var x = 0; x < W; x += 1) {
        var cell = Math.floor((x + off) / step)
        var h = Math.floor(hash(cell) * amp)
        ctx.fillRect(x, base - h, 1, h + (groundTop() - base) + 2)
      }
      function groundTop() { return H }
    }

    var chickenCanvas = null
    function drawChicken(x, y, night) {
      if (!chickenCanvas) {
        chickenCanvas = document.createElement('canvas')
        var def = window.PXSprites.MAPS.chicken
        chickenCanvas.width = def.map[0].length
        chickenCanvas.height = def.map.length
        var c2 = chickenCanvas.getContext('2d')
        for (var ry = 0; ry < def.map.length; ry++) {
          for (var rx = 0; rx < def.map[ry].length; rx++) {
            var ch = def.map[ry][rx]
            if (ch === '.' || ch === ' ') continue
            c2.fillStyle = def.colors[ch] || '#000'
            c2.fillRect(rx, ry, 1, 1)
          }
        }
      }
      ctx.imageSmoothingEnabled = false
      if (night) ctx.globalAlpha = 0.75
      ctx.drawImage(chickenCanvas, x, y)
      ctx.globalAlpha = 1
    }

    function frame(now) {
      if (!running) return
      var dt = last ? (now - last) / 1000 : 0
      last = now
      if (!reduced) pan += dt * 2.4 // 慢速平移：标题界面式的闲逛速度
      draw(now / 1000)
      raf = requestAnimationFrame(frame)
    }

    resize()
    window.addEventListener('resize', resize)
    if (reduced) {
      draw(0) // 尊重减弱动效：只画一帧静态画面
    } else {
      raf = requestAnimationFrame(frame)
    }

    return {
      stop: function () {
        running = false
        cancelAnimationFrame(raf)
        window.removeEventListener('resize', resize)
      },
      redraw: function () { draw(0) },
    }
  }

  window.PXPano = { start: start }
})()
