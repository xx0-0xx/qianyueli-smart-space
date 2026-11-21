<template>
  <ContentWrap title="个人中心">
    <div class="profile">
      <el-avatar :size="72" :src="user.avatar || undefined">
        {{ (user.nickname || 'A').slice(0, 1) }}
      </el-avatar>
      <div class="profile__info">
        <div class="name">{{ user.nickname || '管理员' }}</div>
        <div class="role">{{ roleLabel }}</div>
      </div>
    </div>

    <el-descriptions :column="2" border class="desc">
      <el-descriptions-item label="用户 ID">{{ user.id }}</el-descriptions-item>
      <el-descriptions-item label="角色">{{ roleLabel }}</el-descriptions-item>
      <el-descriptions-item label="所属门店 ID">{{ user.deptId || '-' }}</el-descriptions-item>
      <el-descriptions-item label="登录方式">账号密码</el-descriptions-item>
    </el-descriptions>

    <el-button type="danger" plain @click="logout">退出登录</el-button>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ContentWrap } from '@/components/ContentWrap'
import { useUserStore } from '@/store/modules/user'

defineOptions({ name: 'Profile' })

const router = useRouter()
const userStore = useUserStore()
const user = computed(() => userStore.getUser)

const roleLabel = computed(() => {
  const role = userStore.getRoles[0]
  return role === 'super_admin' ? '超级管理员' : role === 'admin' ? '管理员' : role || '管理员'
})

const logout = async () => {
  await userStore.loginOut()
  router.replace('/login')
}
</script>

<style lang="scss" scoped>
.profile {
  display: flex;
  gap: 18px;
  align-items: center;
  padding: 8px 0 24px;

  &__info {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .name {
    font-size: 20px;
    font-weight: 700;
    color: #111827;
  }

  .role {
    font-size: 13px;
    color: #6b7280;
  }
}

.desc {
  max-width: 720px;
  margin-bottom: 24px;
}
</style>
