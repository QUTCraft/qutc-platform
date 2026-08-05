<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Check, MagicStick, Promotion, Ticket } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { organizationSlug, portalApi } from '@/api/portal'
import type { ApplicationPayload } from '@/api/types'

const isQutcraftOrganization = organizationSlug === 'qutcraft'

const formRef = ref<FormInstance>()
const submitting = ref(false)
const isSubmitted = ref(false)
const isBlooming = ref(false)
const isEntering = ref(true)

const form = reactive<ApplicationPayload>({
  type: isQutcraftOrganization ? 'whitelist' : 'membership',
  class_name: '',
  name: '',
  game_id: '',
  qq_number: '',
  email: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入您的真实姓名', trigger: 'blur' }],
  email: [{ required: true, type: 'email', message: '请输入有效联系邮箱', trigger: 'blur' }],
}

if (isQutcraftOrganization) {
  rules.class_name = [{ required: true, message: '请输入您的班级/专业（如：计算机231）', trigger: 'blur' }]
  rules.game_id = [{ required: true, message: '请输入您的 Minecraft 游戏 ID', trigger: 'blur' }]
  rules.qq_number = [{ required: true, message: '请输入您的 QQ 号码', trigger: 'blur' }]
}

onMounted(() => {
  requestAnimationFrame(() => {
    setTimeout(() => {
      isEntering.value = false
    }, 1450)
  })
})

async function handleApply() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true

  try {
    await portalApi.submitApplication(form)
    isBlooming.value = true
    setTimeout(() => {
      isSubmitted.value = true
    }, 850)
    ElMessage.success('申请已提交，正在等待管理员审批。')
  } catch {
    ElMessage.error('申请提交失败，请稍后重试。')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div
    class="starlight-wrapper"
    :class="{
      'is-entering': isEntering,
      'is-blooming': isBlooming,
    }"
  >
    <!-- Radial Darkness Expansion Layer -->
    <div class="darkness-button-expand" />

    <!-- Ambient Cosmic Nebulae -->
    <div class="nebula-glow nebula-1" />
    <div class="nebula-glow nebula-2" />

    <!-- High-Density Starfield: Stars pop into existence ONE BY ONE sequentially & stay as background -->
    <div class="dense-starfield">
      <span
        v-for="i in 60"
        :key="i"
        class="twinkle-star"
        :style="{
          '--delay': `${i * 24}ms`,
          '--x': `${((i * 37) % 96) + 2}vw`,
          '--y': `${((i * 41) % 94) + 2}vh`,
          '--size': `${((i % 3) + 2)}px`,
        }"
      />
    </div>

    <!-- Background Starfield texture -->
    <div class="starfield" />

    <!-- Post-Submit Explosive Supernova Color Wave & Floating Cube Particles -->
    <div class="supernova-bloom-overlay">
      <div class="shockwave-circle circle-1" />
      <div class="shockwave-circle circle-2" />
      <div class="shockwave-circle circle-3" />
      <div class="supernova-beam beam-1" />
      <div class="supernova-beam beam-2" />

      <!-- Ascending 3D Minecraft Cube Particles ("赴方块之约") -->
      <div v-if="isBlooming" class="ascension-cubes">
        <span
          v-for="c in 16"
          :key="c"
          class="cube-particle"
          :style="{
            '--delay': `${c * 60}ms`,
            '--x': `${((c * 23) % 90) + 5}vw`,
            '--size': `${((c % 4) + 1) * 12}px`,
          }"
        />
      </div>
    </div>

    <section class="starlight-card">
      <RouterLink to="/" class="back-link">
        <el-icon><ArrowLeft /></el-icon> 返回门户首页
      </RouterLink>

      <div class="starlight-header">
        <div class="starlight-badge">
          <el-icon><MagicStick /></el-icon>
          <span>{{ isQutcraftOrganization ? 'QUTCRAFT // WHITELIST ACCESS' : 'ORGANIZATION // MEMBERSHIP ACCESS' }}</span>
        </div>
        <h1 class="starlight-title">{{ isQutcraftOrganization ? '踏星汉而至，赴方块之约' : '加入组织，共建公开项目' }}</h1>
        <p v-if="isQutcraftOrganization" class="starlight-welcome">
          青岛理工大学方块协会（QUTCraft）恭候多时。凡我校同仁，凭热忱与求知，皆可跨越次元壁垒。
          无论君精于宏伟雕琢、沉醉电路红石、擅长 Mod 开发，抑或钟情方寸探索，星河浩瀚，愿与君共铸奇迹。
        </p>
        <p v-else class="starlight-welcome">提交成员申请，向组织介绍自己和希望参与的方向。管理员会在审核后通过邮件联系你。</p>
      </div>

      <!-- Success Hologram Certificate Screen -->
      <div v-if="isSubmitted" class="starlight-success">
        <div class="success-icon-wrapper">
          <!-- Rotating Constellation Orbit Ring (Strictly Concentric with inset: 0 and inset: 8px) -->
          <div class="constellation-orbit">
            <span class="orbit-dot dot-1" />
            <span class="orbit-dot dot-2" />
            <span class="orbit-dot dot-3" />
            <span class="orbit-dot dot-4" />
          </div>

          <div class="check-badge">
            <el-icon><Check /></el-icon>
          </div>
        </div>

        <h2 class="blooming-title">{{ isQutcraftOrganization ? '✦ 已接入 QUTCraft 星河矩阵 ✦' : '✦ 申请已进入组织审批队列 ✦' }}</h2>
        <p class="success-subtitle">
          您的申请已生成并存入后台审批队列，请等待管理员完成审核。
        </p>

        <div class="passport-card">
          <div class="passport-header">
            <span><el-icon><Ticket /></el-icon> {{ isQutcraftOrganization ? 'QUTCraft 白名单验证存根' : '组织成员申请存根' }}</span>
            <small class="passport-id">#QUT-2026-APPLICATION</small>
          </div>
          <div class="passport-grid">
            <div><span>申请人</span><strong>{{ form.name }}<span v-if="isQutcraftOrganization"> ({{ form.class_name }})</span></strong></div>
            <div v-if="isQutcraftOrganization"><span>游戏 ID</span><strong>{{ form.game_id }}</strong></div>
            <div v-if="isQutcraftOrganization"><span>QQ 账号</span><strong>{{ form.qq_number }}</strong></div>
            <div v-else><span>申请类型</span><strong>成员加入</strong></div>
            <div><span>状态</span><strong class="tag-pending">● 待管理员审批 (Pending)</strong></div>
          </div>
        </div>

        <p class="success-tip">
          请保持 <strong>{{ form.email }}</strong> 邮箱畅通，审批结果会通过组织配置的通知渠道发送。
        </p>

        <div class="success-actions">
          <RouterLink to="/projects">
            <el-button type="primary" size="large" round>浏览公开项目</el-button>
          </RouterLink>
          <RouterLink to="/">
            <el-button size="large" round plain>返回首页</el-button>
          </RouterLink>
        </div>
      </div>

      <!-- Form State Screen -->
      <el-form
        v-else
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        size="large"
        class="starlight-form"
        @submit.prevent="handleApply"
      >
        <div class="form-grid">
          <el-form-item v-if="isQutcraftOrganization" label="班级 / 专业" prop="class_name">
            <el-input v-model="form.class_name" placeholder="例如：计算机231 / 建筑222" />
          </el-form-item>

          <el-form-item label="姓名" prop="name">
            <el-input v-model="form.name" placeholder="请输入您的真实姓名" />
          </el-form-item>
        </div>

        <div v-if="isQutcraftOrganization" class="form-grid">
          <el-form-item label="Minecraft 游戏 ID" prop="game_id">
            <el-input v-model="form.game_id" placeholder="Java 版正版或统一 ID" />
          </el-form-item>

          <el-form-item label="QQ 号码" prop="qq_number">
            <el-input v-model="form.qq_number" placeholder="便于拉入内部交流群" />
          </el-form-item>
        </div>

        <el-form-item v-else label="申请说明">
          <el-input v-model="form.note" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="想参与的方向、技能或其他说明（可选）" />
        </el-form-item>

        <el-form-item label="电子邮箱" prop="email">
          <el-input v-model="form.email" autocomplete="email" placeholder="用于接收审核结果与通知" />
        </el-form-item>

        <div class="starlight-submit-row">
          <el-button
            type="primary"
            native-type="submit"
            size="large"
            round
            :loading="submitting"
            :icon="Promotion"
            class="cosmic-submit-btn"
          >
            {{ isQutcraftOrganization ? '提交申请 · 赴方块之约' : '提交成员申请' }}
          </el-button>
        </div>
      </el-form>
    </section>
  </div>
</template>
