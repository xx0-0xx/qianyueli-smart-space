import { Layout } from '@/utils/routerHelper'

/**
 * 商家端静态路由
 * 对齐小程序商家端全部功能。
 */
const remainingRouter: AppRouteRecordRaw[] = [
  {
    path: '/login',
    name: 'MerchantLogin',
    component: () => import('@/views/merchant/Login.vue'),
    meta: {
      hidden: true,
      title: '商家登录',
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
        name: 'MerchantDashboard',
        component: () => import('@/views/merchant/Dashboard.vue'),
        meta: {
          title: '经营看板',
          icon: 'ep:data-board',
          affix: true,
          noCache: false
        }
      }
    ]
  },
  {
    path: '/orders',
    component: Layout,
    name: 'OrdersRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'MerchantOrders',
        component: () => import('@/views/merchant/Orders.vue'),
        meta: {
          title: '订单管理',
          icon: 'ep:tickets'
        }
      }
    ]
  },
  {
    path: '/rooms',
    component: Layout,
    redirect: '/rooms/resources',
    name: 'RoomsRoot',
    meta: {
      title: '房间管理',
      icon: 'ep:office-building'
    },
    children: [
      {
        path: 'resource-types',
        name: 'MerchantResourceTypes',
        component: () => import('@/views/merchant/ResourceTypes.vue'),
        meta: {
          title: '房型管理',
          icon: 'ep:grid'
        }
      },
      {
        path: 'resources',
        name: 'MerchantResources',
        component: () => import('@/views/merchant/Resources.vue'),
        meta: {
          title: '房间列表',
          icon: 'ep:list'
        }
      },
      {
        path: 'timeline',
        name: 'MerchantTimeline',
        component: () => import('@/views/merchant/Timeline.vue'),
        meta: {
          title: '房间时段',
          icon: 'ep:calendar'
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
        name: 'MerchantDevices',
        component: () => import('@/views/merchant/Devices.vue'),
        meta: {
          title: '设备管理',
          icon: 'ep:cpu'
        }
      }
    ]
  },
  {
    path: '/voice-config',
    component: Layout,
    name: 'VoiceConfigRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'MerchantVoiceConfig',
        component: () => import('@/views/merchant/Voice.vue'),
        meta: {
          title: '语音播报',
          icon: 'ep:microphone'
        }
      }
    ]
  },
  {
    path: '/marketing',
    component: Layout,
    redirect: '/marketing/pricing',
    name: 'MarketingRoot',
    meta: {
      title: '营销管理',
      icon: 'ep:present'
    },
    children: [
      {
        path: 'pricing',
        name: 'MerchantPricing',
        component: () => import('@/views/merchant/Pricing.vue'),
        meta: {
          title: '定价管理',
          icon: 'ep:price-tag'
        }
      },
      {
        path: 'coupons',
        name: 'MerchantCoupons',
        component: () => import('@/views/merchant/Coupons.vue'),
        meta: {
          title: '券码管理',
          icon: 'ep:ticket'
        }
      }
    ]
  },
  {
    path: '/membership',
    component: Layout,
    redirect: '/membership/levels',
    name: 'MembershipRoot',
    meta: {
      title: '会员管理',
      icon: 'ep:user'
    },
    children: [
      {
        path: 'levels',
        name: 'MerchantMemberLevels',
        component: () => import('@/views/merchant/MemberLevels.vue'),
        meta: {
          title: '会员等级',
          icon: 'ep:medal'
        }
      },
      {
        path: 'customers',
        name: 'MerchantCustomers',
        component: () => import('@/views/merchant/Customers.vue'),
        meta: {
          title: '客户管理',
          icon: 'ep:user-filled'
        }
      }
    ]
  },
  {
    path: '/finance',
    component: Layout,
    redirect: '/finance/recharges',
    name: 'FinanceRoot',
    meta: {
      title: '财务中心',
      icon: 'ep:wallet'
    },
    children: [
      {
        path: 'recharges',
        name: 'MerchantRecharges',
        component: () => import('@/views/merchant/Recharges.vue'),
        meta: {
          title: '充值确认',
          icon: 'ep:money'
        }
      },
      {
        path: 'report',
        name: 'MerchantReport',
        component: () => import('@/views/merchant/Report.vue'),
        meta: {
          title: '月度报表',
          icon: 'ep:data-analysis'
        }
      }
    ]
  },
  {
    path: '/store-config',
    component: Layout,
    name: 'StoreConfigRoot',
    meta: {},
    children: [
      {
        path: '',
        name: 'MerchantStoreConfig',
        component: () => import('@/views/merchant/StoreConfig.vue'),
        meta: {
          title: '门店设置',
          icon: 'ep:setting'
        }
      }
    ]
  },
  {
    path: '/agreement',
    name: 'MerchantAgreement',
    component: () => import('@/views/merchant/Agreement.vue'),
    meta: {
      hidden: true,
      title: '用户服务协议',
      noTagsView: true
    }
  },
  {
    path: '/privacy',
    name: 'MerchantPrivacy',
    component: () => import('@/views/merchant/Privacy.vue'),
    meta: {
      hidden: true,
      title: '隐私政策',
      noTagsView: true
    }
  },
  {
    path: '/user',
    component: Layout,
    name: 'UserRoot',
    meta: { hidden: true },
    children: [
      {
        path: 'profile',
        name: 'MerchantProfile',
        component: () => import('@/views/merchant/Profile.vue'),
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
