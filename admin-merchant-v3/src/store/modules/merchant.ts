import { defineStore } from 'pinia'
import { store } from '@/store'
import { getMerchantStores } from '@/api/merchant'

const STORE_KEY = 'merchantSid'

interface MerchantState {
  stores: any[]
  storeId: number | null
  storeName: string
  loading: boolean
}

export const useMerchantStore = defineStore('merchant', {
  state: (): MerchantState => ({
    stores: [],
    storeId: Number(localStorage.getItem(STORE_KEY)) || null,
    storeName: localStorage.getItem('merchantStoreName') || '',
    loading: false
  }),
  getters: {
    getStores(): any[] {
      return this.stores
    },
    getStoreId(): number | null {
      return this.storeId
    },
    getStoreName(): string {
      return this.storeName
    }
  },
  actions: {
    async loadStores() {
      this.loading = true
      try {
        const res = await getMerchantStores()
        this.stores = Array.isArray(res) ? res : (res as any)?.list || []
        if (this.storeId) {
          const current = this.stores.find((s) => Number(s.id) === Number(this.storeId))
          if (current) this.setStore(current.id, current.name)
        }
        if (!this.storeId && this.stores.length) {
          this.setStore(this.stores[0].id, this.stores[0].name)
        }
      } finally {
        this.loading = false
      }
    },
    setStore(id: number, name = '') {
      this.storeId = Number(id)
      this.storeName = name
      localStorage.setItem(STORE_KEY, String(id))
      localStorage.setItem('merchantStoreName', name)
    },
    clearStore() {
      this.storeId = null
      this.storeName = ''
      localStorage.removeItem(STORE_KEY)
      localStorage.removeItem('merchantStoreName')
    }
  }
})

export const useMerchantStoreWithOut = () => useMerchantStore(store)
