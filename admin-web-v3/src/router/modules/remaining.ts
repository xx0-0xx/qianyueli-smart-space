import { Layout } from '@/utils/routerHelper'

/**
 * 智慧空间管理后台静态路由
 * 覆盖原 admin-web / 桌面版全部功能。
 */
const remainingRouter: AppRouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/ssm/Login.vue'),
    meta: {
      hidden: true,
      title: '登录',
      noTagsView: true
    }
  },
  {
    path: '/redirect',
    component: Layout,
    name: 'RedirectRoot',
    children: [
      {
        path: '/redirect/:path(.*)',
        name: 'Redirect',
        component: () => import('@/views/Redirect/Redirect.vue'),
        meta: {}
      }
    ],
    meta: {
      hidden: true,
      noTagsView: true
    }
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    name: 'Home',
    meta: {},
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/ssm/Dashboard.vue'),
        meta: {
          title: '首页',
          icon: 'ep:home-filled',
          affix: true,
          noCache: false
        }
      }
    ]
  },
  {
    path: '/stores',
    component: Layout,
    name: 'StoresRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Stores',
        component: () => import('@/views/ssm/Stores.vue'),
        meta: {
          title: '门店管理',
          icon: 'ep:shop'
        }
      }
    ]
  },
  {
    path: '/resource-types',
    component: Layout,
    name: 'ResourceTypesRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'ResourceTypes',
        component: () => import('@/views/ssm/ResourceTypes.vue'),
        meta: {
          title: '房型管理',
          icon: 'ep:grid'
        }
      }
    ]
  },
  {
    path: '/resources',
    component: Layout,
    name: 'ResourcesRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Resources',
        component: () => import('@/views/ssm/Resources.vue'),
        meta: {
          title: '房间管理',
          icon: 'ep:office-building'
        }
      }
    ]
  },
  {
    path: '/devices',
    component: Layout,
    name: 'DevicesRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Devices',
        component: () => import('@/views/ssm/Devices.vue'),
        meta: {
          title: '设备管理',
          icon: 'ep:cpu'
        }
      }
    ]
  },
  {
    path: '/audio',
    component: Layout,
    name: 'AudioRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Audio',
        component: () => import('@/views/ssm/Audio.vue'),
        meta: {
          title: '音频管理',
          icon: 'ep:headset'
        }
      }
    ]
  },
  {
    path: '/voice',
    component: Layout,
    name: 'VoiceRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Voice',
        component: () => import('@/views/ssm/Voice.vue'),
        meta: {
          title: '语音播报',
          icon: 'ep:microphone'
        }
      }
    ]
  },
  {
    path: '/product-templates',
    component: Layout,
    name: 'ProductTemplatesRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'ProductTemplates',
        component: () => import('@/views/ssm/ProductTemplates.vue'),
        meta: {
          title: '产品模板',
          icon: 'ep:box'
        }
      }
    ]
  },
  {
    path: '/leads',
    component: Layout,
    name: 'LeadsRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Leads',
        component: () => import('@/views/ssm/Leads.vue'),
        meta: {
          title: '意向客户',
          icon: 'ep:user-filled'
        }
      }
    ]
  },
  {
    path: '/report',
    component: Layout,
    name: 'ReportRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Report',
        component: () => import('@/views/ssm/Report.vue'),
        meta: {
          title: '数据报表',
          icon: 'ep:data-analysis'
        }
      }
    ]
  },
  {
    path: '/elys',
    component: Layout,
    name: 'ElysRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'Elys',
        component: () => import('@/views/ssm/Elys.vue'),
        meta: {
          title: '团购核销',
          icon: 'ep:ticket'
        }
      }
    ]
  },
  {
    path: '/tools',
    component: Layout,
    redirect: '/tools/device-test',
    name: 'ToolsRoot',
    meta: {
      title: '调试工具',
      icon: 'ep:tools'
    },
    children: [
      {
        path: 'device-test',
        name: 'DeviceTest',
        component: () => import('@/views/ssm/DeviceTest.vue'),
        meta: {
          title: '设备调试',
          icon: 'ep:monitor'
        }
      },
      {
        path: 'speaker-test',
        name: 'SpeakerTest',
        component: () => import('@/views/ssm/SpeakerTest.vue'),
        meta: {
          title: '音响测试',
          icon: 'ep:bell'
        }
      }
    ]
  },
  {
    path: '/user',
    component: Layout,
    name: 'UserRoot',
    meta: { hidden: true },
    children: [
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/ssm/Profile.vue'),
        meta: {
          title: '个人中心',
          icon: 'ep:user',
          hidden: true,
          canTo: true
        }
      }
    ]
  },
  {
    path: '/index',
    redirect: '/dashboard',
    name: 'LegacyIndexRedirect',
    meta: {
      hidden: true,
      noTagsView: true
    }
  },
  {
    path: '/:path(.*)*',
    name: '404Page',
    component: () => import('@/views/Error/404.vue'),
    meta: {
      hidden: true,
      breadcrumb: false
    }
  }
]

export default remainingRouter
