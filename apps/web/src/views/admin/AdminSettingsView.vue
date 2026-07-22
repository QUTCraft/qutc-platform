<script setup lang="ts">
import { reactive } from 'vue'
import { Message, MessageBox } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { SmtpSettings } from '@/api/types'

const form = reactive({
  name: 'QUTCraft Commons',
  slug: 'qutcraft',
  publicPortal: true,
  customPortal: false,
})

const smtp = reactive<SmtpSettings>({
  host: 'smtp.qutcraft.com',
  port: 465,
  sender_email: 'whitelist-bot@qutcraft.com',
  recipient_email: 'admin-whitelist@qutcraft.local',
  auth_code: 'demo-smtp-secret-key',
  enable_notification: true,
})

function saveGeneralSettings() {
  ElMessage.success('组织公开门户设置已成功保存。')
}

function saveSmtpSettings() {
  ElMessage.success('SMTP 邮箱白名单通知接口已配置成功！新加入申请将实时推送给管理员邮箱。')
}
</script>

<template>
  <section class="admin-page-heading">
    <div>
      <p class="eyebrow">ORGANIZATION & SYSTEM CONFIGURATION</p>
      <h2>组织与服务设置</h2>
      <p>配置门户公开属性与白名单提交后台 SMTP 邮箱转发规则。</p>
    </div>
  </section>

  <section class="settings-layout">
    <div class="settings-main-column">
      <article class="admin-panel">
        <div class="panel-heading">
          <div>
            <p class="eyebrow">PUBLIC PORTAL</p>
            <h2>公开门户基础配置</h2>
          </div>
        </div>
        <el-form :model="form" label-position="top">
          <el-form-item label="组织名称">
            <el-input v-model="form.name" />
          </el-form-item>
          <el-form-item label="组织标识">
            <el-input v-model="form.slug" disabled />
          </el-form-item>
          <el-form-item>
            <el-switch v-model="form.publicPortal" active-text="启用公开门户" />
          </el-form-item>
          <el-form-item>
            <el-switch v-model="form.customPortal" active-text="允许第三方自定义 Portal API 接入" />
          </el-form-item>
          <el-button type="primary" round @click="saveGeneralSettings">保存基础设置</el-button>
        </el-form>
      </article>

      <article class="admin-panel" style="margin-top: 16px;">
        <div class="panel-heading">
          <div>
            <p class="eyebrow">WHITELIST EMAIL NOTIFICATION</p>
            <h2>SMTP 申请接收与邮箱通知接口</h2>
          </div>
        </div>
        <el-form :model="smtp" label-position="top">
          <div class="form-grid">
            <el-form-item label="SMTP 服务器地址 (Host)">
              <el-input v-model="smtp.host" placeholder="smtp.exmail.qq.com" />
            </el-form-item>
            <el-form-item label="端口 (Port)">
              <el-input v-model.number="smtp.port" placeholder="465" />
            </el-form-item>
          </div>

          <div class="form-grid">
            <el-form-item label="发件机器人邮箱 (Sender)">
              <el-input v-model="smtp.sender_email" placeholder="noreply@qutcraft.com" />
            </el-form-item>
            <el-form-item label="管理员接收邮箱 (Recipient)">
              <el-input v-model="smtp.recipient_email" placeholder="admin@qutcraft.com" />
            </el-form-item>
          </div>

          <el-form-item label="SMTP 授权码 / 密码">
            <el-input v-model="smtp.auth_code" type="password" show-password placeholder="输入邮箱授权码" />
          </el-form-item>

          <el-form-item>
            <el-switch v-model="smtp.enable_notification" active-text="收到新玩家申请时通过 SMTP 实时投递至管理员邮箱" />
          </el-form-item>

          <el-button type="primary" round @click="saveSmtpSettings">保存 SMTP 配置</el-button>
        </el-form>
      </article>
    </div>

    <aside class="admin-panel settings-note">
      <p class="eyebrow">SYSTEM ARCHITECTURE</p>
      <h2>SMTP 邮件转发机制</h2>
      <p>公开申请端调用：<code>POST /api/v1/portal/apply</code></p>
      <p>后端消费队列后，使用此处的 SMTP 配置直接将玩家的【班级、姓名、游戏ID、QQ号、邮箱】格式化输出并邮件提醒管理员。</p>
      <small>注意：授权码等敏感凭证只保存在后端受控环境变量中，不会经由公开 API 泄露。</small>
    </aside>
  </section>
</template>
