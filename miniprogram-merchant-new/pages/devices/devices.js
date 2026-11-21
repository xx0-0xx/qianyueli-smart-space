const api=require('../../utils/api');
var TYPE_LABELS={door_lock:'门锁',light:'灯光',ac:'空调',exhaust:'排风',machine:'插座/机器',other:'其他'};
function makeButtons(dev){var t=dev.type,n=dev.name||'设备';
  // 模板设备：读 config_json 的 cmd_on/cmd_off
  try{
    var c=JSON.parse(dev.config_json||'')
    if(c.cmd_on){
      if(c.is_toggle===false)return[{label:n,cmd:c.cmd_on}]
      return[{label:n+'开',cmd:c.cmd_on},{label:n+'关',cmd:c.cmd_off||''}]
    }
    if(c.r1)dev._comboCfg=c
  }catch(e){}
  if(t==='door_lock')return[{label:'开门',cmd:'door_open'},{label:'关门',cmd:'door_close'}];
  if(t==='machine')return[{label:n+'开',cmd:'machine_on'},{label:n+'关',cmd:'machine_off'}];
  if(t==='combo'){var r1='灯',r2='空调',r3='排风';try{var c=JSON.parse(dev.config_json||'{}');if(c.r1)r1=c.r1;if(c.r2)r2=c.r2;if(c.r3)r3=c.r3}catch(e){}return[{label:'开门',cmd:'door_open'},{label:'关门',cmd:'door_close'},{label:r1+'开',cmd:'light_on'},{label:r1+'关',cmd:'light_off'},{label:r2+'开',cmd:'ac_on'},{label:r2+'关',cmd:'ac_off'},{label:r3+'开',cmd:'exhaust_on'},{label:r3+'关',cmd:'exhaust_off'}]}
  var cmdMap={light:'light',ac:'ac',exhaust:'exhaust',other:'light'};var pre=cmdMap[t]||'light';
  return[{label:n+'开',cmd:pre+'_on'},{label:n+'关',cmd:pre+'_off'}]}
