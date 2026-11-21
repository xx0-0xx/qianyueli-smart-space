import router from './router'
import { getAccessToken } from '@/utils/auth'
import { useTitle } from '@/hooks/web/useTitle'
import { useNProgress } from '@/hooks/web/useNProgress'
import { usePageLoading } from '@/hooks/web/usePageLoading'
import { useUserStoreWithOut } from '@/store/modules/user'
import { usePermissionStoreWithOut } from '@/store/modules/permission'

const { start, done } = useNProgress()
const { loadStart, loadDone } = usePageLoading()

const whiteList = ['/login', '/agreement', '/privacy']

router.beforeEach(async (to, _from, next) => {
  start()
  loadStart()

  if (getAccessToken()) {
    if (to.path === '/login') {
      next({ path: '/' })
      return
    }
    const userStore = useUserStoreWithOut()
    if (!userStore.getIsSetUser) {
      await userStore.setUserInfoAction()
    }
    const permissionStore = usePermissionStoreWithOut()
    if (!permissionStore.getRouters.length) {
      await permissionStore.generateRoutes()
    }
    next()
  } else if (whiteList.includes(to.path)) {
    next()
  } else {
    next(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }
})

router.afterEach((to) => {
  useTitle(to?.meta?.title as string)
  done()
  loadDone()
})
