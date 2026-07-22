<script setup lang="ts">
import { reactive } from 'vue'
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
  ElMessage.success('基础设置已成功保存。')
}

function saveSmtpSettings() {
  ElMessage.success('SMTP 邮箱通知机制已配置成功。')
}
</script>

<template>
  <section class="admin-page-heading">
    <div>
      <h2>系统设置</h2>
      <p>管理公开门户的基础属性以及新申请的邮件通知设置。</p>
    </div>
  </section>

  <section class="settings-layout">
    <div class="settings-main-column">
      <article class="admin-panel">
        <div class="panel-heading">
          <div>
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
            <el-switch v-model="form.customPortal" active-text="允许开放自定义 API 访问" />
          </el-form-item>
          <el-button type="primary" round @click="saveGeneralSettings">保存基础设置</el-button>
        </el-form>
      </article>

      <article class="admin-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>新申请邮件通知设置</h2>
          </div>
        </div>
        <el-form :model="smtp" label-position="top">
          <div class="form-grid">
            <el-form-item label="SMTP 服务器地址">
              <el-input v-model="smtp.host" placeholder="smtp.exmail.qq.com" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input v-model.number="smtp.port" placeholder="465" />
            </el-form-item>
          </div>

          <div class="form-grid">
            <el-form-item label="发件人邮箱">
              <el-input v-model="smtp.sender_email" placeholder="noreply@qutcraft.com" />
            </el-form-item>
            <el-form-item label="管理员接收邮箱">
              <el-input v-model="smtp.recipient_email" placeholder="admin@qutcraft.com" />
            </el-form-item>
          </div>

          <el-form-item label="SMTP 授权码 / 密码">
            <el-input v-model="smtp.auth_code" type="password" show-password placeholder="输入授权码" />
          </el-form-item>

          <el-form-item>
            <el-switch v-model="smtp.enable_notification" active-text="当有新玩家申请加入时向管理员发送邮件通知" />
          </el-form-item>

          <el-button type="primary" round @click="saveSmtpSettings">保存邮件通知配置</el-button>
        </el-form>
      </article>
    </div>

    <aside class="admin-panel settings-note">
      <h2>邮件提醒说明</h2>
      <p>当有新玩家提交加入申请或白名单申请时，系统将自动汇总其信息并通过邮件投递给管理员，便于第一时间审批。</p>
    </aside>
  </section>
</template>
