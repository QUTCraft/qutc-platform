/* ============================================================================
 * views-apply.js — 申请向导（全流程重设计）
 * 叙事线：启程(选通道) → 创建角色(填资料) → 书写故事(申请说明) → 合成申请(提交)
 * 进度 = XP 经验条（Lv.0 → Lv.3）；提交成功 = 成就弹窗 + 物品提示框凭证 + 升级。
 * 字段与 ApplicationPayload 完全对齐：
 *   whitelist 通道：name / class_name / game_id / qq_number / email 必填
 *   membership 通道：name / email 必填，note 可选
 * ========================================================================== */
(function () {
  'use strict'

  var STEP_TITLES = ['启程', '创建角色', '书写故事', '合成申请']

  function render(root) {
    var Ui = window.PXUi

    // 向导状态机
    var state = {
      step: 0,
      type: 'whitelist',
      values: { class_name: '', name: '', game_id: '', qq_number: '', email: '', note: '' },
      submitting: false,
      done: false,
      result: null,
    }

    root.innerHTML =
      '<section class="wizard container">' +
        '<div class="wizard-top">' +
          '<h1 class="wizard-title">加入协会</h1>' +
          '<div class="xp">' +
            '<span class="xp-level" id="xp-level">Lv.0</span>' +
            '<div class="xp-track"><div class="xp-fill" id="xp-fill"></div></div>' +
          '</div>' +
        '</div>' +
        '<div id="wz-body"></div>' +
      '</section>'

    var body = root.querySelector('#wz-body')

    function setXp(step) {
      var lv = root.querySelector('#xp-level')
      var fill = root.querySelector('#xp-fill')
      if (!lv || !fill) return
      var target = 'Lv.' + step
      if (lv.textContent !== target) {
        lv.textContent = target
        lv.classList.remove('bump')
        void lv.offsetWidth // 重触发动画
        lv.classList.add('bump')
      }
      fill.style.width = Math.round((step / 3) * 100) + '%'
    }

    // ---------------- Step 0 · 启程 ----------------
    function renderIntro() {
      body.innerHTML =
        '<div class="wz-step">' +
          '<div class="px-panel" style="text-align:center;padding:30px 22px 26px">' +
            '<h2 style="letter-spacing:4px">新的旅程</h2>' +
            '<p style="color:var(--px-text-2);max-width:560px;margin:0 auto">' +
              '方块不言语，像素自成诗。选择一条通道，告诉我们在哪儿能找到你。' +
              '审批由管理员人工完成，通常三个工作日内有结果。</p>' +
          '</div>' +
          '<div class="type-cards">' +
            '<button class="type-card' + (state.type === 'whitelist' ? ' selected' : '') + '" type="button" data-type="whitelist">' +
              Ui.spriteHtml('grass', 5) +
              '<h3>服务器白名单</h3>' +
              '<p>进入协会协作服务器，参与「方块重现校园」等大型工程。需要正版或统一游戏 ID。</p>' +
            '</button>' +
            '<button class="type-card' + (state.type === 'membership' ? ' selected' : '') + '" type="button" data-type="membership">' +
              Ui.spriteHtml('heart', 5) +
              '<h3>社团成员</h3>' +
              '<p>像素美术、Game Jam、线下活动与知识库共建。不玩服务器也欢迎。</p>' +
            '</button>' +
          '</div>' +
          '<div style="text-align:center">' +
            '<button class="px-btn px-btn--xp" type="button" id="wz-next-0">开始创建</button>' +
          '</div>' +
        '</div>'

      body.querySelectorAll('.type-card').forEach(function (card) {
        card.addEventListener('click', function () {
          state.type = card.getAttribute('data-type')
          body.querySelectorAll('.type-card').forEach(function (c) { c.classList.remove('selected') })
          card.classList.add('selected')
        })
      })
      body.querySelector('#wz-next-0').addEventListener('click', function () { go(1) })
      window.PXSprites.hydrate(body)
    }

    // ---------------- Step 1 · 创建角色 ----------------
    function fieldHtml(key, label, opts) {
      opts = opts || {}
      return (
        '<div class="field" data-field="' + key + '">' +
          '<label for="f-' + key + '">' + label + (opts.required ? '<span class="req">*</span>' : '') + '</label>' +
          '<input class="px-input" id="f-' + key + '" type="' + (opts.type || 'text') + '" placeholder="' + Ui.esc(opts.placeholder || '') + '" value="' + Ui.esc(state.values[key]) + '"' + (opts.maxlength ? ' maxlength="' + opts.maxlength + '"' : '') + '>' +
          (opts.hint ? '<div class="hint">' + opts.hint + '</div>' : '') +
          '<div class="err">' + (opts.errText || '这一项还没填好') + '</div>' +
        '</div>'
      )
    }

    function renderForm() {
      var isWl = state.type === 'whitelist'
      var fields = ''
      if (isWl) fields += fieldHtml('class_name', '班级 / 专业', { required: true, placeholder: '例如：计算机231', errText: '请输入班级/专业' })
      fields += fieldHtml('name', '姓名', { required: true, placeholder: '真实姓名', errText: '请输入姓名' })
      if (isWl) {
        fields += fieldHtml('game_id', 'Minecraft 游戏 ID', { required: true, placeholder: 'Java 版正版或统一 ID', errText: '请输入游戏 ID' })
        fields += fieldHtml('qq_number', 'QQ 号码', { required: true, placeholder: '便于拉入内部交流群', type: 'text', errText: '请输入 QQ 号码' })
      }
      fields += fieldHtml('email', '电子邮箱', { required: true, type: 'email', placeholder: '用于接收审批结果', errText: '请输入有效邮箱' })

      body.innerHTML =
        '<div class="wz-step">' +
          '<h2 style="letter-spacing:3px">创建角色</h2>' +
          '<div class="form-split">' +
            '<div class="px-panel"><form id="wz-form" novalidate>' + fields + '</form></div>' +
            '<aside class="px-panel char-card" aria-label="角色卡预览">' +
              '<canvas class="avatar" id="cc-avatar" width="8" height="8"></canvas>' +
              '<div class="cc-name" id="cc-name">未命名角色</div>' +
              '<div class="cc-sub" id="cc-sub">' + (isWl ? 'WHITELIST // APPLICANT' : 'MEMBER // APPLICANT') + '</div>' +
              '<div style="margin-top:10px">' +
                (isWl ? '<div class="cc-row"><span>班级</span><strong id="cc-class">—</strong></div>' +
                        '<div class="cc-row"><span>游戏 ID</span><strong id="cc-game">—</strong></div>' +
                        '<div class="cc-row"><span>QQ</span><strong id="cc-qq">—</strong></div>'
                      : '<div class="cc-row"><span>通道</span><strong>社团成员</strong></div>') +
                '<div class="cc-row"><span>邮箱</span><strong id="cc-email">—</strong></div>' +
              '</div>' +
            '</aside>' +
          '</div>' +
          '<div style="display:flex;gap:12px;margin-top:20px">' +
            '<button class="px-btn px-btn--ghost" type="button" id="wz-back-1">返回</button>' +
            '<button class="px-btn px-btn--primary" type="button" id="wz-next-1">继续</button>' +
          '</div>' +
        '</div>'

      // 实时预览：角色卡 + 像素头像（名字哈希生成，确定性）
      function drawAvatar() {
        var canvas = body.querySelector('#cc-avatar')
        var seedStr = (state.values.name || '') + '|' + (state.values.game_id || state.values.email || '')
        var seed = 0
        for (var i = 0; i < seedStr.length; i++) seed = (seed * 31 + seedStr.charCodeAt(i)) >>> 0
        if (!seedStr.trim()) seed = 20261009
        var ctx = canvas.getContext('2d')
        var theme = window.PXTheme.get().palette
        var colors = [theme.primary, theme.secondary, theme.tertiary, theme.onSurface]
        var bg = seed % 2 === 0 ? theme.surfaceContainerLow : theme.primaryContainer
        ctx.fillStyle = bg
        ctx.fillRect(0, 0, 8, 8)
        for (var y = 0; y < 8; y++) {
          for (var x = 0; x < 4; x++) {
            seed = (seed * 1103515245 + 12345) >>> 0
            if ((seed >> 16) % 3 === 0) {
              ctx.fillStyle = colors[(seed >> 20) % colors.length]
              ctx.fillRect(x, y, 1, 1)
              ctx.fillRect(7 - x, y, 1, 1) // 左右镜像，identicon 式
            }
          }
        }
      }

      function syncCard() {
        var set = function (id, v) { var n = body.querySelector(id); if (n) n.textContent = v || '—' }
        body.querySelector('#cc-name').textContent = state.values.name || '未命名角色'
        set('#cc-class', state.values.class_name)
        set('#cc-game', state.values.game_id)
        set('#cc-qq', state.values.qq_number)
        set('#cc-email', state.values.email)
        drawAvatar()
      }

      body.querySelectorAll('.px-input').forEach(function (input) {
        input.addEventListener('input', function () {
          var key = input.id.replace('f-', '')
          state.values[key] = input.value
          input.closest('.field').classList.remove('invalid')
          syncCard()
        })
      })
      syncCard()

      body.querySelector('#wz-back-1').addEventListener('click', function () { go(0) })
      body.querySelector('#wz-next-1').addEventListener('click', function () {
        // 校验：与平台 ApplyView 的必填规则一致
        var isWl2 = state.type === 'whitelist'
        var checks = { name: function (v) { return !!v.trim() }, email: function (v) { return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) } }
        if (isWl2) {
          checks.class_name = function (v) { return !!v.trim() }
          checks.game_id = function (v) { return !!v.trim() }
          checks.qq_number = function (v) { return /^\d{5,11}$/.test(v.trim()) }
        }
        var ok = true
        Object.keys(checks).forEach(function (key) {
          var field = body.querySelector('[data-field="' + key + '"]')
          var valid = checks[key](state.values[key] || '')
          if (field) field.classList.toggle('invalid', !valid)
          if (!valid) ok = false
        })
        if (ok) go(2)
      })
    }

    // ---------------- Step 2 · 书写故事 ----------------
    function renderNote() {
      body.innerHTML =
        '<div class="wz-step">' +
          '<h2 style="letter-spacing:3px">书写故事</h2>' +
          '<div class="px-panel">' +
            '<div class="field" data-field="note">' +
              '<label for="f-note">申请说明（可选）</label>' +
              '<textarea class="px-textarea" id="f-note" maxlength="500" placeholder="想参与的方向、正在练习的技能、做过的东西……管理员会认真读。">' + Ui.esc(state.values.note) + '</textarea>' +
              '<div class="hint">最多 500 字。不擅长写作也没关系，真诚比修辞重要。</div>' +
            '</div>' +
          '</div>' +
          '<div style="display:flex;gap:12px;margin-top:20px">' +
            '<button class="px-btn px-btn--ghost" type="button" id="wz-back-2">返回</button>' +
            '<button class="px-btn px-btn--primary" type="button" id="wz-next-2">前往合成台</button>' +
          '</div>' +
        '</div>'
      body.querySelector('#f-note').addEventListener('input', function (e) { state.values.note = e.target.value })
      body.querySelector('#wz-back-2').addEventListener('click', function () { go(1) })
      body.querySelector('#wz-next-2').addEventListener('click', function () { go(3) })
    }

    // ---------------- Step 3 · 合成申请 ----------------
    function renderCraft() {
      var v = state.values
      var isWl = state.type === 'whitelist'
      body.innerHTML =
        '<div class="wz-step">' +
          '<h2 style="letter-spacing:3px">合成申请</h2>' +
          '<div class="px-panel">' +
            '<div class="craft-wrap">' +
              '<div class="craft-grid" id="craft-grid">' +
                '<div class="craft-cell" data-slot="0"></div><div class="craft-cell" data-slot="1"></div><div class="craft-cell" data-slot="2"></div>' +
                '<div class="craft-cell" data-slot="3"></div><div class="craft-cell" data-slot="4"></div><div class="craft-cell" data-slot="5"></div>' +
                '<div class="craft-cell" data-slot="6"></div><div class="craft-cell" data-slot="7"></div><div class="craft-cell" data-slot="8"></div>' +
              '</div>' +
              '<div class="craft-arrow">➜</div>' +
              '<div class="craft-cell craft-result" id="craft-result"><div class="glint"></div></div>' +
            '</div>' +
            '<p class="craft-note">原料：角色卡 ×1、书写的故事 ×1、意向通道 ×1 —— 合成《入会申请书》</p>' +
            '<div class="passport" style="margin:0 auto 18px">' +
              '<div class="pp-title">' + (isWl ? '服务器白名单申请' : '社团成员申请') + '</div>' +
              '<div class="pp-id">QUTCRAFT // DRAFT</div>' +
              '<div style="height:8px"></div>' +
              '<div class="pp-row"><span>姓名</span><strong>' + Ui.esc(v.name) + '</strong></div>' +
              (isWl ? '<div class="pp-row"><span>班级 / 专业</span><strong>' + Ui.esc(v.class_name) + '</strong></div>' +
                      '<div class="pp-row"><span>游戏 ID</span><strong>' + Ui.esc(v.game_id) + '</strong></div>' +
                      '<div class="pp-row"><span>QQ</span><strong>' + Ui.esc(v.qq_number) + '</strong></div>' : '') +
              '<div class="pp-row"><span>邮箱</span><strong>' + Ui.esc(v.email) + '</strong></div>' +
              (v.note ? '<div class="pp-row"><span>说明</span><strong>' + Ui.esc(v.note.slice(0, 60)) + (v.note.length > 60 ? '…' : '') + '</strong></div>' : '') +
            '</div>' +
            '<div style="text-align:center">' +
              '<button class="px-btn px-btn--xp" type="button" id="wz-submit"' + (state.submitting ? ' disabled' : '') + '>合成并提交</button>' +
            '</div>' +
          '</div>' +
          '<div style="margin-top:16px"><button class="px-btn px-btn--ghost" type="button" id="wz-back-3">返回修改</button></div>' +
        '</div>'

      // 原料依次飞入合成格（错峰 pop）
      var grid = body.querySelector('#craft-grid')
      var ingredients = ['paper', 'book', isWl ? 'grass' : 'heart']
      var slots = [1, 4, 7] // 中列三格
      ingredients.forEach(function (name, i) {
        setTimeout(function () {
          var cell = grid.querySelector('[data-slot="' + slots[i] + '"]')
          if (cell) {
            cell.appendChild(window.PXSprites.make(name, 4))
            cell.firstChild.style.animationDelay = '0ms'
          }
        }, 300 + i * 340)
      })
      setTimeout(function () {
        var result = body.querySelector('#craft-result')
        if (result) result.insertBefore(window.PXSprites.make('paper', 6), result.firstChild)
      }, 300 + ingredients.length * 340 + 260)

      body.querySelector('#wz-back-3').addEventListener('click', function () { go(2) })
      body.querySelector('#wz-submit').addEventListener('click', submit)
      window.PXSprites.hydrate(body)
    }

    // ---------------- 提交 ----------------
    function submit() {
      if (state.submitting) return
      state.submitting = true
      window.PXUi.showLoader('正在连接主城…')
      window.PXApi.submitApplication({
        type: state.type,
        class_name: state.values.class_name || undefined,
        name: state.values.name,
        game_id: state.values.game_id || undefined,
        qq_number: state.values.qq_number || undefined,
        email: state.values.email,
        note: state.values.note || undefined,
      }).then(function (res) {
        state.done = true
        state.result = res
        window.PXUi.hideLoader()
        renderSuccess()
        window.PXUi.showAchievement('递交申请', 'paper')
      }).catch(function () {
        window.PXUi.hideLoader()
        renderError()
      }).finally(function () {
        state.submitting = false
      })
    }

    function renderSuccess() {
      var v = state.values
      var isWl = state.type === 'whitelist'
      body.innerHTML =
        '<div class="wz-step success-block">' +
          '<h2>出生点已记录</h2>' +
          '<p class="sub">申请已进入审批队列，管理员上线后会第一时间处理。请保持邮箱畅通。</p>' +
          '<div class="passport">' +
            '<div class="pp-title">' + (isWl ? '白名单验证存根' : '成员申请存根') + '</div>' +
            '<div class="pp-id">' + Ui.esc((state.result && state.result.id) || '#QUTCRAFT-APPLICATION') + '</div>' +
            '<div style="height:8px"></div>' +
            '<div class="pp-row"><span>申请人</span><strong>' + Ui.esc(v.name) + (isWl && v.class_name ? '（' + Ui.esc(v.class_name) + '）' : '') + '</strong></div>' +
            (isWl ? '<div class="pp-row"><span>游戏 ID</span><strong>' + Ui.esc(v.game_id) + '</strong></div>' : '') +
            '<div class="pp-row"><span>状态</span><strong class="pp-status">● 待管理员审批</strong></div>' +
          '</div>' +
          '<div class="success-actions">' +
            '<a class="px-btn px-btn--primary" href="#/projects">看看公开项目</a>' +
            '<a class="px-btn px-btn--ghost" href="#/">返回主世界</a>' +
          '</div>' +
        '</div>'
      setXp(3)
      var lv = root.querySelector('#xp-level')
      if (lv) { lv.classList.remove('bump'); void lv.offsetWidth; lv.classList.add('bump') }
    }

    function renderError() {
      body.innerHTML =
        '<div class="wz-step">' +
          '<div class="err-panel px-panel">' +
            '<div class="es-art">' + Ui.spriteHtml('redstone', 6) + '</div>' +
            '<h2>连接中断</h2>' +
            '<p style="color:var(--px-text-2)">申请没有送出去。你的填写内容还在，检查网络后再试一次。</p>' +
            '<div class="success-actions">' +
              '<button class="px-btn px-btn--primary" type="button" id="wz-retry">重试提交</button>' +
              '<button class="px-btn px-btn--ghost" type="button" id="wz-edit">返回修改</button>' +
            '</div>' +
          '</div>' +
        '</div>'
      window.PXSprites.hydrate(body)
      body.querySelector('#wz-retry').addEventListener('click', function () { renderCraft(); submit() })
      body.querySelector('#wz-edit').addEventListener('click', function () { go(3) })
    }

    // ---------------- 步骤调度 ----------------
    function go(step) {
      state.step = step
      setXp(step)
      if (step === 0) renderIntro()
      else if (step === 1) renderForm()
      else if (step === 2) renderNote()
      else renderCraft()
      window.scrollTo(0, 0)
    }

    go(0)
  }

  window.PXViews = window.PXViews || {}
  window.PXViews.apply = render
})()
