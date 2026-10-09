/* ============================================================================
 * app.js — 路由 / 顶栏与 hotbar / 页脚（F3 调试行 + 昼夜开关）
 * Hash 路由（静态包无需服务端 rewrite）：
 *   #/            首页        #/posts      动态列表    #/post/:id   动态详情
 *   #/projects    公开项目    #/resources  资源中心    #/resource/:id 资源详情
 *   #/knowledge   知识库      #/article/:id 知识文章
 *   #/apply       申请向导    #/report     问题反馈
 * 登录 / 注册 / 工作台由主应用整页承担（runtime 永不接管这些路径）。
 * ========================================================================== */
(function () {
  'use strict'

  var NAV = [
    { hash: '#/', label: '首页', icon: 'grass' },
    { hash: '#/posts', label: '动态', icon: 'paper' },
    { hash: '#/projects', label: '项目', icon: 'pickaxe' },
    { hash: '#/resources', label: '资源', icon: 'chest' },
    { hash: '#/knowledge', label: '知识', icon: 'book' },
  ]

  var cleanup = null

  function organizationQuery() {
    var slug = window.PXApi && window.PXApi.organizationSlug
    return slug ? '?organization=' + encodeURIComponent(slug) : ''
  }

  function platformHref(path) {
    return path + organizationQuery()
  }

  function buildChrome() {
    var topbar = document.getElementById('topbar')
    var slots = NAV.map(function (n, i) {
      return '<a class="nav-slot" data-nav="' + n.hash + '" href="' + n.hash + '">' +
        '<span class="slot-num">' + (i + 1) + '</span>' +
        '<span class="hb-ico" data-sprite="' + n.icon + '" data-px="2"></span>' +
        '<span>' + n.label + '</span></a>'
    }).join('')
    topbar.innerHTML =
      '<a class="brand" href="#/"><span data-sprite="grass" data-px="4"></span>' +
        '<span><span class="brand-name">QUTCraft 像素协会</span><br><span class="brand-sub">QUTCRAFT · EST.2026</span></span></a>' +
      '<nav class="nav">' + slots +
        '<a class="nav-link" href="' + platformHref('/login') + '">登录</a>' +
        '<a class="nav-link" href="' + platformHref('/admin') + '">工作台</a>' +
        '<a class="px-btn px-btn--primary px-btn--sm btn-join" href="#/apply">加入协会</a>' +
      '</nav>' +
      '<a class="px-btn px-btn--xp px-btn--sm btn-skin" href="https://skin.qutcraft.cn/" target="_blank" rel="noopener noreferrer" aria-label="打开 QUTC Skin 皮肤站">' +
        '皮肤站<span class="btn-skin-mark" aria-hidden="true">↗</span>' +
      '</a>'

    var hotbar = document.createElement('nav')
    hotbar.className = 'hotbar'
    hotbar.innerHTML = NAV.concat([{ hash: '#/apply', label: '加入', icon: 'heart' }]).map(function (n, i) {
      return '<a class="nav-slot" data-nav="' + n.hash + '" href="' + n.hash + '">' +
        '<span class="slot-num">' + (i + 1) + '</span>' +
        '<span class="hb-ico" data-sprite="' + n.icon + '" data-px="3"></span>' +
        '<span>' + n.label + '</span></a>'
    }).join('')
    document.body.appendChild(hotbar)

    window.PXSprites.hydrate(topbar)
    window.PXSprites.hydrate(hotbar)
  }

  function markActiveNav(hash) {
    var current = '#/'
    var lookup = hash
    if (hash.indexOf('#/post/') === 0) lookup = '#/posts'
    if (hash.indexOf('#/article/') === 0) lookup = '#/knowledge'
    if (hash.indexOf('#/resource/') === 0) lookup = '#/resources'
    for (var i = 0; i < NAV.length; i++) {
      if (lookup === NAV[i].hash || (NAV[i].hash !== '#/' && lookup.indexOf(NAV[i].hash) === 0)) current = NAV[i].hash
    }
    document.querySelectorAll('[data-nav]').forEach(function (a) {
      a.classList.toggle('active', a.getAttribute('data-nav') === current || (a.getAttribute('data-nav') === '#/apply' && hash === '#/apply'))
    })
  }

  document.addEventListener('click', function (e) {
    var a = e.target && e.target.closest ? e.target.closest('a[href="#"]') : null
    if (a) e.preventDefault()
  })

  function buildFooter() {
    var footer = document.getElementById('footer')
    var md3Href = '/?portal=md3' + (organizationQuery() ? '&organization=' + encodeURIComponent(window.PXApi.organizationSlug) : '')
    footer.innerHTML =
      '<div class="footer-inner">' +
        '<span class="f-brand">QUTCraft · 像素艺术创作协会</span>' +
        '<span class="f-links">' +
          '<a href="#/apply">加入协会</a><a href="#/report">问题反馈</a>' +
          '<a href="' + platformHref('/admin') + '">工作台</a>' +
          '<a href="' + md3Href + '">MD3 门户</a>' +
        '</span>' +
        '<span class="time-switch" role="group" aria-label="昼夜切换">' +
          '<button type="button" data-time="day">昼</button>' +
          '<button type="button" data-time="night">夜</button>' +
          '<button type="button" data-time="auto">自动</button>' +
        '</span>' +
        '<span class="f-creeper" data-tip="ssssSSS……&#10;放心，这里是安全区。"><span data-sprite="creeper" data-px="2"></span></span>' +
      '</div>' +
      '<div class="f3-line" id="f3-line"></div>'
    window.PXSprites.hydrate(footer)

    function markTimeSwitch() {
      var current = window.PXTheme.getTweaks().time || 'auto'
      footer.querySelectorAll('.time-switch button').forEach(function (btn) {
        btn.classList.toggle('is-active', btn.getAttribute('data-time') === current)
      })
    }
    footer.querySelectorAll('.time-switch button').forEach(function (btn) {
      btn.addEventListener('click', function () {
        window.PXTheme.setTweaks({ time: btn.getAttribute('data-time') })
        markTimeSwitch()
      })
    })
    markTimeSwitch()

    var update = function () {
      var t = window.PXTheme.get()
      var f3 = document.getElementById('f3-line')
      if (!f3) return
      if (t.tweaks.easter === 'subtle') {
        f3.textContent = '© 2026 QUTCraft · 计算机像素艺术创作协会'
      } else {
        f3.textContent =
          'XYZ: 120.4949 / 64 / 36.0671 · biome: qut:pixel_plains · seed: 2026 · ' +
          (t.time === 'night' ? 'night' : 'day') + ' · 60 fps · © 2026 QUTCraft'
      }
      markTimeSwitch()
    }
    update()
    window.PXTheme.onChange(update)
  }

  function route() {
    var hash = location.hash || '#/'
    var view = document.getElementById('view')
    markActiveNav(hash)

    if (typeof cleanup === 'function') { try { cleanup() } catch (e) {} cleanup = null }

    window.PXUi.showLoader('正在生成地形…')

    var done = function (fn) {
      setTimeout(function () {
        window.PXUi.hideLoader()
        if (typeof fn === 'function') cleanup = fn(view) || null
        window.scrollTo(0, 0)
      }, 260)
    }

    var m
    if (hash === '#/' || hash === '#') done(window.PXViews.home)
    else if (hash === '#/posts') done(window.PXViews.posts)
    else if ((m = hash.match(/^#\/post\/([\w-]+)/))) done(function (v) { window.PXViews.postDetail(v, m[1]) })
    else if (hash === '#/projects') done(window.PXViews.projects)
    else if (hash === '#/resources') done(window.PXViews.resources)
    else if ((m = hash.match(/^#\/resource\/([\w-]+)/))) done(function (v) { window.PXViews.resourceDetail(v, m[1]) })
    else if (hash === '#/knowledge') done(window.PXViews.knowledge)
    else if ((m = hash.match(/^#\/article\/([\w-]+)/))) done(function (v) { window.PXViews.articleDetail(v, m[1]) })
    else if (hash === '#/apply') done(window.PXViews.apply)
    else if (hash === '#/report') done(window.PXViews.report)
    else done(window.PXViews.notFound)
  }

  buildChrome()
  buildFooter()
  window.addEventListener('hashchange', route)
  route()
})()