function decorate(d){d._typeLabel=TYPE_LABELS[d.type]||d.type;try{var cfg=JSON.parse(d.config_json||'{}');d._config=cfg;d.autoClose=!!cfg.auto_close;var _dl=Number(cfg.auto_close_delay_min)||0;d.autoCloseDelay=_dl;d._autoText=d.autoClose?('关'+(_dl>0?'('+_dl+'分)':'')):'开';d._autoBg=d.autoClose?'#FBCD08':'#f5f5f5';d._autoColor=d.autoClose?'#022A99':'#666';d._closeStr=d.autoClose?'1':'0'}catch(e){d._config={};d.autoClose=false;d.autoCloseDelay=0;d._autoText='开';d._autoBg='#f5f5f5';d._autoColor='#666'}d.userControlEnabled=!!d.user_control_enabled;d._userControlStr=d.userControlEnabled?'1':'0';return d}
Page({
  data:{groups:[],expandedRoom:'',showRoomPick:false,pendingDevice:null,roomChoices:[],typeOptions:[
    {v:'door_lock',label:'门锁'},{v:'light',label:'灯光'},{v:'machine',label:'插座/机器'},{v:'ac',label:'空调'},{v:'exhaust',label:'排风'},{v:'other',label:'其他'}
  ],showBind:false,bindType:'machine',bindName:'',bindSn:'',bindRoomId:0,allRooms:[],roomsCache:[]},
  goBack(){wx.navigateBack()},
  onShow(){this.load()},
  toggleRoom(e){var rid=e.currentTarget.dataset.rid;this.setData({expandedRoom:this.data.expandedRoom===rid?'':rid})},
  noop(){},
  async load(){
    try{
      var r1=await api.getResources();var r2=await api.getMerchantDevices();
      var rooms=r1||[];var devs=r2||[];
      this.roomsCache=rooms;
      var typeMap={};rooms.forEach(function(room){
        var tid=room.resource_type_id||(room.resource_type&&room.resource_type.id);var tname=(room.resource_type&&room.resource_type.name)||'未分类';
        if(!typeMap[tid])typeMap[tid]={id:tid,name:tname,rooms:[]};
        var roomDevices=[];try{var ids=JSON.parse(room.device_ids||'[]').filter(function(x){return x&&x!==''}).map(function(x){return Number(x)});roomDevices=devs.filter(function(d){return ids.some(function(id){return Number(id)===Number(d.id)})})}catch(e){}
        roomDevices.forEach(decorate);
        typeMap[tid].rooms.push({id:room.id,name:room.name,devices:roomDevices.map(function(d){return Object.assign({},d,{buttons:makeButtons(d)})})})});
      var boundIds=[];rooms.forEach(function(r){try{var ids=JSON.parse(r.device_ids||'[]').filter(function(x){return x&&x!==''}).map(function(x){return Number(x)});ids.forEach(function(id){boundIds.push(Number(id))})}catch(e){}});
      var unbound=devs.filter(function(d){return!boundIds.some(function(id){return Number(id)===Number(d.id)})});
      if(unbound.length>0){if(!typeMap[0])typeMap[0]={id:0,name:'未绑定',rooms:[]};unbound.forEach(decorate);typeMap[0].rooms.push({id:0,name:'未分配设备',devices:unbound.map(function(d){return Object.assign({},d,{buttons:makeButtons(d)})})})}
      var groups=Object.keys(typeMap).map(function(k){return typeMap[k]});
      // 「未分配设备」组永远排在最后
      groups.sort(function(a,b){return (a.id===0?1:0)-(b.id===0?1:0)});
      this.setData({groups:groups,allRooms:rooms});
    }catch(e){}
  },
  // ── 添加设备（与房间管理一致：扫码/输SN → 选类型 → 命名） ──
  showBind:function(){
    var self=this
    wx.showActionSheet({
      itemList:["扫码绑定","手动输入SN"],
      success:function(res){
        if(res.tapIndex===0)self.scanBind()
        else self.inputBind()
      }
    })
  },
  scanBind:function(){
    var self=this
    wx.scanCode({success:function(res){self.pickType(res.result.trim())}})
  },
  inputBind:function(){
    var self=this
    wx.showModal({
      title:"输入设备SN",
      editable:true,
      placeholderText:"设备序列号",
      success:function(res){
        if(res.confirm&&res.content)self.pickType(res.content.trim())
      }
    })
  },
  pickType:function(sn){
    var self=this
    self._bindSn=sn
    api.getProductTemplates().then(function(list){
      var arr=list||[]
      if(!Array.isArray(arr))arr=arr.list||[]
      if(arr.length===0){wx.showToast({title:'暂无产品，请先在管理后台添加',icon:'none'});return}
      self.setData({templateList:arr,showTemplatePick:true})
    }).catch(function(){wx.showToast({title:'获取产品列表失败',icon:'none'})})
  },
  hideTemplatePick:function(){this.setData({showTemplatePick:false})},
  pickTemplate:function(e){
    var self=this
    var tid=Number(e.currentTarget.dataset.id)
    var tpl=this.data.templateList.find(function(t){return Number(t.id)===tid})
    if(!tpl){return}
    this.setData({showTemplatePick:false})
    // 设备名称由模板子设备决定，商家添加后可在设备列表里重命名。
    var defaultName=tpl.devices&&tpl.devices.length===1?(tpl.devices[0].name||tpl.name):tpl.name
    self.pickRoom(self._bindSn,defaultName,tid,true)
  },
  // 选择绑定房间。使用可滚动弹层，不受 showActionSheet 选项数量限制。
  pickRoom:function(sn,name,type,isTemplate){
    var self=this
    api.getResources().then(function(rooms){
      var list=Array.isArray(rooms)?rooms:[]
      if(list.length===0){
        wx.showToast({title:'当前门店还没有房间，请先创建房间',icon:'none'})
        return
      }
      self.setData({
        roomChoices:list,
        pendingDevice:{sn:sn,name:name,type:type,isTemplate:!!isTemplate},
        showRoomPick:true
      })
    }).catch(function(err){
      wx.showToast({title:(err&&err.message)||'获取房间列表失败，请稍后重试',icon:'none'})
    })
  },
  cancelRoomPick:function(){
    this.setData({showRoomPick:false,pendingDevice:null,roomChoices:[]})
    wx.showToast({title:'已取消添加',icon:'none'})
  },
  selectRoom:function(e){
    var roomId=e.currentTarget.dataset.id
    var pending=this.data.pendingDevice
    if(!pending||!roomId)return
    this.setData({showRoomPick:false,pendingDevice:null,roomChoices:[]})
    this.createAndLink(pending.sn,pending.name,pending.type,roomId,pending.isTemplate)
  },
  createAndLink:function(sn,name,type,roomId,isTemplate){
    var self=this
    var storeId=getApp().globalData.storeId
    var sid=storeId?parseInt(storeId):0
    wx.showLoading({title:"添加中...",mask:true})
    var body={device_sn:sn,name:name,type:(isTemplate?'other':type),store_id:sid,room_id:parseInt(roomId)||0}
    if(isTemplate){body.mqtt_topic=String(type)} // 复用 mqtt_topic 传 template_id
    api.createDevice(body)
      .then(function(){
        wx.hideLoading()
        wx.showToast({title:"已添加并绑定到房间"})
        self.load()
      }).catch(function(e){
        console.log('createDevice fail',e)
        wx.hideLoading()
        wx.showToast({title:(e&&e.message)||'添加失败',icon:"none"})
      })
  },
  // 分配设备到房间（单选：自动从其他房间移出，保证一个设备只属一个房间）
  assignDev:function(e){
    var self=this
    var did=e.currentTarget.dataset.id
    api.getResources().then(function(rooms){
      var list=rooms||[]
      if(!Array.isArray(list))list=[]
      if(list.length===0){wx.showToast({title:"暂无房间可分配",icon:"none"});return}
      wx.showActionSheet({
        itemList:list.map(function(r){return r.name}),
        success:function(res){self.moveDeviceToRoom(did, list[res.tapIndex].id)}
      })
    }).catch(function(){wx.showToast({title:"获取房间失败",icon:"none"})})
  },
  // 把设备移动到目标房间：先从所有其他房间移除，再加入目标房间
  moveDeviceToRoom:function(did, roomId){
    var self=this
    wx.showLoading({title:"分配中...",mask:true})
    api.getResources().then(function(rooms){
      var list=rooms||[]
      var tasks=[]
      list.forEach(function(r){
        var ids=[]
        try{ids=JSON.parse(r.device_ids||"[]").filter(function(x){return x&&x!==""})}catch(e){}
        var has=ids.some(function(x){return Number(x)===Number(did)})
        if(Number(r.id)===Number(roomId)){
          if(!has){ids.push(String(did));tasks.push(api.linkDevices(r.id,ids))}
        }else if(has){
          var nids=ids.filter(function(x){return Number(x)!==Number(did)})
          tasks.push(api.linkDevices(r.id,nids))
        }
      })
      Promise.all(tasks).then(function(){
        wx.hideLoading();wx.showToast({title:"已分配到该房间"});self.load()
      }).catch(function(){wx.hideLoading();wx.showToast({title:"分配失败",icon:"none"});self.load()})
    }).catch(function(){wx.hideLoading();wx.showToast({title:"获取房间失败",icon:"none"})})
  },
  sendCmd(e){if(this._sending)return;this._sending=true;var self=this,id=e.currentTarget.dataset.id,cmd=e.currentTarget.dataset.cmd;api.controlDevice(id,cmd).then(function(){wx.showToast({title:'指令已发送'});setTimeout(function(){self._sending=false},1000)}).catch(function(err){self._sending=false;wx.showToast({title:(err&&err.message)||'设备控制失败',icon:'none'})})},
  editDev(e){var id=e.currentTarget.dataset.id;var oldName=e.currentTarget.dataset.name;var self=this;wx.showModal({title:'修改名称',editable:true,placeholderText:oldName,content:'',success:function(res){if(res.confirm&&res.content){api.updateDevice(id,{name:res.content}).then(function(){self.load()}).catch(function(){wx.showToast({title:'失败',icon:'none'})})}}})},
  delDev(e){var id=e.currentTarget.dataset.id;var self=this;wx.showModal({title:'确认删除',content:'删除后不可恢复',success:function(res){if(!res.confirm)return;api.deleteDevice(id).then(function(){wx.showToast({title:'已删除'});self.load()}).catch(function(){wx.showToast({title:'失败',icon:'none'})})}})},
  toggleUserControl(e){var self=this;var id=Number(e.currentTarget.dataset.id);var enabled=!!e.detail.value;api.updateDevice(id,{user_control_enabled:enabled}).then(function(){wx.showToast({title:enabled?'用户端控制已开启':'用户端控制已关闭'});self.load()}).catch(function(err){wx.showToast({title:(err&&err.message)||'权限更新失败',icon:'none'});self.load()})},
  toggleAutoClose(e){var self=this;var id=Number(e.currentTarget.dataset.id);var isOn=e.currentTarget.dataset.close==='1';var current=null;(this.data.groups||[]).some(function(g){return(g.rooms||[]).some(function(r){var d=(r.devices||[]).find(function(x){return Number(x.id)===id});if(d){current=d;return true}return false})});var cfg={};try{cfg=JSON.parse(current&&current.config_json||'{}')||{}}catch(err){}
    if(isOn){wx.showModal({title:'预约结束自动关闭',content:'关闭后，到时间设备不会自动关',confirmText:'关闭',success:function(r){if(!r.confirm)return;cfg.auto_close=false;api.updateDevice(id,{config_json:JSON.stringify(cfg)}).then(function(){self.load()}).catch(function(err){wx.showToast({title:(err&&err.message)||'更新失败',icon:'none'})})}});return}
    var cur=Number(cfg.auto_close_delay_min)||0
    wx.showModal({title:'到点自动关闭',editable:true,content:'',placeholderText:'到点后多少分钟关闭（当前 '+cur+'，留空为0）',success:function(r){
      if(!r.confirm)return
      var s=(r.content||'').trim()
      var delay=s===''?0:parseInt(s,10)
      if(isNaN(delay)||delay<0||delay>1440){wx.showToast({title:'请输入 0-1440 的分钟数',icon:'none'});return}
      cfg.auto_close=true;cfg.auto_close_delay_min=delay
      api.updateDevice(id,{config_json:JSON.stringify(cfg)}).then(function(){wx.showToast({title:'已开启'});self.load()}).catch(function(err){wx.showToast({title:(err&&err.message)||'更新失败',icon:'none'})})
    }})}
})
