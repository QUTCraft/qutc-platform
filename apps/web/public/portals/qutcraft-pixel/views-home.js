/* ============================================================================
 * views-home.js — 首页：全景 Hero / 统计条 / 招新状态卡 / 三柱卡 /
 * 协会速览 / 最新动态（lead + 行式列表）/ 资源·知识 compact 双栏 / 项目 / CTA
 * v0.4（§2a-4）：草方块分隔条全站仅保留 hero 后一条；新区块见 §2b-6/7/8/9/10。
 * ========================================================================== */
(function () {
  'use strict'

  var SPLASHES = [
    '不是 BUG，是特性！',
    '也在月球上盖过房子！',
    '红石也是艺术！',
    '别在深夜挖矿！',
    '种子号：2026',
    '100% 手工像素！',
  ]

  function render(root) {
    var Ui = window.PXUi
    var Api = window.PXApi

    root.innerHTML =
      '<section class="hero">' +
        '<canvas class="hero-pano" id="hero-pano" aria-hidden="true"></canvas>' +
        '<div class="hero-inner">' +
          '<div class="hero-wordmark"><span class="hero-logo">QUTCRAFT</span>' +
            '<span class="splash" id="splash"></span>' +
          '</div>' +
          '<div class="hero-cn">计算机像素艺术创作协会</div>' +
          '<p class="hero-tagline" id="hero-tagline">以像素为语言，以方块丈量世界。</p>' +
          '<div class="hero-cta">' +
            '<a class="px-btn px-btn--xp" href="#/apply">加入协会</a>' +
            '<a class="px-btn px-btn--ghost" href="#/projects">看看我们在造什么</a>' +
          '</div>' +
        '</div>' +
      '</section>' +

      // §2a-4：全站唯一一条草方块分隔条（hero 底部）
      '<div class="grass-strip" aria-hidden="true"></div>' +

      // §2b-6：统计条（动态 / 项目 / 知识 total 计数）
      '<section class="section container">' +
        '<div class="stat-strip" id="home-stats"></div>' +
      '</section>' +

      // §2b-7：招新状态卡
      '<section class="section container">' +
        '<div class="px-panel recruit-card">' +
          '<span class="recruit-dot" aria-hidden="true"></span>' +
          '<div class="recruit-info">' +
            '<strong>招新通道开放</strong>' +
            '<span>白名单与社团成员双通道 · 审批约三个工作日</span>' +
          '</div>' +
          '<a class="px-btn px-btn--xp px-btn--sm" href="#/apply">去申请</a>' +
        '</div>' +
      '</section>' +

      // §2b-8：三柱介绍卡
      '<section class="section container">' +
        '<div class="section-head"><h2>三个方向</h2><span class="en">// PILLARS</span></div>' +
        '<div class="grid grid--3">' +
          '<div class="px-panel pillar-card">' + Ui.spriteHtml('xpstar', 5) +
            '<h3>像素美术</h3>' +
            '<p>从 16×16 开始练习限色、抖动与逐帧动画，每月一次主题周练，作品直指作品集。</p></div>' +
          '<div class="px-panel pillar-card">' + Ui.spriteHtml('pickaxe', 5) +
            '<h3>游戏开发</h3>' +
            '<p>Game Jam、引擎入门课与结对开发，把脑洞做成能玩的 demo，再到能发布的作品。</p></div>' +
          '<div class="px-panel pillar-card">' + Ui.spriteHtml('grass', 5) +
            '<h3>协作建造</h3>' +
            '<p>百人同图的大型工程：从生存小屋到 1:1 校园复原，学习分工、规范与验收。</p></div>' +
        '</div>' +
      '</section>' +

      '<section class="section container" id="sec-intro"></section>' +

      // §2b-9：最新动态 = 首条 lead 大卡 + 3 条行式列表
      '<section class="section container">' +
        '<div class="section-head"><h2>最新动态</h2><span class="en">// LATEST</span>' +
          '<a class="more" href="#/posts">全部动态 →</a></div>' +
        '<div class="grid grid--2">' +
          '<div id="home-post-lead"></div>' +
          '<div class="px-panel"><div class="row-list" id="home-post-rows"></div></div>' +
        '</div>' +
      '</section>' +

      // §2b-10：资源 / 知识 compact 双栏
      '<section class="section container">' +
        '<div class="grid grid--2">' +
          '<div class="px-panel">' +
            '<div class="compact-head">' + Ui.spriteHtml('chest', 3) + '<h3>资源中心</h3>' +
              '<a class="more" href="#/resources">全部 →</a></div>' +
            '<div class="row-list" id="home-res-rows"></div>' +
          '</div>' +
          '<div class="px-panel">' +
            '<div class="compact-head">' + Ui.spriteHtml('book', 3) + '<h3>知识库</h3>' +
              '<a class="more" href="#/knowledge">全部 →</a></div>' +
            '<div class="row-list" id="home-art-rows"></div>' +
          '</div>' +
        '</div>' +
      '</section>' +

      '<section class="section container">' +
        '<div class="section-head"><h2>进行中的项目</h2><span class="en">// ACTIVE</span>' +
          '<a class="more" href="#/projects">全部项目 →</a></div>' +
        '<div class="grid grid--2" id="home-projects"></div>' +
      '</section>' +

      '<section class="section container">' +
        '<div class="px-panel" style="text-align:center;padding:34px 22px">' +
          '<h2 style="letter-spacing:4px">准备好放下第一块方块了吗</h2>' +
          '<p style="color:var(--px-text-2);margin-bottom:20px">白名单与社团成员通道全年开放，审批通常在三个工作日内完成。</p>' +
          '<a class="px-btn px-btn--primary" href="#/apply">开始申请</a>' +
        '</div>' +
      '</section>'

    // splash 文案每次进首页随机一条（彩蛋密度 = subtle 时 CSS 已隐藏）
    var splash = root.querySelector('#splash')
    splash.textContent = SPLASHES[Math.floor(Math.random() * SPLASHES.length)]

    var stopPano = null
    var pano = root.querySelector('#hero-pano')
    if (pano) stopPano = window.PXPano.start(pano)

    // §2b-6：统计条（各列表 total → 像素数字牌）
    Promise.all([
      Api.getPosts({ page: 1, page_size: 1 }),
      Api.getProjects({ page: 1, page_size: 1 }),
      Api.getKnowledgeArticles({ page: 1, page_size: 1 }),
    ]).then(function (r) {
      var box = root.querySelector('#home-stats')
      if (!box) return
      var tiles = [
        { num: r[0].total, label: '协会动态', en: 'POSTS' },
        { num: r[1].total, label: '公开项目', en: 'PROJECTS' },
        { num: r[2].total, label: '知识文章', en: 'KNOWLEDGE' },
      ]
      box.innerHTML = tiles.map(function (t) {
        return '<div class="px-panel stat-tile"><span class="stat-num">' + t.num + '</span>' +
          '<span class="stat-label">' + t.label + '<small>// ' + t.en + '</small></span></div>'
      }).join('')
    }).catch(function () {
      var box = root.querySelector('#home-stats')
      if (box) box.innerHTML = ''
    })

    // 协会速览
    Api.getOrganization().then(function (org) {
      var links = (org.social_links || []).map(function (l) {
        return '<a class="px-btn px-btn--ghost px-btn--sm" href="' + Ui.esc(l.href) + '" target="_blank" rel="noopener">' + Ui.esc(l.label) + '</a>'
      }).join(' ')
      var box = root.querySelector('#sec-intro')
      if (!box) return
      box.innerHTML =
        '<div class="section-head"><h2>协会速览</h2><span class="en">// ABOUT</span></div>' +
        '<div class="px-panel">' +
          '<p style="font-size:15px">' + Ui.esc(org.introduction) + '</p>' +
          '<p style="margin-bottom:14px"><strong>联系邮箱：</strong><a href="mailto:' + Ui.esc(org.contact_email) + '">' + Ui.esc(org.contact_email) + '</a></p>' +
          '<div style="display:flex;gap:10px;flex-wrap:wrap">' + links + '</div>' +
        '</div>'
      var tag = root.querySelector('#hero-tagline')
      if (tag) tag.textContent = org.tagline
    }).catch(function () { /* 原型：简介缺席时不阻塞首屏 */ })

    // §2b-9：最新动态（1 条 lead 大卡 + 3 条行式列表）
    Api.getPosts({ page: 1, page_size: 4 }).then(function (res) {
      var leadBox = root.querySelector('#home-post-lead')
      var rowBox = root.querySelector('#home-post-rows')
      if (!leadBox || !rowBox) return
      if (!res.items.length) {
        leadBox.innerHTML = Ui.emptyState('还没有动态，去催更会长。')
        return
      }
      var lead = res.items[0]
      leadBox.innerHTML =
        '<a class="card lead-card" href="#/post/' + Ui.esc(lead.id) + '">' +
          '<div class="card-head">' + Ui.spriteHtml('paper', 4) +
            '<h3 class="card-title">' + Ui.esc(lead.title) + '</h3>' +
            (lead.members_only ? Ui.lockTag('本篇动态含成员内容&#10;完整版仅协会成员可见') : '') +
          '</div>' +
          '<p class="card-excerpt">' + Ui.esc(lead.excerpt) + '</p>' +
          '<div class="card-meta"><span class="px-tag px-tag--primary">' + Ui.esc(lead.category) + '</span>' +
            '<span>' + Ui.fmtDate(lead.published_at) + '</span>' +
            '<span class="zh">' + (lead.reading_minutes || 1) + ' 分钟</span></div>' +
        '</a>'
      rowBox.innerHTML = res.items.slice(1).map(function (p) {
        return '<a class="row-item" href="#/post/' + Ui.esc(p.id) + '">' +
          '<span class="ri-title">' + Ui.esc(p.title) + '</span>' +
          '<span class="ri-date">' + Ui.fmtDate(p.published_at) + '</span></a>'
      }).join('')
      window.PXSprites.hydrate(leadBox)
    }).catch(function () {
      var leadBox = root.querySelector('#home-post-lead')
      if (leadBox) { leadBox.innerHTML = Ui.errorState('动态加载失败。'); window.PXSprites.hydrate(leadBox) }
    })

    // §2b-10：compact 双栏（资源 / 知识 各 3 条行式条目）
    Api.getResources({ page: 1, page_size: 3 }).then(function (res) {
      var box = root.querySelector('#home-res-rows')
      if (!box) return
      box.innerHTML = res.items.length ? res.items.map(function (r) {
        return '<a class="row-item" href="#/resource/' + Ui.esc(r.id) + '">' +
          '<span class="px-tag px-tag--muted">' + Ui.esc(window.PXUi.RESOURCE_KIND[r.kind] || r.kind) + '</span>' +
          '<span class="ri-title">' + Ui.esc(r.title) + '</span>' +
          '<span class="ri-date">' + Ui.fmtDate(r.updated_at) + '</span></a>'
      }).join('') : Ui.emptyState('箱子里还没有东西。')
      window.PXSprites.hydrate(box)
    }).catch(function () {
      var box = root.querySelector('#home-res-rows')
      if (box) { box.innerHTML = Ui.emptyState('资源加载失败。'); window.PXSprites.hydrate(box) }
    })

    Api.getKnowledgeArticles({ page: 1, page_size: 3 }).then(function (res) {
      var box = root.querySelector('#home-art-rows')
      if (!box) return
      box.innerHTML = res.items.length ? res.items.map(function (a) {
        return '<a class="row-item" href="#/article/' + Ui.esc(a.id) + '">' +
          '<span class="px-tag">' + Ui.esc(a.category) + '</span>' +
          '<span class="ri-title">' + Ui.esc(a.title) + '</span>' +
          '<span class="ri-date">' + Ui.fmtDate(a.updated_at) + '</span></a>'
      }).join('') : Ui.emptyState('书架上这个位置还空着。')
      window.PXSprites.hydrate(box)
    }).catch(function () {
      var box = root.querySelector('#home-art-rows')
      if (box) { box.innerHTML = Ui.emptyState('知识库加载失败。'); window.PXSprites.hydrate(box) }
    })

    // 活跃项目（2 条）
    Api.getProjects({ status: 'active', page: 1, page_size: 2 }).then(function (res) {
      var box = root.querySelector('#home-projects')
      if (!box) return
      box.innerHTML = res.items.length
        ? res.items.slice(0, 2).map(Ui.projectCard).join('')
        : Ui.emptyState('暂时没有进行中的项目。')
      window.PXSprites.hydrate(box)
    }).catch(function () {
      var box = root.querySelector('#home-projects')
      if (box) { box.innerHTML = Ui.errorState('项目加载失败。'); window.PXSprites.hydrate(box) }
    })

    return function cleanup() { if (stopPano) stopPano.stop() }
  }

  window.PXViews = window.PXViews || {}
  window.PXViews.home = render
})()
