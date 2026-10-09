/* ============================================================================
 * views-misc.js — 问题反馈（report）/ 404（“重生”）
 * 登录 / 注册由主应用 /login /register 承担，本包不提供 hash 演示页。
 * ========================================================================== */
(function () {
  'use strict'

  // ---- 反馈：bug / feature 双槽 + 表单，提交成功弹成就 ----
  function report(root) {
    var Ui = window.PXUi
    var state = { kind: 'bug', submitting: false }

    root.innerHTML =
      '<section class="section container" style="max-width:680px">' +
        '<div class="section-head"><h2>网页缺陷与建议</h2><span class="en">// FEEDBACK</span></div>' +
        '<div class="kind-cards">' +
          '<button class="type-card selected" type="button" data-kind="bug">' + Ui.spriteHtml('redstone', 5) +
            '<h3>缺陷报告</h3><p>页面错位、加载失败、内容错误……告诉我们哪里炸了。</p></button>' +
          '<button class="type-card" type="button" data-kind="feature">' + Ui.spriteHtml('xpstar', 5) +
            '<h3>功能建议</h3><p>希望门户多什么能力？写下来，我们会认真评估。</p></button>' +
        '</div>' +
        '<form class="px-panel" id="fb-form" novalidate>' +
          '<div class="field" data-field="title"><label for="fb-title">标题<span class="req">*</span></label>' +
            '<input class="px-input" id="fb-title" maxlength="80" placeholder="一句话说清楚">' +
            '<div class="err">请输入标题</div></div>' +
          '<div class="field" data-field="description"><label for="fb-desc">详细描述<span class="req">*</span></label>' +
            '<textarea class="px-textarea" id="fb-desc" maxlength="2000" placeholder="发生了什么？你原本期待什么？"></textarea>' +
            '<div class="err">请填写详细描述</div></div>' +
          '<div class="field" data-field="contact_name"><label for="fb-name">称呼<span class="req">*</span></label>' +
            '<input class="px-input" id="fb-name" maxlength="40" placeholder="怎么称呼你">' +
            '<div class="err">请填写称呼</div></div>' +
          '<div class="field" data-field="contact_email"><label for="fb-email">联系邮箱<span class="req">*</span></label>' +
            '<input class="px-input" id="fb-email" type="email" placeholder="用于回复处理结果">' +
            '<div class="err">请输入有效邮箱</div></div>' +
          '<button class="px-btn px-btn--primary" type="submit" id="fb-submit">投递反馈</button>' +
        '</form>' +
      '</section>'

    root.querySelectorAll('.type-card').forEach(function (card) {
      card.addEventListener('click', function () {
        state.kind = card.getAttribute('data-kind')
        root.querySelectorAll('.type-card').forEach(function (c) { c.classList.remove('selected') })
        card.classList.add('selected')
      })
    })

    root.querySelector('#fb-form').addEventListener('submit', function (e) {
      e.preventDefault()
      if (state.submitting) return
      var val = function (id) { return root.querySelector(id).value.trim() }
      var data = {
        kind: state.kind,
        title: val('#fb-title'),
        description: val('#fb-desc'),
        contact_name: val('#fb-name'),
        contact_email: val('#fb-email'),
        page_url: location.href,
      }
      var ok = true
      var mark = function (key, valid) {
        root.querySelector('[data-field="' + key + '"]').classList.toggle('invalid', !valid)
        if (!valid) ok = false
      }
      mark('title', !!data.title)
      mark('description', !!data.description)
      mark('contact_name', !!data.contact_name)
      mark('contact_email', /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(data.contact_email))
      if (!ok) return

      state.submitting = true
      Ui.showLoader('正在投递…')
      window.PXApi.submitFeedback(data).then(function () {
        Ui.hideLoader()
        Ui.showAchievement('反馈已送达', state.kind === 'bug' ? 'redstone' : 'xpstar')
        root.querySelector('#fb-form').reset()
      }).catch(function () {
        Ui.hideLoader()
        Ui.showAchievement('投递失败，请重试', 'redstone')
      }).finally(function () { state.submitting = false })
    })

    window.PXSprites.hydrate(root)
  }

  // ---- 404：你迷失了方向 → 重生 ----
  function notFound(root) {
    var Ui = window.PXUi
    root.innerHTML =
      '<section class="notfound container">' +
        '<div class="nf-art">' + Ui.spriteHtml('creeper', 9) + '</div>' +
        '<h1>你迷失了方向</h1>' +
        '<span class="en">// 404 · CHUNK NOT FOUND</span>' +
        '<p style="color:var(--px-text-2)">这个坐标什么都没有。也许是传送门建错了位置。</p>' +
        '<a class="px-btn px-btn--xp" href="#/">重生</a>' +
      '</section>'
    window.PXSprites.hydrate(root)
  }

  window.PXViews = window.PXViews || {}
  Object.assign(window.PXViews, { report: report, notFound: notFound })
})()
