<template>
  <div class="ssm-login">
    <div class="ssm-login__brand">
      <div class="brand-head">
        <img src="@/assets/imgs/logo.png" alt="" />
        <span>{{ title }}</span>
      </div>
      <div class="brand-body">
        <h1>智慧空间管理后台</h1>
        <p>门店 · 房间 · 设备 · 营收，一站式管理</p>
        <ul>
          <li>多门店统一管理</li>
          <li>设备实时状态与远程控制</li>
          <li>营收报表与退款审批</li>
        </ul>
      </div>
      <div class="brand-foot">无锡锡山阡月里信息科技工作室</div>
    </div>

    <div class="ssm-login__form">
      <div class="form-head">
        <img src="@/assets/imgs/logo.png" alt="" />
        <span>{{ title }}</span>
      </div>
      <el-card shadow="never" class="login-card">
        <h2>欢迎登录</h2>
        <p class="sub">请使用管理员账号登录</p>
        <el-form ref="formRef" :model="form" :rules="rules" size="large" @keyup.enter="submit">
          <el-form-item prop="username">
            <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" clearable />
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="密码"
              :prefix-icon="Lock"
              show-password
            />
          </el-form-item>
          <el-form-item v-if="captcha.question">
            <div class="captcha-row">
              <span class="captcha-question">安全验证：{{ captcha.question }}</span>
              <el-input
                v-model="captcha.answer"
                placeholder="请输入答案"
                inputmode="numeric"
                maxlength="8"
                class="captcha-input"
              />
            </div>
            <div class="captcha-help">连续登录失败，请输入验证码后重试</div>
          </el-form-item>
          <el-button type="primary" class="submit" :loading="loading" @click="submit">
            登 录
          </el-button>
        </el-form>
        <div class="tip">请使用管理员账号登录</div>
      </el-card>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Lock, User } from '@element-plus/icons-vue'
import { login } from '@/api/ssm'
import { setToken } from '@/utils/auth'
import { CACHE_KEY, useCache } from '@/hooks/web/useCache'

defineOptions({ name: 'Login' })

const route = useRoute()
const router = useRouter()
const { wsCache } = useCache()

const title = import.meta.env.VITE_APP_TITLE
const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({
  username: import.meta.env.VITE_APP_DEFAULT_LOGIN_USERNAME || '',
  password: import.meta.env.VITE_APP_DEFAULT_LOGIN_PASSWORD || ''
})
const captcha = reactive({ id: '', question: '', answer: '' })

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const submit = async () => {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  if (captcha.question && !captcha.answer.trim()) {
    ElMessage.warning('请输入验证码')
    return
  }

  loading.value = true
  try {
    const data = await login({
      username: form.username,
      password: form.password,
      ...(captcha.question
        ? { captcha_id: captcha.id, captcha_answer: captcha.answer.trim() }
        : {})
    })
    setToken({
      id: 0,
      accessToken: data.access_token,
      refreshToken: '',
      userId: data.user.id,
      userType: 0,
      clientId: '',
      expiresTime: 0
    })
    wsCache.set(CACHE_KEY.USER, {
      permissions: [],
      roles: [data.user.role],
      user: {
        id: data.user.id,
        nickname: data.user.nickname || '管理员',
        avatar: '',
        deptId: data.user.store_id || 0
      },
      menus: []
    })
    captcha.id = ''
    captcha.question = ''
    captcha.answer = ''
    ElMessage.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.replace(redirect)
  } catch (e: any) {
    if (e?.code === 10009 && e?.data) {
      captcha.id = String(e.data.captcha_id || '')
      captcha.question = String(e.data.question || '')
      captcha.answer = ''
    }
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.ssm-login {
  display: flex;
  height: 100%;
  background: #f5f7fa;

  &__brand {
    position: relative;
    display: flex;
    flex: 1;
    flex-direction: column;
    justify-content: space-between;
    padding: 40px 56px;
    color: #fff;
    background: linear-gradient(135deg, #0b2a6b 0%, #1a56db 55%, #3b82f6 100%);
    overflow: hidden;

    &::after {
      position: absolute;
      right: -120px;
      bottom: -160px;
      width: 520px;
      height: 520px;
      border-radius: 50%;
      background: rgba(255, 255, 255, 0.08);
      content: '';
    }
  }

  &__form {
    display: flex;
    flex: 0 0 520px;
    flex-direction: column;
    justify-content: center;
    padding: 40px 72px;
    background: #fff;
  }
}

.brand-head,
.form-head {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 20px;
  font-weight: 700;

  img {
    width: 44px;
    height: 44px;
    border-radius: 10px;
  }
}

.form-head {
  display: none;
  color: #1a56db;
}

.brand-body {
  position: relative;
  z-index: 1;

  h1 {
    margin: 0 0 16px;
    font-size: 42px;
    letter-spacing: 2px;
  }

  p {
    margin: 0 0 28px;
    font-size: 16px;
    opacity: 0.85;
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;

    li {
      margin-bottom: 14px;
      font-size: 15px;
      opacity: 0.9;

      &::before {
        margin-right: 8px;
        content: '✓';
      }
    }
  }
}

.brand-foot {
  position: relative;
  z-index: 1;
  font-size: 13px;
  opacity: 0.7;
}

.login-card {
  border: none;

  h2 {
    margin: 0;
    font-size: 26px;
    color: #1f2937;
  }

  .sub {
    margin: 8px 0 28px;
    font-size: 14px;
    color: #9ca3af;
  }

  .captcha-row {
    display: flex;
    gap: 10px;
    align-items: center;
    width: 100%;
  }

  .captcha-question {
    flex: 0 0 auto;
    font-size: 14px;
    font-weight: 600;
    color: #1f2937;
  }

  .captcha-input {
    flex: 1;
  }

  .captcha-help {
    margin-top: 4px;
    font-size: 12px;
    color: #9ca3af;
  }

  .submit {
    width: 100%;
    height: 44px;
    margin-top: 8px;
    font-size: 16px;
  }

  .tip {
    margin-top: 18px;
    font-size: 12px;
    color: #9ca3af;
    text-align: center;
  }
}

@media (max-width: 900px) {
  .ssm-login__brand {
    display: none;
  }

  .ssm-login__form {
    flex: 1;
    padding: 24px;
  }

  .form-head {
    display: flex;
    margin-bottom: 20px;
  }
}
</style>
