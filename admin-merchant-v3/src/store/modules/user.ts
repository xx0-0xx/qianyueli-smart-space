import { store } from '@/store'
import { defineStore } from 'pinia'
import { getAccessToken, removeToken } from '@/utils/auth'
import { CACHE_KEY, useCache, deleteUserCache } from '@/hooks/web/useCache'

const { wsCache } = useCache()

interface UserVO {
  id: number
  avatar: string
  nickname: string
  deptId: number
}

interface UserInfoVO {
  permissions: Set<string>
  roles: string[]
  isSetUser: boolean
  user: UserVO
}

const emptyUser = (): UserVO => ({
  id: 0,
  avatar: '',
  nickname: '',
  deptId: 0
})

export const useUserStore = defineStore('admin-user', {
  state: (): UserInfoVO => ({
    permissions: new Set<string>(),
    roles: [],
    isSetUser: false,
    user: emptyUser()
  }),
  getters: {
    getPermissions(): Set<string> {
      return this.permissions
    },
    getRoles(): string[] {
      return this.roles
    },
    getIsSetUser(): boolean {
      return this.isSetUser
    },
    getUser(): UserVO {
      return this.user
    }
  },
  actions: {
    async setUserInfoAction() {
      if (!getAccessToken()) {
        this.resetState()
        return null
      }
      const userInfo = wsCache.get(CACHE_KEY.USER)
      if (!userInfo) {
        this.resetState()
        return null
      }
      this.permissions = new Set(userInfo.permissions || [])
      this.roles = userInfo.roles || []
      this.user = userInfo.user || emptyUser()
      this.isSetUser = true
    },
    async loginOut() {
      removeToken()
      deleteUserCache()
      this.resetState()
    },
    resetState() {
      this.permissions = new Set<string>()
      this.roles = []
      this.isSetUser = false
      this.user = emptyUser()
    }
  }
})

export const useUserStoreWithOut = () => {
  return useUserStore(store)
}
