/* ============================================================================
 * components.js — 共享 UI 构建器（卡片 / 标签 / 分页器 / 弹窗 / 加载 / 空态 / 格式化）
 * 全部输出 DOM 或 HTML 字符串，供 views-*.js 组合。文案中的梗集中在
 * EASTER 字典里，方便“彩蛋密度”调档或移交时统一改写。
 * ========================================================================== */
(function () {
  'use strict'

  // ---- 工具 ----
  function esc(s) {
    return String(s == null ? '' : s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&#39;')
  }

  function el(html) {
    var t = document.createElement('template')
    t.innerHTML = html.trim()
    return t.content.firstElementChild
  }

  function fmtDate(iso) {
    if (!iso) return '—'
    var d = new Date(iso)
    if (isNaN(d)) return '—'
    var p = function (n) { return n < 10 ? '0' + n : '' + n }
    return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate())
  }

  function fmtSize(bytes) {
    if (!bytes && bytes !== 0) return '—'
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / 1048576).toFixed(1) + ' MB'
  }

  function spriteHtml(name, px) {
    // 声明式占位，交由 PXSprites.hydrate 渲染（保持 HTML 字符串拼装管线简单）
    return '<span class="ico" data-sprite="' + name + '" data-px="' + (px || 3) + '"></span>'
  }

  // ---- 标签 / 状态 ----
  var PROJECT_STATUS = {
    active: { label: '进行中', dot: 'dot--active' },
    research: { label: '调研中', dot: 'dot--research' },
    completed: { label: '已完成', dot: 'dot--completed' },
  }
  var RESOURCE_KIND = {
    document: '文档', template: '模板', package: '程序包', video: '影像',
  }

  function statusDot(status) {
    var s = PROJECT_STATUS[status] || { label: status || '未知', dot: 'dot--completed' }
    return '<span class="zh"><i class="dot ' + s.dot + '"></i> ' + s.label + '</span>'
  }

  // ---- 卡片 ----
  function lockTag(tip) {
    return '<span class="px-tag px-tag--muted" data-tip="' + (tip || '该内容仅协会成员可见&#10;先提交申请加入我们吧') + '">' + spriteHtml('lock', 2) + ' 成员</span>'
  }

  function postCard(post) {
    return (
      '<a class="card" href="#/post/' + esc(post.id) + '">' +
        '<div class="card-head">' + spriteHtml('paper', 3) +
          '<h3 class="card-title">' + esc(post.title) + '</h3>' +
          (post.members_only ? lockTag('本篇动态含成员内容&#10;完整版仅协会成员可见') : '') +
        '</div>' +
        '<p class="card-excerpt">' + esc(post.excerpt) + '</p>' +
        '<div class="card-meta"><span class="px-tag px-tag--primary">' + esc(post.category) + '</span>' +
          '<span>' + fmtDate(post.published_at) + '</span>' +
          '<span class="zh">' + (post.reading_minutes || 1) + ' 分钟</span></div>' +
      '</a>'
    )
  }

  function projectCard(p) {
    var tags = (p.tags || []).slice(0, 4).map(function (t) { return '<span class="px-tag">' + esc(t) + '</span>' }).join('')
    return (
      '<div class="card">' +
        '<div class="card-head">' + spriteHtml('pickaxe', 3) +
          '<h3 class="card-title">' + esc(p.title) + '</h3>' +
        '</div>' +
        '<p class="card-excerpt">' + esc(p.summary) + '</p>' +
        '<div class="tags">' + tags + '</div>' +
        '<div class="card-meta">' + statusDot(p.status) + '<span>' + fmtDate(p.updated_at) + '</span></div>' +
        (p.public_url ? '<a class="px-btn px-btn--ghost px-btn--sm" href="' + esc(p.public_url) + '" target="_blank" rel="noopener">访问项目站点</a>' : '') +
      '</div>'
    )
  }

  function resourceCard(r) {
    var locked = !!r.members_only
    return (
      '<div class="card">' +
        '<div class="card-head">' + spriteHtml('chest', 3) +
          '<h3 class="card-title">' + esc(r.title) + '</h3>' +
          (locked ? lockTag('该资源仅协会成员可见&#10;先提交申请加入我们吧') : '') +
        '</div>' +
        '<p class="card-excerpt">' + esc(r.description) + '</p>' +
        '<div class="card-meta"><span class="zh">' + (RESOURCE_KIND[r.kind] || r.kind) + '</span><span>' + fmtSize(r.size_bytes) + '</span><span>' + fmtDate(r.updated_at) + '</span></div>' +
        (r.download_url && !locked
          ? '<a class="px-btn px-btn--sm px-btn--primary" href="' + esc(r.download_url) + '">下载</a>'
          : '<span class="px-btn px-btn--sm px-btn--ghost" data-tip="' + (locked ? '成员资源，登录成员账号后由管理员发放' : '暂无下载地址') + '">' + (locked ? '上锁了' : '待补给') + '</span>') +
      '</div>'
    )
  }

  function knowledgeCard(a) {
    return (
      '<a class="card" href="#/article/' + esc(a.id) + '">' +
        '<div class="card-head">' + spriteHtml('book', 3) +
          '<h3 class="card-title">' + esc(a.title) + '</h3>' +
        '</div>' +
        '<p class="card-excerpt">' + esc(a.summary) + '</p>' +
        '<div class="card-meta"><span class="px-tag">' + esc(a.category) + '</span>' +
          '<span class="zh">' + (a.reading_minutes || 1) + ' 分钟</span>' +
          '<span>' + fmtDate(a.updated_at) + '</span></div>' +
      '</a>'
    )
  }

  // ---- 成就弹窗 ----
  function showAchievement(title, iconName) {
    var root = document.getElementById('toast-root')
    if (!root) return
    var node = el(
      '<div class="ach-toast" role="status">' +
        '<span class="ach-icon" data-sprite="' + (iconName || 'xpstar') + '" data-px="4"></span>' +
        '<div><div class="ach-head">进度达成！</div><div class="ach-title">' + esc(title) + '</div></div>' +
      '</div>'
    )
    root.appendChild(node)
    window.PXSprites.hydrate(node)
    setTimeout(function () { node.classList.add('leaving') }, 4200)
    setTimeout(function () { node.remove() }, 4600)
  }

  // ---- 地形加载条 ----
  var loaderTimer = 0
  function showLoader(text) {
    var root = document.getElementById('loader-root')
    root.innerHTML =
      '<div class="chunk-loader">' +
        '<div class="cl-text">' + esc(text || '正在生成地形…') + '</div>' +
        '<div class="chunk-bar"><i id="chunk-fill"></i></div>' +
      '</div>'
    root.classList.add('show')
    var fill = document.getElementById('chunk-fill')
    var w = 8
    clearInterval(loaderTimer)
    loaderTimer = setInterval(function () {
      w = Math.min(92, w + 6 + Math.random() * 8)
      if (fill) fill.style.width = w + '%'
    }, 90)
  }
  function hideLoader() {
    clearInterval(loaderTimer)
    var fill = document.getElementById('chunk-fill')
    if (fill) fill.style.width = '100%'
    setTimeout(function () {
      var root = document.getElementById('loader-root')
      root.classList.remove('show')
      root.innerHTML = ''
    }, 140)
  }

  // ---- 空状态 / 错误状态 ----
  function emptyState(text) {
    return (
      '<div class="empty-state"><div class="es-art">' + spriteHtml('chest', 6) + '</div>' +
      '<p>' + esc(text || '这里空空如也，连蠹鱼都没有。') + '</p></div>'
    )
  }
  function errorState(text) {
    return (
      '<div class="err-panel px-panel">' +
        '<div class="es-art">' + spriteHtml('redstone', 5) + '</div>' +
        '<h2>连接中断</h2><p>' + esc(text || '与主城的通讯出了问题，请稍后重试。') + '</p>' +
        '<button class="px-btn" type="button" data-retry>重新连接</button>' +
      '</div>'
    )
  }

  function mdLite(src) {
    if (window.PXMd && typeof window.PXMd.render === 'function') return window.PXMd.render(src)
    return '<p>' + esc(src) + '</p>'
  }

  // ---- 分页器（§2b-1：上一页 / 页码 / 下一页，像素按钮 + 禁用态） ----
  // 返回 DOM 元素；onPage(targetPage) 在点击可用页码时回调
  function pagerEl(pageNo, pageSize, total, onPage) {
    var pages = Math.max(1, Math.ceil(total / pageSize))
    var wrap = el('<div class="px-pager" role="navigation" aria-label="分页"></div>')

    function addBtn(label, target, opts) {
      opts = opts || {}
      var cls = 'px-btn px-btn--sm px-btn--ghost' + (opts.current ? ' px-btn--primary' : '')
      var b = el('<button class="' + cls + '" type="button">' + esc(label) + '</button>')
      if (opts.disabled) b.setAttribute('disabled', '')
      else b.addEventListener('click', function () { if (target !== pageNo) onPage(target) })
      wrap.appendChild(b)
    }
    function addEllipsis() {
      wrap.appendChild(el('<span class="px-pager-meta">…</span>'))
    }

    addBtn('‹ 上一页', pageNo - 1, { disabled: pageNo <= 1 })
    var windowSize = 5
    var start = Math.max(1, Math.min(pageNo - 2, pages - windowSize + 1))
    var end = Math.min(pages, start + windowSize - 1)
    if (start > 1) { addBtn('1', 1); if (start > 2) addEllipsis() }
    for (var i = start; i <= end; i++) addBtn('' + i, i, { current: i === pageNo })
    if (end < pages) { if (end < pages - 1) addEllipsis(); addBtn('' + pages, pages) }
    addBtn('下一页 ›', pageNo + 1, { disabled: pageNo >= pages })

    wrap.appendChild(el('<span class="px-pager-meta">共 ' + total + ' 条 / ' + pages + ' 页</span>'))
    return wrap
  }

  window.PXUi = {
    esc: esc, el: el, fmtDate: fmtDate, fmtSize: fmtSize, spriteHtml: spriteHtml,
    postCard: postCard, projectCard: projectCard, resourceCard: resourceCard, knowledgeCard: knowledgeCard,
    pagerEl: pagerEl, lockTag: lockTag,
    showAchievement: showAchievement, showLoader: showLoader, hideLoader: hideLoader,
    emptyState: emptyState, errorState: errorState, mdLite: mdLite,
    PROJECT_STATUS: PROJECT_STATUS, RESOURCE_KIND: RESOURCE_KIND,
  }
})()
