var api=require('../../utils/api')
var STORE_KEY='selected_store_id'
function dist(lat1,lon1,lat2,lon2){
  if(!lat1||!lon1||!lat2||!lon2)return 1e9
  var R=6371000,dLat=(lat2-lat1)*Math.PI/180,dLon=(lon2-lon1)*Math.PI/180
  var a=Math.sin(dLat/2)*Math.sin(dLat/2)+Math.cos(lat1*Math.PI/180)*Math.cos(lat2*Math.PI/180)*Math.sin(dLon/2)*Math.sin(dLon/2)
  return R*2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a))
}
Page({
  data:{storeName:'',storeAddr:'',cover:'',loggedIn:false,wifiName:'',wifiPass:'',phone:'',storeLat:0,storeLon:0,storeId:'',showStoreList:false,storeList:[],locating:false,statusBarH:20},
  onShareAppMessage:function(){
    var t=this.data.storeName?(this.data.storeName+' · 金之莱无人空间'):'金之莱无人空间'
    return {title:t,path:'/pages/index/index'}
  },
  onShareTimeline:function(){
    var t=this.data.storeName?(this.data.storeName+' · 金之莱无人空间'):'金之莱无人空间'
    return {title:t}
  },
  noop:function(){},
  onLoad:function(options){
    var sys=wx.getSystemInfoSync()
    this.setData({statusBarH:sys.statusBarHeight||20})
    // 小程序码进入：scene 参数（需解码，形如 "store=6913002"）
    var scene=''
    if(options&&options.scene){scene=decodeURIComponent(options.scene)}
    else if(options&&options.store_id){scene='store='+options.store_id}
    var m=scene.match(/store=(\d+)/)
    if(m){wx.setStorageSync(STORE_KEY,m[1])}
  },
  onShow:function(){
    this.setData({loggedIn:!!getApp().globalData.token})
    this.initStore()
  },
  // 启动逻辑：冷启动（退出重进）或距上次定位超过 10 分钟才重新定位；
  // 会话内短时间内反复进出首页直接用当前门店
  initStore:function(){
    var app=getApp()
    var sid=wx.getStorageSync(STORE_KEY)
    var lastTs=wx.getStorageSync('last_locate_ts')||0
    var expired=(Date.now()-lastTs)>10*60*1000
    // 会话内且未过期：用已选门店，不定位
    if(app.globalData.sessionLocated&&!expired){
      if(sid){this.setData({storeId:sid});this.loadStore(sid);return}
      this.showStoreList()
      return
    }
    // 冷启动或超时：重新定位
    app.globalData.sessionLocated=true
    wx.setStorageSync('last_locate_ts',Date.now())
    if(sid){
      this.setData({storeId:sid})
      this.autoLocateWithFallback(sid)
    }else{
      this.autoLocate()
    }
  },
  autoLocateWithFallback:function(fallbackId){
    var self=this
    wx.getLocation({
      type:'gcj02',
      success:function(res){self.pickNearest(res.latitude,res.longitude)},
      fail:function(){self.loadStore(fallbackId)}
    })
  },
  autoLocate:function(){
    var self=this
    this.setData({locating:true})
    wx.getLocation({
      type:'gcj02',
      success:function(res){self.pickNearest(res.latitude,res.longitude)},
      fail:function(){
        self.setData({locating:false})
        self.showStoreList()
      }
    })
  },
  // 计算最近门店并展示
  pickNearest:function(lat,lon){
    var self=this
    self._myLat=lat;self._myLon=lon
    api.getStores().then(function(r){
      var list=(r.data&&r.data.list)||r.data||r||[]
      if(!Array.isArray(list))list=[]
      var nearest=null,best=1e18
      list.forEach(function(s){
        var d=dist(lat,lon,s.latitude,s.longitude)
        if(d<best){best=d;nearest=s}
      })
      if(nearest){
        wx.setStorageSync(STORE_KEY,String(nearest.id))
        self.setData({storeId:String(nearest.id),locating:false})
        self.loadStore(nearest.id)
      }else{
        self.setData({locating:false})
        self.showStoreList()
      }
    }).catch(function(){self.setData({locating:false})})
  },
  loadStore:function(id){
    var self=this
    // 1. 先读本地缓存 → 立即显示（封面秒出）
    try{
      var cached=wx.getStorageSync('cache_store_'+id)
      if(cached&&cached.cover){
        cached.cover=String(cached.cover).replace('http://127.0.0.1:8080','http://localhost:8080')
        this.setData(cached)
      }
    }catch(e){}
    // 2. 后台拉最新数据
    api.getStore(id).then(function(r){
      var s=r.data||r
      var d={storeName:s.name,storeAddr:s.address,cover:String(s.cover||'').replace('http://127.0.0.1:8080','http://localhost:8080'),phone:s.phone||'',storeLat:s.latitude||0,storeLon:s.longitude||0}
      try{
        var cfg=JSON.parse(s.config_json||'{}')
        d.wifiName=cfg.wifi_ssid||'';d.wifiPass=cfg.wifi_pass||''
      }catch(e){}
      self.setData(d)
      // 3. 写入缓存供下次秒开
      wx.setStorageSync('cache_store_'+id, d)
    })
  },
  // 门店切换
  showStoreList:function(){
    var self=this
    api.getStores().then(function(r){
      var list=(r.data&&r.data.list)||r.data||r||[]
      if(!Array.isArray(list))list=[]
      list.forEach(function(s){s._distText=self._distLabel(s)})
      self.setData({storeList:list,showStoreList:true})
    }).catch(function(){wx.showToast({title:'获取门店列表失败',icon:'none'})})
  },
  _distLabel:function(s){
    var lat=this._myLat,lon=this._myLon
    if(!lat||!lon)return ''
    var d=dist(lat,lon,s.latitude,s.longitude)
    return d<1000?Math.round(d)+'m':(d/1000).toFixed(1)+'km'
  },
  hideStoreList:function(){this.setData({showStoreList:false})},
  pickStore:function(e){
    var sid=e.currentTarget.dataset.id
    wx.setStorageSync(STORE_KEY,String(sid))
    this.setData({storeId:String(sid),showStoreList:false})
    this.loadStore(sid)
  },
  goLogin:function(){wx.reLaunch({url:'/pages/login/login'})},
  goBook:function(){wx.navigateTo({url:'/pages/store/store?id='+this.data.storeId})},
  goRedeem:function(){wx.navigateTo({url:'/pages/redeem/redeem'})},
  showWifi:function(){
    var n=this.data.wifiName||'Guest-Free'
    var p=this.data.wifiPass||'88888888'
    wx.showModal({title:'店内WiFi',content:'WiFi：'+n+'\n密码：'+p,showCancel:false,confirmText:'知道了'})
  },
  callPhone:function(){
    var phone=this.data.phone||'18912391927'
    wx.makePhoneCall({phoneNumber:phone})
  },
  openMap:function(){
    var lat=this.data.storeLat,lon=this.data.storeLon
    if(!lat||!lon){wx.showToast({title:'门店暂未设置坐标',icon:'none'});return}
    wx.openLocation({latitude:lat,longitude:lon,name:this.data.storeName,address:this.data.storeAddr,scale:18})
  }
})
