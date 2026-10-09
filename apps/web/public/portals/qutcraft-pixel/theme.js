/* ============================================================================
 * theme.js — 昼夜时钟引擎（像素门户版）
 * 机制：本地时间 7:00–19:00 = 昼，其余 = 夜；每 60s 重算，把调色板写入 <html>
 * 的 --px-* CSS 变量 + data-time / data-easter 属性。
 * 配色：单套原创像素色系（暖调 terracotta/泥土系，与草方块精灵图同源），
 * 只做昼/夜两版——四季方案已被设计评审砍掉（v0.3 决议）。
 * 昼夜覆盖与彩蛋密度持久化在 localStorage 'px.tweaks'。
 * 生产页脚只提供昼/夜/自动开关；彩蛋密度沿用已保存值，默认 standard。
 * ========================================================================== */
(function () {
  'use strict'

  // 17 槽位结构沿用平台 Monet 方案；配色值为像素门户原创
  var PALETTES = {
    day: {
      primary: '#d94f16', onPrimary: '#ffffff', primaryContainer: '#ffd9c2', onPrimaryContainer: '#421600',
      secondary: '#e6a800', secondaryContainer: '#ffedb3', tertiary: '#7a4a2b', tertiaryContainer: '#e8d5c4',
      surfaceContainerLowest: '#fffdf8', surfaceContainerLow: '#fff8ec', surfaceContainer: '#fcebd6', surfaceContainerHigh: '#f4dab9',
      onSurface: '#26170f', onSurfaceVariant: '#5a4536', outlineVariant: '#eec9a6',
    },
    night: {
      primary: '#ff9158', onPrimary: '#3a1404', primaryContainer: '#9c3d10', onPrimaryContainer: '#ffdfc9',
      secondary: '#ffd968', secondaryContainer: '#6e5200', tertiary: '#d0a887', tertiaryContainer: '#54331d',
      surfaceContainerLowest: '#1b130d', surfaceContainerLow: '#241a12', surfaceContainer: '#2f2115', surfaceContainerHigh: '#3d2b1b',
      onSurface: '#f7ead9', onSurfaceVariant: '#e0c9ae', outlineVariant: '#7a5636',
    },
  }

  var VAR_MAP = {
    primary: '--px-primary',
    onPrimary: '--px-on-primary',
    primaryContainer: '--px-primary-container',
    onPrimaryContainer: '--px-on-primary-container',
    secondary: '--px-secondary',
    secondaryContainer: '--px-secondary-container',
    tertiary: '--px-tertiary',
    tertiaryContainer: '--px-tertiary-container',
    surfaceContainerLowest: '--px-bg',
    surfaceContainerLow: '--px-surface',
    surfaceContainer: '--px-panel',
    surfaceContainerHigh: '--px-panel-2',
    onSurface: '--px-text',
    onSurfaceVariant: '--px-text-2',
    outlineVariant: '--px-line',
  }

  var TWEAKS_KEY = 'px.tweaks'
  var TWEAK_DEFAULTS = { time: 'auto', easter: 'standard' }

  function timeAt(date) {
    var h = date.getHours()
    return h >= 7 && h < 19 ? 'day' : 'night'
  }

  function readTweaks() {
    try {
      var raw = JSON.parse(localStorage.getItem(TWEAKS_KEY) || '{}')
      var out = {}
      for (var k in TWEAK_DEFAULTS) out[k] = raw[k] !== undefined ? raw[k] : TWEAK_DEFAULTS[k]
      return out
    } catch (e) {
      var fallback = {}
      for (var k in TWEAK_DEFAULTS) fallback[k] = TWEAK_DEFAULTS[k]
      return fallback
    }
  }

  function writeTweaks(t) {
    try { localStorage.setItem(TWEAKS_KEY, JSON.stringify(t)) } catch (e) { /* file:// 私密模式等场景下静默 */ }
  }

  var listeners = []
  var state = null

  function compute() {
    var t = readTweaks()
    var time = t.time === 'auto' ? timeAt(new Date()) : t.time
    if (time !== 'day' && time !== 'night') time = timeAt(new Date())
    return { time: time, palette: PALETTES[time], tweaks: t }
  }

  function apply() {
    state = compute()
    var root = document.documentElement
    for (var slot in VAR_MAP) root.style.setProperty(VAR_MAP[slot], state.palette[slot])
    root.setAttribute('data-time', state.time)
    root.setAttribute('data-easter', state.tweaks.easter)
    if (document.body) document.body.classList.toggle('is-night', state.time === 'night')
    listeners.slice().forEach(function (fn) {
      try { fn(state) } catch (e) { console.error('[px-theme]', e) }
    })
  }

  window.PXTheme = {
    get: function () { return state },
    onChange: function (fn) { listeners.push(fn); return function () { listeners = listeners.filter(function (f) { return f !== fn }) } },
    setTweaks: function (patch) { writeTweaks(Object.assign(readTweaks(), patch)); apply() },
    getTweaks: readTweaks,
    PALETTES: PALETTES,
    timeAt: timeAt,
  }

  apply()
  // 分钟级时钟：跨 7:00 / 19:00 边界无需刷新即换肤
  setInterval(apply, 60000)
})()
