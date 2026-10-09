/* ============================================================================
 * views-lists.js — 动态 / 项目 / 资源 / 资源详情 / 知识库 / 文章详情
 * v0.4：全列表接分页器（§2b-1）、资源搜索（§2b-2）、项目 chips 计数（§2b-4）、
 *       资源详情页（§2b-3）；列表统一 加载中 / 空态 / 错误重试 三态。
 * ========================================================================== */
(function () {
  'use strict'

  var Ui = function () { return window.PXUi }

  // 通用列表页：筛选 chips（可带计数）+ 可选搜索框 + 卡片网格 + 分页器
  // opt: { title, en, cols, categories:[{value,label,count?}], search?, searchPlaceholder?,
  //        pageSize, empty, fetch(state)->Promise<Page> } state = { cat, q, page }
  function listPage(root, opt) {
    var state = { cat: '', q: '', page: 1 }

    var chipsHtml = '<div class="chips">' +
      '<button class="chip active" data-cat="" type="button">全部</button>' +
      opt.categories.map(function (c) {
        var label = c.label + (c.count != null ? ' ' + c.count : '')
        return '<button class="chip" data-cat="' + Ui().esc(c.value) + '" type="button">' + Ui().esc(label) + '</button>'
      }).join('') + '</div>'

    var searchHtml = opt.search
      ? '<div class="list-search"><input class="px-input" id="list-q" type="search" placeholder="' +
        Ui().esc(opt.searchPlaceholder || '搜索关键词…') + '" aria-label="搜索"></div>'
      : ''

    root.innerHTML =
      '<section class="section container">' +
        '<div class="section-head"><h2>' + opt.title + '</h2><span class="en">// ' + opt.en + '</span></div>' +
        searchHtml + chipsHtml +
        '<div class="grid ' + (opt.cols === 2 ? 'grid--2' : 'grid--3') + '" id="list-grid"></div>' +
        '<div id="list-pager"></div>' +
      '</section>'

    var grid = root.querySelector('#list-grid')
    var pagerBox = root.querySelector('#list-pager')
    var searchTimer = 0

    function load() {
      grid.innerHTML = '<div class="px-panel" style="grid-column:1/-1;color:var(--px-text-2)">正在翻找箱子…</div>'
      pagerBox.innerHTML = ''
      opt.fetch(state).then(function (res) {
        grid.innerHTML = res.items.length ? res.items.map(opt.card).join('') : Ui().emptyState(opt.empty)
        window.PXSprites.hydrate(grid)
        if (res.total > res.page_size) {
          pagerBox.appendChild(Ui().pagerEl(res.page, res.page_size, res.total, function (n) {
            state.page = n
            load()
          }))
        }
      }).catch(function () {
        grid.innerHTML = '<div style="grid-column:1/-1">' + Ui().errorState('列表加载失败。') + '</div>'
        window.PXSprites.hydrate(grid)
        var retry = grid.querySelector('[data-retry]')
        if (retry) retry.addEventListener('click', function () { load() })
      })
    }

    root.querySelectorAll('.chip').forEach(function (chip) {
      chip.addEventListener('click', function () {
        root.querySelectorAll('.chip').forEach(function (c) { c.classList.remove('active') })
        chip.classList.add('active')
        state.cat = chip.getAttribute('data-cat')
        state.page = 1
        load()
      })
    })

    var qInput = root.querySelector('#list-q')
    if (qInput) {
      qInput.addEventListener('input', function () {
        clearTimeout(searchTimer)
        searchTimer = setTimeout(function () {
          state.q = qInput.value.trim()
          state.page = 1
          load()
        }, 250)
      })
    }

    load('')
  }

  // ---- 动态列表（页大小 6） ----
  function posts(root) {
    window.PXApi.getPosts({ page: 1, page_size: 100 }).then(function (res) {
      var cats = []
      res.items.forEach(function (p) {
        if (p.category && cats.indexOf(p.category) < 0) cats.push(p.category)
      })
      listPage(root, {
        title: '协会动态', en: 'POSTS', cols: 3, pageSize: 6,
        categories: cats.map(function (c) { return { value: c, label: c } }),
        fetch: function (s) {
          return window.PXApi.getPosts({
            category: s.cat || undefined, q: s.q || undefined,
            page: s.page, page_size: 6,
          })
        },
        card: function (p) { return Ui().postCard(p) },
        empty: '这个分类下还没有动态。',
      })
    }).catch(function () {
      root.innerHTML = '<section class="section container">' + Ui().errorState('动态列表加载失败。') + '</section>'
      window.PXSprites.hydrate(root)
      var retry = root.querySelector('[data-retry]')
      if (retry) retry.addEventListener('click', function () { posts(root) })
    })
  }

  // ---- 动态详情 ----
  function postDetail(root, id) {
    root.innerHTML = '<section class="section container article" id="art"><div class="px-panel">正在生成地形…</div></section>'
    window.PXApi.getContentDetail(id).then(function (c) {
      var box = root.querySelector('#art')
      box.innerHTML =
        '<div class="article-head">' +
          '<a href="#/posts" style="font-weight:700">← 返回动态</a>' +
          '<h1>' + Ui().esc(c.title) + '</h1>' +
          '<div class="card-meta"><span class="px-tag px-tag--primary">' + Ui().esc(c.category) + '</span>' +
            (c.members_only ? Ui().lockTag('本篇动态含成员内容&#10;完整版仅协会成员可见') : '') +
            '<span>' + Ui().fmtDate(c.published_at) + '</span>' +
            '<span class="zh">' + (c.reading_minutes || 1) + ' 分钟读完</span></div>' +
        '</div>' +
        '<div class="px-panel article-body markdown-content">' + Ui().mdLite(c.body) + '</div>' +
        (c.download_url ? '<p style="margin-top:16px"><a class="px-btn px-btn--primary" href="' + Ui().esc(c.download_url) + '">下载附件</a></p>' : '')
      window.PXSprites.hydrate(box)
    }).catch(function () {
      root.querySelector('#art').innerHTML = Ui().errorState('这篇内容走丢了（404 或网络故障）。')
      window.PXSprites.hydrate(root)
      var retry = root.querySelector('[data-retry]')
      if (retry) retry.addEventListener('click', function () { postDetail(root, id) })
    })
  }

  // ---- 项目（chips 带计数 · 页大小 4） ----
  function projects(root) {
    Promise.all([
      window.PXApi.getProjects({ status: 'active', page: 1, page_size: 1 }),
      window.PXApi.getProjects({ status: 'research', page: 1, page_size: 1 }),
      window.PXApi.getProjects({ status: 'completed', page: 1, page_size: 1 }),
    ]).then(function (pages) {
      var counts = { active: pages[0].total, research: pages[1].total, completed: pages[2].total }
      listPage(root, {
        title: '公开项目', en: 'PROJECTS', cols: 2, pageSize: 4,
        categories: [
          { value: 'active', label: '进行中', count: counts.active },
          { value: 'research', label: '调研中', count: counts.research },
          { value: 'completed', label: '已完成', count: counts.completed },
        ],
        fetch: function (s) {
          return window.PXApi.getProjects({
            status: s.cat || undefined, page: s.page, page_size: 4,
          })
        },
        card: function (p) { return Ui().projectCard(p) },
        empty: '这里没有项目，连地基都没打。',
      })
    }).catch(function () {
      root.innerHTML = '<section class="section container">' + Ui().errorState('项目列表加载失败。') + '</section>'
      window.PXSprites.hydrate(root)
      var retry = root.querySelector('[data-retry]')
      if (retry) retry.addEventListener('click', function () { projects(root) })
    })
  }

  // ---- 资源（搜索 + kind chips + 分页共存 · 页大小 4） ----
  function resources(root) {
    var kinds = window.PXUi.RESOURCE_KIND
    listPage(root, {
      title: '资源中心', en: 'RESOURCES', cols: 2, pageSize: 4,
      search: true, searchPlaceholder: '搜索资源标题或描述…',
      categories: Object.keys(kinds).map(function (k) { return { value: k, label: kinds[k] } }),
      fetch: function (s) {
        return window.PXApi.getResources({
          kind: s.cat || undefined, q: s.q || undefined,
          page: s.page, page_size: 4,
        })
      },
      card: function (r) { return Ui().resourceCard(r) },
      empty: '箱子里没有符合条件的东西。',
    })
  }

  // ---- 资源详情（§2b-3：#/resource/:id） ----
  function resourceDetail(root, id) {
    root.innerHTML = '<section class="section container article" id="res"><div class="px-panel">正在翻找箱子…</div></section>'
    window.PXApi.getContentDetail(id).then(function (c) {
      var locked = !!c.members_only
      var asset = c.asset || {}
      var download = c.download_url || ''
      var box = root.querySelector('#res')
      box.innerHTML =
        '<div class="article-head">' +
          '<a href="#/resources" style="font-weight:700">← 返回资源中心</a>' +
          '<h1>' + Ui().esc(c.title) + '</h1>' +
          '<div class="card-meta">' +
            '<span class="px-tag">' + Ui().esc(window.PXUi.RESOURCE_KIND[c.category] || c.category) + '</span>' +
            (locked ? Ui().lockTag('该资源仅协会成员可见&#10;先提交申请加入我们吧') : '') +
            '<span>' + Ui().fmtSize(asset.size_bytes) + '</span>' +
            '<span>' + Ui().fmtDate(c.updated_at) + '</span>' +
          '</div>' +
        '</div>' +
        '<div class="px-panel article-body markdown-content">' + Ui().mdLite(c.body) + '</div>' +
        '<div class="px-panel attach-card">' +
          '<span class="ico" data-sprite="chest" data-px="5"></span>' +
          '<div class="ac-info">' +
            '<div class="ac-name">' + Ui().esc(asset.original_name || c.title) + '</div>' +
            '<div class="ac-meta">' + Ui().esc(asset.mime_type || 'application/octet-stream') + ' · ' + Ui().fmtSize(asset.size_bytes) + '</div>' +
          '</div>' +
          (locked
            ? '<span class="px-btn px-btn--sm px-btn--ghost" data-tip="成员资源，登录成员账号后由管理员发放">上锁了</span>'
            : (download
              ? '<a class="px-btn px-btn--sm px-btn--primary" href="' + Ui().esc(download) + '">下载</a>'
              : '<span class="px-btn px-btn--sm px-btn--ghost" data-tip="暂无下载地址">待补给</span>')) +
        '</div>'
      window.PXSprites.hydrate(box)
    }).catch(function () {
      root.querySelector('#res').innerHTML = Ui().errorState('这个资源走丢了（404 或网络故障）。')
      window.PXSprites.hydrate(root)
      var retry = root.querySelector('[data-retry]')
      if (retry) retry.addEventListener('click', function () { resourceDetail(root, id) })
    })
  }

  // ---- 知识库（目录 + 文章两级，文章区接分页 · 页大小 6） ----
  function knowledge(root) {
    var artState = { dir: '', page: 1 }

    root.innerHTML =
      '<section class="section container">' +
        '<div class="section-head"><h2>知识库</h2><span class="en">// KNOWLEDGE</span></div>' +
        '<div class="grid grid--3" id="dir-grid" style="margin-bottom:26px"></div>' +
        '<div class="chips" id="art-chips">' +
          '<button class="chip active" data-dir="" type="button">全部文章</button>' +
        '</div>' +
        '<div class="grid grid--3" id="art-grid"></div>' +
        '<div id="art-pager"></div>' +
      '</section>'

    var dirGrid = root.querySelector('#dir-grid')
    var chips = root.querySelector('#art-chips')
    var artGrid = root.querySelector('#art-grid')
    var artPager = root.querySelector('#art-pager')

    window.PXApi.getKnowledgeDirectories().then(function (res) {
      dirGrid.innerHTML = res.items.map(function (d) {
        return (
          '<div class="card">' +
            '<div class="card-head">' + Ui().spriteHtml('book', 3) + '<h3 class="card-title">' + Ui().esc(d.name) + '</h3></div>' +
            '<p class="card-excerpt">' + Ui().esc(d.description) + '</p>' +
            '<div class="card-meta"><span class="zh">' + d.article_count + ' 篇</span></div>' +
          '</div>'
        )
      }).join('')
      window.PXSprites.hydrate(dirGrid)
      res.items.forEach(function (d) {
        var b = Ui().el('<button class="chip" data-dir="' + Ui().esc(d.name) + '" type="button">' + Ui().esc(d.name) + '</button>')
        b.addEventListener('click', function () { selectDir(b) })
        chips.appendChild(b)
      })
    }).catch(function () { dirGrid.innerHTML = '' })

    function selectDir(btn) {
      chips.querySelectorAll('.chip').forEach(function (c) { c.classList.remove('active') })
      btn.classList.add('active')
      artState.dir = btn.getAttribute('data-dir')
      artState.page = 1
      loadArticles()
    }
    chips.querySelector('.chip').addEventListener('click', function () { selectDir(chips.querySelector('.chip')) })

    function loadArticles() {
      artGrid.innerHTML = '<div class="px-panel" style="grid-column:1/-1;color:var(--px-text-2)">正在翻阅书架…</div>'
      artPager.innerHTML = ''
      window.PXApi.getKnowledgeArticles({
        category: artState.dir || undefined, page: artState.page, page_size: 6,
      }).then(function (res) {
        artGrid.innerHTML = res.items.length ? res.items.map(function (a) { return Ui().knowledgeCard(a) }).join('') : Ui().emptyState('书架上这个位置还空着。')
        window.PXSprites.hydrate(artGrid)
        if (res.total > res.page_size) {
          artPager.appendChild(Ui().pagerEl(res.page, res.page_size, res.total, function (n) {
            artState.page = n
            loadArticles()
          }))
        }
      }).catch(function () {
        artGrid.innerHTML = '<div style="grid-column:1/-1">' + Ui().errorState('文章列表加载失败。') + '</div>'
        window.PXSprites.hydrate(artGrid)
        var retry = artGrid.querySelector('[data-retry]')
        if (retry) retry.addEventListener('click', function () { loadArticles() })
      })
    }
    loadArticles()
  }

  // ---- 知识文章详情（a-* 走 getContentDetail，失败可重试） ----
  function articleDetail(root, id) {
    root.innerHTML = '<section class="section container article" id="art"><div class="px-panel">正在生成地形…</div></section>'
    window.PXApi.getContentDetail(id).then(function (a) {
      var box = root.querySelector('#art')
      box.innerHTML =
        '<div class="article-head">' +
          '<a href="#/knowledge" style="font-weight:700">← 返回知识库</a>' +
          '<h1>' + Ui().esc(a.title) + '</h1>' +
          '<div class="card-meta"><span class="px-tag">' + Ui().esc(a.category) + '</span>' +
            '<span>' + Ui().fmtDate(a.updated_at) + '</span>' +
            '<span class="zh">' + (a.reading_minutes || 1) + ' 分钟读完</span></div>' +
        '</div>' +
        '<div class="px-panel article-body markdown-content">' + Ui().mdLite(a.body) + '</div>'
      window.PXSprites.hydrate(box)
    }).catch(function () {
      root.querySelector('#art').innerHTML = Ui().errorState('这篇文章不存在或已下线。')
      window.PXSprites.hydrate(root)
      var retry = root.querySelector('[data-retry]')
      if (retry) retry.addEventListener('click', function () { articleDetail(root, id) })
    })
  }

  window.PXViews = window.PXViews || {}
  Object.assign(window.PXViews, {
    posts: posts,
    postDetail: postDetail,
    projects: projects,
    resources: resources,
    resourceDetail: resourceDetail,
    knowledge: knowledge,
    articleDetail: articleDetail,
  })
})()
