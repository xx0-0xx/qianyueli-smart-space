<template>
  <div class="merchant-login">
    <div class="merchant-login__brand">
      <div class="brand-head">
        <img src="@/assets/imgs/logo.png" alt="" />
        <span>{{ title }}</span>
      </div>
      <div class="brand-body">
        <h1>商家经营后台</h1>
        <p>门店经营 · 订单处理 · 设备控制，一个后台全搞定</p>
        <ul>
          <li>实时看板与房间状态</li>
          <li>订单、退款、充值一站处理</li>
          <li>设备远程控制与语音配置</li>
        </ul>
      </div>
      <div class="brand-foot">无锡锡山阡月里信息科技工作室</div>
    </div>

    <div class="merchant-login__form">
      <div class="form-head">
        <img src="@/assets/imgs/logo.png" alt="" />
        <span>{{ title }}</span>
      </div>
      <el-card shadow="never" class="login-card">
        <h2>商家登录</h2>
        <p class="sub">请输入门店邀请码进入后台</p>
        <el-form ref="formRef" :model="form" :rules="rules" size="large" @keyup.enter="submit">
          <el-form-item prop="inviteCode">
            <el-input
              v-model="form.inviteCode"
              placeholder="门店邀请码"
              :prefix-icon="Key"
              clearable
              @input="onInput"
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
          <div class="agreement">
            <el-checkbox v-model="agreed">
              我已阅读并同意
              <el-link type="primary" :underline="false" @click="open('/agreement')">
                《用户服务协议》
              </el-link>
              和
              <el-link type="primary" :underline="false" @click="open('/privacy')">
                《隐私政策》
              </el-link>
            </el-checkbox>
          </div>
          <el-button type="primary" class="submit" :loading="loading" @click="submit">
            进入后台
          </el-button>
        </el-form>
        <div class="tip">邀请码由平台管理员在「门店管理」中生成</div>
      </el-card>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Key } from '@element-plus/icons-vue'
import { inviteLoginSecure } from '@/api/merchant'
import { setToken } from '@/utils/auth'
import { CACHE_KEY, useCache } from '@/hooks/web/useCache'

defineOptions({ name: 'MerchantLogin' })

const router = useRouter()
const { wsCache } = useCache()
const title = import.meta.env.VITE_APP_TITLE
const formRef = ref<FormInstance>()
const loading = ref(false)
const agreed = ref(false)
const form = reactive({ inviteCode: '' })
const captcha = reactive({ id: '', question: '', answer: '' })

const rules: FormRules = {
  inviteCode: [{ required: true, message: '请输入门店邀请码', trigger: 'blur' }]
}

const onInput = (value: string) => {
  form.inviteCode = (value || '').toUpperCase()
}

const open = (path: string) => {
  const url = router.resolve(path)
  window.open(url.href, '_blank')
}

const submit = async () => {
  if (!agreed.value) {
    ElMessage.warning('请先阅读并同意协议')
    return
  }
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  if (captcha.question && !captcha.answer.trim()) {
    ElMessage.warning('请输入验证码')
    return
  }

  loading.value = true
  try {
    const res = await inviteLoginSecure(
      form.inviteCode.trim(),
      captcha.question
        ? { captcha_id: captcha.id, captcha_answer: captcha.answer.trim() }
        : undefined
    )
    const user = res.user || ({} as any)
    const storeId = Number(user.store_id || res.store_id || res.store?.id || 0)
    setToken({
      id: 0,
      accessToken: res.access_token,
      refreshToken: '',
      userId: user.id || 0,
      userType: 0,
      clientId: '',
      expiresTime: 0
    })
    localStorage.removeItem('merchantSid')
    localStorage.removeItem('merchantStoreName')
    if (storeId) {
      localStorage.setItem('merchantSid', String(storeId))
      localStorage.setItem('merchantStoreName', res.store?.name || '')
    }
    wsCache.set(CACHE_KEY.USER, {
      permissions: [],
      roles: [user.role || 'merchant'],
      user: {
        id: user.id || 0,
        nickname: user.nickname || '商家',
        avatar: '',
        deptId: storeId
      },
      menus: []
    })
    captcha.id = ''
    captcha.question = ''
    captcha.answer = ''
    ElMessage.success('进入成功')
    router.replace('/dashboard')
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
.merchant-login {
  display: flex;
  height: 100%;
  background: var(--el-bg-color-page);

  &__brand {
    position: relative;
    display: flex;
    flex: 1;
    flex-direction: column;
    justify-content: space-between;
    padding: 40px 56px;
    overflow: hidden;
    color: #fff;
    background: linear-gradient(135deg, #0b2a6b 0%, #1a56db 55%, #3b82f6 100%);

    &::after {
      position: absolute;
      right: -120px;
      bottom: -160px;
      width: 520px;
      height: 520px;
      content: '';
      background: rgb(255 255 255 / 8%);
      border-radius: 50%;
    }
  }

  &__form {
    display: flex;
    flex: 0 0 520px;
    flex-direction: column;
    justify-content: center;
    padding: 40px 72px;
    background: var(--el-bg-color);
  }
}

.brand-head,
.form-head {
  display: flex;
  gap: 12px;
  align-items: center;
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
    padding: 0;
    margin: 0;
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
    color: var(--el-text-color-primary);
  }

  .sub {
    margin: 8px 0 24px;
    font-size: 14px;
    color: var(--el-text-color-placeholder);
  }

  .agreement {
    margin-bottom: 16px;
    font-size: 13px;
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
    color: var(--el-text-color-primary);
  }

  .captcha-input {
    flex: 1;
  }

  .captcha-help {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
  }

  .submit {
    width: 100%;
    height: 44px;
    font-size: 16px;
  }

  .tip {
    margin-top: 18px;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
    text-align: center;
  }
}

@media (max-width: 900px) {
  .merchant-login__brand {
    display: none;
  }

  .merchant-login__form {
    flex: 1;
    padding: 24px;
  }

  .form-head {
    display: flex;
    margin-bottom: 20px;
  }
}
</style>
