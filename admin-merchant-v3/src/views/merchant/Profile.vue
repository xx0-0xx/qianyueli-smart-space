<template>
  <ContentWrap title="个人中心" :body-style="{ padding: '0 18px 18px' }">
    <div class="profile-wrap">
      <div class="profile-hero">
        <div class="avatar">{{ avatarText }}</div>
        <div class="hero-copy">
          <div class="nickname">{{ nickname }}</div>
          <div class="hero-sub">{{ roleText }}</div>
        </div>
        <el-tag effect="light" type="primary" class="role-tag">{{ roleText }}</el-tag>
      </div>

      <div class="info-grid">
        <div class="info-card">
          <div class="info-label">当前门店</div>
          <div class="info-value">{{ currentStoreName }}</div>
        </div>
        <div class="info-card">
          <div class="info-label">可切换门店</div>
          <div class="info-value">
            {{ storeCount }}
            <span class="info-unit">家</span>
          </div>
        </div>
        <div class="info-card">
          <div class="info-label">所属角色</div>
          <div class="info-value">{{ roleText }}</div>
        </div>
      </div>

      <div class="profile-actions">
        <el-button type="danger" plain :loading="loggingOut" @click="handleLogout">
          退出登录
        </el-button>
      </div>
    </div>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { useUserStore } from '@/store/modules/user'

defineOptions({ name: 'MerchantProfile' })

const ROLE_LABELS: Record<string, string> = {
  merchant: '商家',
  admin: '管理员',
  super_admin: '超级管理员'
}

const router = useRouter()
const merchantStore = useMerchantStore()
const userStore = useUserStore()

const loggingOut = ref(false)

const nickname = computed(() => userStore.getUser?.nickname || '商家用户')
const avatarText = computed(() => (nickname.value || '商').slice(0, 1).toUpperCase())

const roleText = computed(() => {
  const roles = userStore.getRoles || []
  if (!roles.length) return '商家'
  return roles.map((role) => ROLE_LABELS[role] || role).join(' / ')
})

const currentStoreName = computed(() => {
  if (merchantStore.storeName) return merchantStore.storeName
  const current = merchantStore.stores.find(
    (store) => Number(store.id) === Number(merchantStore.storeId)
  )
  return current?.name || '未选择门店'
})

const storeCount = computed(() => merchantStore.stores.length)

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm(
      '退出后需重新输入门店邀请码登录，确定退出吗？',
      '退出登录',
      {
        type: 'warning',
        confirmButtonText: '退出登录',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }

  loggingOut.value = true
  try {
    await userStore.loginOut()
    merchantStore.clearStore()
    ElMessage.success('已退出登录')
    router.replace('/login')
  } finally {
    loggingOut.value = false
  }
}

onMounted(() => {
  if (!merchantStore.stores.length) {
    merchantStore.loadStores().catch(() => {
      // 请求层已统一提示错误
    })
  }
})
</script>

<style lang="scss" scoped>
.profile-wrap {
  max-width: 760px;
  padding-top: 4px;
}

.profile-hero {
  display: flex;
  gap: 16px;
  align-items: center;
  padding: 20px 22px;
  background: linear-gradient(135deg, #eef4ff 0%, var(--el-fill-color-light) 100%);
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
}

.avatar {
  display: flex;
  flex: 0 0 58px;
  width: 58px;
  height: 58px;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 700;
  color: #fff;
  background: linear-gradient(135deg, #1a56db 0%, #3b82f6 100%);
  border-radius: 50%;
}

.hero-copy {
  flex: 1;
  min-width: 0;
}

.nickname {
  overflow: hidden;
  font-size: 20px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hero-sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.role-tag {
  flex: 0 0 auto;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-top: 18px;
}

.info-card {
  padding: 16px 18px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
}

.info-label {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.info-value {
  margin-top: 8px;
  overflow: hidden;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-unit {
  margin-left: 2px;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-placeholder);
}

.profile-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 22px;
  margin-top: 18px;
  border-top: 1px dashed var(--el-border-color-light);
}

@media (max-width: 768px) {
  .info-grid {
    grid-template-columns: 1fr;
  }

  .profile-hero {
    align-items: flex-start;
  }

  .role-tag {
    display: none;
  }

  .profile-actions {
    justify-content: stretch;
  }

  .profile-actions :deep(.el-button) {
    width: 100%;
  }
}
</style>
