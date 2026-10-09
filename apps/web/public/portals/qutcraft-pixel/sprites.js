/* ============================================================================
 * sprites.js — 逐像素精灵图系统
 * 用 box-shadow 把字符画渲染成像素图标，零图片资源、零请求。
 * 用法：PXSprites.make('grass', 4) 或声明式 <span data-sprite="grass" data-px="4">
 * 新增图标：在 MAPS 里加一张字符画 + 颜色表即可。
 * ========================================================================== */
(function () {
  'use strict'

  // 字符画约定：'.'/空格 = 透明，其余字符查 colors 表
  var MAPS = {
    // 草方块：顶三层草皮 + 底下泥土
    grass: {
      map: [
        'aAabaAab',
        'AbaAabaA',
        'baAbaAba',
        'cCdCdcCd',
        'dCdcCdcD',
        'CdcDdCdc',
        'dCdcdCdD',
        'CdDdCdDc',
      ],
      colors: { a: '#7fbf4d', A: '#69a83b', b: '#8fd05f', c: '#8a5f3c', C: '#6f4a2c', d: '#7a5233', D: '#5e3f24' },
    },
    // 生命之心（会员/健康语义）
    heart: {
      map: [
        '.RR..RR.',
        'RrRRRRrR',
        'RrRRRRRR',
        'RRRRRRRR',
        '.RRRRRR.',
        '..RRRR..',
        '...RR...',
      ],
      colors: { R: '#d42a2a', r: '#8f1d1d' },
    },
    // 经验星光（XP 语义）
    xpstar: {
      map: [
        '...G....',
        '..GgG...',
        '.GgggG..',
        'GgggggG.',
        '.GgggG..',
        '..GgG...',
        '...G....',
      ],
      colors: { G: '#80ff20', g: '#c8ff8a' },
    },
    // 苦力怕脸（页脚水印彩蛋，无文字标注）
    creeper: {
      map: [
        'cCccCcCc',
        'CcKcKcCc',
        'cKKKKKKc',
        'CcKKKKcC',
        'ccKKKKcc',
        'CcKccKcC',
        'cKkcckKc',
        'CcCccCcC',
      ],
      colors: { c: '#4b9e2f', C: '#3c8125', K: '#141414', k: '#2b2b2b' },
    },
    // 书（知识库）
    book: {
      map: [
        '.BBBBBB.',
        'BbbbbbbB',
        'BlllllbB',
        'BbbbbbbB',
        'BllllbbB',
        'BlllllbB',
        'BbbbbbbB',
        '.BBBBBB.',
      ],
      colors: { B: '#8a5f3c', b: '#e8e0c8', l: '#9a8f70' },
    },
    // 箱子（资源中心 / 空状态）
    chest: {
      map: [
        '.CCCCCC.',
        'CccccccC',
        'CkKKKKkC',
        'CkKttKkC',
        'CCCCCCCC',
        'CccccccC',
        'CccccccC',
        '.CCCCCC.',
      ],
      colors: { C: '#4a2f16', c: '#8a5f2e', K: '#3a2410', k: '#6b4a22', t: '#d8c25a' },
    },
    // 镐（项目）
    pickaxe: {
      map: [
        '....PPPP',
        '..PppppP',
        '.Pp..P..',
        'P....P..',
        '....P...',
        '...P....',
        '..P.....',
        '.P......',
      ],
      colors: { P: '#7a5233', p: '#c8c8d0' },
    },
    // 鸡（全景图氛围生物，偶尔路过）
    chicken: {
      map: [
        '..www...',
        '.wwwww..',
        '.wwKwwo.',
        '.wwwwr..',
        '.wwwww..',
        '..www...',
        '...l.l..',
        '...l.l..',
      ],
      colors: { w: '#f2f2f2', K: '#141414', o: '#e8912a', r: '#c23a2e', l: '#e8b25a' },
    },
    // 纸（申请书 / 公告）
    paper: {
      map: [
        '.PPPPPP.',
        'PwwwwwwP',
        'PwllllwP',
        'PwwwwwwP',
        'PwllllwP',
        'PwlllwwP',
        'PwwwwwwP',
        '.PPPPPP.',
      ],
      colors: { P: '#b5ab90', w: '#f6f2e4', l: '#8f8872' },
    },
    // 红石（缺陷反馈）
    redstone: {
      map: [
        '...RR...',
        '..RrrR..',
        '.RrRRrR.',
        'RrRRRRrR',
        'RrRRRRrR',
        '.RrRRrR.',
        '..RrrR..',
        '...RR...',
      ],
      colors: { R: '#d43a2a', r: '#7e1d12' },
    },
    // 锁（仅成员可见资源）
    lock: {
      map: [
        '..SSSS..',
        '.S....S.',
        '.S....S.',
        'BBBBBBBB',
        'BbbbbbbB',
        'BbbkkbbB',
        'BbbkkbbB',
        'BBBBBBBB',
      ],
      colors: { S: '#9a9aa6', B: '#c9a227', b: '#a8873a', k: '#3a2f10' },
    },
  }

  function shadowFor(map, colors, px) {
    var out = []
    for (var y = 0; y < map.length; y++) {
      var row = map[y]
      for (var x = 0; x < row.length; x++) {
        var ch = row[x]
        if (ch === '.' || ch === ' ') continue
        var color = colors[ch]
        if (!color) continue
        out.push(x * px + 'px ' + y * px + 'px 0 0 ' + color)
      }
    }
    return out.join(',')
  }

  function make(name, px) {
    px = px || 3
    var def = MAPS[name]
    var wrap = document.createElement('span')
    wrap.className = 'pxs-wrap'
    wrap.setAttribute('aria-hidden', 'true')
    if (!def) return wrap
    var cell = document.createElement('span')
    cell.className = 'pxs'
    cell.style.width = px + 'px'
    cell.style.height = px + 'px'
    cell.style.boxShadow = shadowFor(def.map, def.colors, px)
    wrap.style.width = def.map[0].length * px + 'px'
    wrap.style.height = def.map.length * px + 'px'
    wrap.appendChild(cell)
    return wrap
  }

  // 声明式水合：<span data-sprite="grass" data-px="4"></span>
  function hydrate(root) {
    ;(root || document).querySelectorAll('[data-sprite]').forEach(function (host) {
      if (host.dataset.done) return
      host.dataset.done = '1'
      host.appendChild(make(host.getAttribute('data-sprite'), parseInt(host.getAttribute('data-px') || '3', 10)))
    })
  }

  window.PXSprites = { MAPS: MAPS, make: make, hydrate: hydrate, shadowFor: shadowFor }
})()
