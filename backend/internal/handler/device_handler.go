package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	devicePkg "smart-space-management/backend/internal/device"
	"smart-space-management/backend/internal/model/device"
	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/ws"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	ctx *Context
}

var doorAutoClose = newDoorAutoCloseManager(func(delay time.Duration, fn func()) doorTimer {
	return time.AfterFunc(delay, fn)
})

func NewDeviceHandler(ctx *Context) *DeviceHandler {
	return &DeviceHandler{ctx: ctx}
}

func merchantDeviceStoreAllowed(role string, currentStoreID, deviceStoreID uint64) bool {
	if role == "admin" || role == "super_admin" {
		return true
	}
	return role == "merchant" && currentStoreID > 0 && currentStoreID == deviceStoreID
}

func defaultUserControlEnabled(deviceType device.DeviceType) bool {
	return deviceType == device.TypeDoorLock
}

func userCanControlDevice(dev device.Device, hasReservation bool) bool {
	return dev.UserControlEnabled && hasReservation
}

// canControlWithoutReservation 平台管理员、以及从商家端登录的商家可不受预约时间限制。
// 同样是 merchant 角色，从用户端登录时 client=user，仍必须校验预约时间。
func canControlWithoutReservation(role, client string) bool {
	if role == "admin" || role == "super_admin" {
		return true
	}
	return role == "merchant" && client == "merchant"
}

func userControlCommandAllowed(dev device.Device, cmd string) bool {
	cfg := readDeviceCommandConfig(dev.ConfigJSON)
	if cfg.CmdOn != "" || cfg.CmdOff != "" {
		return cmd == cfg.CmdOn || cmd == cfg.CmdOff
	}
	allowed := map[device.DeviceType]map[string]bool{
		device.TypeDoorLock: {"door_open": true, "door_close": true},
		device.TypeLight:    {"light_on": true, "light_off": true},
		device.TypeAC:       {"ac_on": true, "ac_off": true},
		device.TypeExhaust:  {"exhaust_on": true, "exhaust_off": true},
		device.TypeMachine:  {"machine_on": true, "machine_off": true},
		device.TypeOther:    {"door_open": true, "door_close": true, "light_on": true, "light_off": true},
	}
	return allowed[dev.Type][cmd]
}

type deviceCommandConfig struct {
	CmdOn    string `json:"cmd_on"`
	CmdOff   string `json:"cmd_off"`
	IsToggle *bool  `json:"is_toggle"`
}

func readDeviceCommandConfig(raw string) deviceCommandConfig {
	var cfg deviceCommandConfig
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	return cfg
}

// ControlDeviceReq 设备控制请求
type ControlDeviceReq struct {
	Cmd    string                 `json:"cmd" binding:"required"` // door_open, light_on, ac_off ...
	Params map[string]interface{} `json:"params"`                 // 附加参数
}

// ControlDevice 远程控制设备 — 需校验预约时段权限
// POST /api/v1/devices/:id/control
func (h *DeviceHandler) scheduleDoorAutoClose(deviceID uint64) {
	doorAutoClose.schedule(deviceID, 3*time.Second, func() {
		var dev device.Device
		if err := h.ctx.DB.First(&dev, deviceID).Error; err != nil || dev.Type != device.TypeDoorLock {
			return
		}
		cmd := doorCloseCommand(dev)
		if cmd == "" {
			return
		}
		cli := devicePkg.GetGlobalClient()
		if cli == nil {
			log.Printf("[DEVICE] 门锁自动关门失败 device=%d: MQTT client 未就绪", deviceID)
			return
		}
		var err error
		if dev.Protocol == device.ProtocolASCII {
			err = cli.PublishRaw(dev.DeviceSN, cmd)
		} else {
			err = cli.PublishCommand(dev.DeviceSN, cmd, nil)
		}
		if err != nil {
			log.Printf("[DEVICE] 门锁自动关门失败 device=%d: %v", deviceID, err)
			return
		}
		log.Printf("[DEVICE] 门锁自动关门已发送 device=%d command=%s", deviceID, cmd)
	})
}

func (h *DeviceHandler) activeResourceForDevice(userID uint64, dev device.Device) (uint64, bool) {
	idStr := strconv.FormatUint(dev.ID, 10)
	var rooms []resource.Resource
	h.ctx.DB.Where("store_id = ? AND device_ids LIKE ?", dev.StoreID, "%"+idStr+"%").Find(&rooms)

	now := time.Now().UnixMilli()
	for _, room := range rooms {
		var count int64
		h.ctx.DB.Model(&order.Order{}).
			Where("user_id = ? AND resource_id = ? AND status IN ? AND start_time <= ? AND end_time > ?",
				userID, room.ID,
				[]order.OrderStatus{order.StatusPaid, order.StatusInProgress}, now, now).
			Count(&count)
		if count > 0 {
			return room.ID, true
		}
	}
	return 0, false
}

func (h *DeviceHandler) ControlDevice(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var req ControlDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	userID := GetUserID(c)

	// ── 查询设备 ──
	var dev device.Device
	if h.ctx.DB.First(&dev, deviceID).Error != nil {
		response.FailCode(c, errcode.ErrDeviceNotFound)
		return
	}
	// MVP: 跳过在线检查，允许离线发送（ESP32 可能断连但不影响指令队列）

	// ── 权限校验：商家端/管理员直接放行；用户端 token 只能控制自己预订房间内的设备 ──
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	clientValue, _ := c.Get("client")
	client, _ := clientValue.(string)
	if role == "merchant" && client == "merchant" {
		if !merchantDeviceStoreAllowed(role, GetStoreID(c), dev.StoreID) {
			response.FailCode(c, errcode.ErrNoControlPerm)
			return
		}
	} else if !canControlWithoutReservation(role, client) {
		if !userCanControlDevice(dev, h.hasRoomControlPermission(userID, dev.StoreID, dev.ID)) || !userControlCommandAllowed(dev, req.Cmd) {
			response.FailCode(c, errcode.ErrNoControlPerm)
			return
		}
	}

	// ── 通过 MQTT 下发指令 ──
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		response.FailMsg(c, errcode.ErrDeviceCmdFail, "设备服务未连接")
		return
	}
	var pubErr error
	if dev.Protocol == device.ProtocolASCII {
		// 成品控制器：发纯文本 ASCII 指令（如 "a1" / "mp3=00001"）
		pubErr = cli.PublishRaw(dev.DeviceSN, req.Cmd)
	} else {
		// 自制固件：发 JSON {"cmd":"..."}
		pubErr = cli.PublishCommand(dev.DeviceSN, req.Cmd, req.Params)
	}
	if pubErr != nil {
		response.FailMsg(c, errcode.ErrDeviceCmdFail, fmt.Sprintf("指令下发失败: %v", pubErr))
		return
	}

	if doorOpenCommand(dev, req.Cmd) {
		h.scheduleDoorAutoClose(dev.ID)
	} else if doorManualCloseCommand(dev, req.Cmd) {
		doorAutoClose.cancel(dev.ID)
	}

	// 用户在设备控制页点击任意开关时，只触发当前房间的语音开机播报（3 秒防抖）。
	if role == "user" {
		if resourceID, ok := h.activeResourceForDevice(userID, dev); ok {
			key := fmt.Sprintf("ssm:voice:userstart:%d:%d", userID, resourceID)
			if ok, _ := h.ctx.Redis.SetNX(c.Request.Context(), key, "1", 3*time.Second).Result(); ok {
				go h.ctx.DeviceSvc.SendVoiceForResource(resourceID, "session_start")
			}
		}
	}

	response.OK(c, gin.H{
		"device_id": dev.ID,
		"cmd":       req.Cmd,
		"status":    "sent",
	})
}

// GetDeviceStatus 获取设备实时状态
func (h *DeviceHandler) GetDeviceStatus(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	// 先查设备基本信息获取 device_sn
	var dev device.Device
	if h.ctx.DB.First(&dev, deviceID).Error != nil {
		response.FailCode(c, errcode.ErrDeviceNotFound)
		return
	}

	// 与设备控制相同的客户端范围校验，避免用户端 token 在预约结束后继续读取设备状态。
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	clientValue, _ := c.Get("client")
	client, _ := clientValue.(string)
	if role == "merchant" && client == "merchant" {
		if !merchantDeviceStoreAllowed(role, GetStoreID(c), dev.StoreID) {
			response.FailCode(c, errcode.ErrNoControlPerm)
			return
		}
	} else if !canControlWithoutReservation(role, client) {
		if !h.hasRoomControlPermission(GetUserID(c), dev.StoreID, dev.ID) {
			response.FailCode(c, errcode.ErrNoControlPerm)
			return
		}
	}

	// 从 Redis 读取最新设备状态 (按 device_sn)
	key := fmt.Sprintf("ssm:device:status:%s", dev.DeviceSN)
	val, err := h.ctx.Redis.Get(c.Request.Context(), key).Result()
	if err != nil {
		response.OK(c, deviceStatusResponse(dev, nil))
		return
	}

	var status map[string]interface{}
	json.Unmarshal([]byte(val), &status)
	response.OK(c, deviceStatusResponse(dev, status))
}

func deviceStatusResponse(dev device.Device, status map[string]interface{}) gin.H {
	if status == nil {
		status = make(map[string]interface{})
	}
	// DB 状态由 MQTT 上报/上下线事件更新，可能滞后；设备可能保持连接但上报稀疏，
	// 故用「最近上报时间（10 分钟宽限）」兜底判定在线，避免"连着但静默"被误显示离线。
	online := dev.Status == device.StatusOnline
	if !online && dev.LastOnlineAt != nil {
		if *dev.LastOnlineAt > time.Now().Add(-10*time.Minute).UnixMilli() {
			online = true
		}
	}
	status["online"] = online
	status["name"] = dev.Name
	status["type"] = dev.Type
	status["user_control_enabled"] = dev.UserControlEnabled
	status["device_id"] = dev.ID
	status["device_sn"] = dev.DeviceSN
	if dev.UserControlEnabled {
		commands := readDeviceCommandConfig(dev.ConfigJSON)
		status["cmd_on"] = commands.CmdOn
		status["cmd_off"] = commands.CmdOff
		status["is_toggle"] = commands.IsToggle
	} else {
		delete(status, "cmd_on")
		delete(status, "cmd_off")
		delete(status, "is_toggle")
	}
	return gin.H(status)
}

// hasRoomControlPermission 校验用户是否有该设备的控制权限（房间级）
// 规则：
//  1. 设备必须绑定在「用户有效订单所属房间」上
//  2. 允许提前 15 分钟开始控制（预约开始前 15 分钟内可开门/开电）
//  3. 若该房间当前正被他人使用（有进行中订单），不允许提前控制，避免打断他人
func (h *DeviceHandler) hasRoomControlPermission(userID, storeID, deviceID uint64) bool {
	if storeID == 0 {
		return false
	}
	did := strconv.FormatUint(deviceID, 10)
	// 1. 找包含该设备的房间（device_ids 是 JSON 数组字符串，如 ["123","456"]）
	var rooms []resource.Resource
	h.ctx.DB.Where("store_id = ? AND (device_ids LIKE ? OR device_ids LIKE ?)",
		storeID, "%\""+did+"\"%", "%["+did+"%").Find(&rooms)
	if len(rooms) == 0 {
		return false
	}
	roomIDs := make([]uint64, 0, len(rooms))
	for _, r := range rooms {
		roomIDs = append(roomIDs, r.ID)
	}
	now := time.Now().UnixMilli()
	// 2. 查用户的有效订单（允许提前 15 分钟 = 900000ms）
	var myOrders []order.Order
	h.ctx.DB.Where("user_id = ? AND resource_id IN ? AND status IN ? AND start_time - 900000 <= ? AND end_time > ?",
		userID, roomIDs, []order.OrderStatus{order.StatusPaid, order.StatusInProgress}, now, now).Find(&myOrders)
	if len(myOrders) == 0 {
		return false
	}
	// 3. 房间当前若被他人占用，不允许提前控制
	for _, o := range myOrders {
		var busy int64
		h.ctx.DB.Model(&order.Order{}).
			Where("resource_id = ? AND status IN (1,2) AND id <> ? AND start_time <= ? AND end_time > ?",
				o.ResourceID, o.ID, now, now).Count(&busy)
		if busy == 0 {
			return true // 该房间当前空闲，可提前控制
		}
	}
	return false // 相关房间都还在使用中
}

// hasControlPermission 校验用户是否有当前设备的控制权限
// 规则：用户在该门店有当前时段内的有效订单
func (h *DeviceHandler) hasControlPermission(userID, storeID uint64) bool {
	if storeID == 0 {
		return false
	}

	now := time.Now().UnixMilli()
	var count int64
	h.ctx.DB.Model(&order.Order{}).
		Where("user_id = ? AND store_id = ? AND status = ? AND start_time - 300000 <= ? AND end_time > ?",
			userID, storeID, order.StatusPaid, now, now).
		Count(&count)

	return count > 0
}

// ListMerchantDevices 商家查看本店所有设备
// GET /api/v1/merchant/devices
func (h *DeviceHandler) ListMerchantDevices(c *gin.Context) {
	storeID := GetStoreID(c)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var devices []device.Device
	h.ctx.DB.Where("store_id = ? AND type <> ?", storeID, device.TypeSpeaker).
		Order("created_at ASC").Find(&devices)

	response.OK(c, devices)
}

// CreateDevice 商家创建新设备
// 支持 template_id：按产品模板批量生成子设备（同一 SN，指令不同）
// 目标房间必选，创建后自动绑定到房间。
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	storeID := GetStoreID(c)
	var req struct {
		device.Device
		RoomID uint64 `json:"room_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	d := req.Device
	d.DeviceSN = strings.TrimSpace(d.DeviceSN)
	if d.DeviceSN == "" {
		response.FailMsg(c, errcode.ErrBadRequest, "请输入设备 SN")
		return
	}
	if req.RoomID == 0 {
		response.FailMsg(c, errcode.ErrBadRequest, "请选择目标房间")
		return
	}
	var room resource.Resource
	if h.ctx.DB.Where("id = ? AND store_id = ?", req.RoomID, storeID).First(&room).Error != nil {
		response.FailMsg(c, errcode.ErrBadRequest, "目标房间不存在")
		return
	}

	// 产品模板模式：MQTTTopic 字段复用为 template_id 传输
	var templateID uint64
	if d.MQTTTopic != "" {
		templateID, _ = strconv.ParseUint(d.MQTTTopic, 10, 64)
	}

	d.StoreID = storeID

	// 云喇叭按 SN 唯一，避免重复绑定生成多条设备记录。
	isSpeaker := d.Type == device.TypeSpeaker
	if templateID > 0 {
		var check device.ProductTemplate
		if h.ctx.DB.First(&check, templateID).Error == nil {
			subDevices := check.Devices()
			if len(subDevices) == 0 {
				isSpeaker = check.Type == string(device.TypeSpeaker)
			} else {
				for _, td := range subDevices {
					if td.Type == string(device.TypeSpeaker) {
						isSpeaker = true
						break
					}
				}
			}
		}
	}
	if isSpeaker {
		var existing int64
		if err := h.ctx.DB.Model(&device.Device{}).
			Where("store_id = ? AND device_sn = ? AND type = ?", storeID, d.DeviceSN, device.TypeSpeaker).
			Count(&existing).Error; err != nil {
			response.FailCode(c, errcode.ErrInternal)
			return
		}
		if existing > 0 {
			response.FailMsg(c, errcode.ErrConflict, "该云喇叭已绑定，请勿重复添加")
			return
		}
	}

	var newIDs []uint64
	var created []device.Device

	tx := h.ctx.DB.Begin()
	if templateID > 0 {
		var t device.ProductTemplate
		if tx.First(&t, templateID).Error != nil {
			tx.Rollback()
			response.FailCode(c, errcode.ErrNotFound)
			return
		}
		subDevices := t.Devices()
		created = make([]device.Device, 0, len(subDevices))
		if len(subDevices) == 0 {
			// 无子设备模板（如云喇叭）：按模板生成一台独立设备。
			typ := device.DeviceType(t.Type)
			if typ == "" {
				typ = device.TypeOther
			}
			sub := device.Device{
				StoreID: storeID, DeviceSN: d.DeviceSN, Name: t.Name, Type: typ, Protocol: t.Protocol,
				UserControlEnabled: defaultUserControlEnabled(typ),
				ConfigJSON:         fmt.Sprintf(`{"template_id":%d}`, templateID),
			}
			sub.ID = utils.ShortID()
			if err := tx.Create(&sub).Error; err != nil {
				tx.Rollback()
				response.FailMsg(c, errcode.ErrConflict, "创建失败")
				return
			}
			created = append(created, sub)
			newIDs = append(newIDs, sub.ID)
		} else {
			for _, td := range subDevices {
				typ := device.DeviceType(td.Type)
				sub := device.Device{
					StoreID: storeID, DeviceSN: d.DeviceSN, Name: td.Name, Type: typ, Protocol: t.Protocol,
					UserControlEnabled: defaultUserControlEnabled(typ),
					ConfigJSON: fmt.Sprintf(`{"gpio":%d,"cmd_on":"%s","cmd_off":"%s","is_toggle":%v,"auto_close":false,"template_id":%d}`,
						td.GPIO, td.CmdOn, td.CmdOff, td.IsToggle, templateID),
				}
				sub.ID = utils.ShortID()
				if err := tx.Create(&sub).Error; err != nil {
					tx.Rollback()
					response.FailMsg(c, errcode.ErrConflict, "创建失败")
					return
				}
				created = append(created, sub)
				newIDs = append(newIDs, sub.ID)
			}
		}
	} else {
		// 普通模式：直接创建设备。未显式启用时遵循类型安全默认值。
		if d.Name == "" {
			d.Name = d.DeviceSN
		}
		d.ID = utils.ShortID()
		d.UserControlEnabled = defaultUserControlEnabled(d.Type)
		if err := tx.Create(&d).Error; err != nil {
			tx.Rollback()
			response.FailMsg(c, errcode.ErrConflict, "创建失败")
			return
		}
		created = append(created, d)
		newIDs = append(newIDs, d.ID)
	}

	// 把新设备追加到房间，避免前端二次调用失败后设备处于未绑定状态。
	ids, err := resource.ParseDeviceIDs(room.DeviceIDs)
	if err != nil {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrBadRequest, "房间设备数据异常，请刷新后重试")
		return
	}
	seen := make(map[uint64]bool, len(ids)+len(newIDs))
	for _, id := range ids {
		seen[id] = true
	}
	for _, id := range newIDs {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	rawIDs, _ := json.Marshal(ids)
	if err := tx.Model(&resource.Resource{}).Where("id = ?", room.ID).
		Update("device_ids", string(rawIDs)).Error; err != nil {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrInternal, "设备绑定房间失败")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "创建失败")
		return
	}

	if templateID > 0 {
		response.Created(c, created)
		return
	}
	response.Created(c, created[0])
}

// ListProductTemplates 商家查看产品模板列表
// GET /api/v1/merchant/product-templates
func (h *DeviceHandler) ListProductTemplates(c *gin.Context) {
	var list []device.ProductTemplate
	h.ctx.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&list)
	type row struct {
		device.ProductTemplate
		Devices []device.TemplateDevice `json:"devices"`
	}
	var rows []row
	for _, t := range list {
		rows = append(rows, row{ProductTemplate: t, Devices: t.Devices()})
	}
	response.OK(c, rows)
}

// UpdateDevice 商家更新设备
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)
	var d device.Device
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&d).Error != nil {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	var req struct {
		Name               string            `json:"name"`
		Type               device.DeviceType `json:"type"`
		ConfigJSON         string            `json:"config_json"`
		UserControlEnabled *bool             `json:"user_control_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.ConfigJSON != "" {
		updates["config_json"] = req.ConfigJSON
	}
	if req.UserControlEnabled != nil {
		updates["user_control_enabled"] = *req.UserControlEnabled
	}
	if len(updates) == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := h.ctx.DB.Model(&d).Updates(updates).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "已更新")
}

// DeleteDevice 商家删除设备
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)
	var dev device.Device
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&dev).Error != nil {
		response.FailCode(c, errcode.ErrDeviceNotFound)
		return
	}
	h.ctx.DB.Unscoped().Where("id = ? AND store_id = ?", id, storeID).Delete(&device.Device{})
	if dev.Type == device.TypeSpeaker && dev.DeviceSN != "" {
		_ = removeVoiceConfigSN(h.ctx.DB, dev.DeviceSN)
	}
	// 清理所有房间对该设备的引用，避免按 id 查询状态时 404
	idStr := strconv.FormatUint(id, 10)
	var rooms []resource.Resource
	h.ctx.DB.Where("store_id = ? AND device_ids LIKE ?", storeID, "%"+idStr+"%").Find(&rooms)
	for _, rm := range rooms {
		var raw []interface{}
		if json.Unmarshal([]byte(rm.DeviceIDs), &raw) != nil {
			continue
		}
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			var sv string
			switch t := v.(type) {
			case string:
				sv = t
			case float64:
				sv = strconv.FormatInt(int64(t), 10)
			}
			if sv != "" && sv != idStr {
				out = append(out, sv)
			}
		}
		b, _ := json.Marshal(out)
		h.ctx.DB.Model(&resource.Resource{}).Where("id = ?", rm.ID).Update("device_ids", string(b))
	}
	response.OKMsg(c, "已删除")
}

// publishMQTT 发布 MQTT 消息
func (h *DeviceHandler) publishMQTT(topic string, payload []byte) error {
	// 通过全局 MQTT 客户端真实下发指令
	cli := devicePkg.GetGlobalClient()
	if cli != nil {
		// 直接使用原始 topic 和 payload 发布
		return nil // MQTT publish 在 PublishCommand 中处理
	}

	// 无 MQTT 连接时，通过 Redis pub 通知（用于 WS 推送）
	ws.PublishDeviceState(h.ctx.Redis, 0, 0, map[string]interface{}{
		"topic":   topic,
		"payload": string(payload),
	})
	return nil
}

// AudioConfigResp 音频配置响应
type AudioConfigResp struct {
	DeviceSN   string `json:"device_sn"`
	RoomName   string `json:"room_name"`
	ResourceID uint64 `json:"resource_id"`
	StartAudio string `json:"start_audio"`
	EndAudio   string `json:"end_audio"`
}

// ListAudioConfigs 返回本店按 SN 去重的音频配置
func (h *DeviceHandler) ListAudioConfigs(c *gin.Context) {
	storeID := GetStoreID(c)
	role, _ := c.Get("user_role")

	var resources []resource.Resource
	var devs []device.Device
	if role == "super_admin" || role == "admin" {
		h.ctx.DB.Preload("ResourceType").Find(&resources)
		h.ctx.DB.Find(&devs)
	} else {
		h.ctx.DB.Where("store_id = ?", storeID).Preload("ResourceType").Find(&resources)
		h.ctx.DB.Where("store_id = ?", storeID).Find(&devs)
	}

	// 建立 device_id → resource 映射
	type resLink struct {
		Name string
		ID   uint64
	}
	devToResource := make(map[uint64]resLink)
	for _, r := range resources {
		ids, _ := resource.ParseDeviceIDs(r.DeviceIDs)
		for _, did := range ids {
			devToResource[did] = resLink{Name: r.Name, ID: r.ID}
		}
	}

	// 按 SN 去重
	snMap := make(map[string]*AudioConfigResp)
	for _, d := range devs {
		if _, ok := snMap[d.DeviceSN]; ok {
			continue
		}
		cfg := struct {
			StartAudio string `json:"start_audio"`
			EndAudio   string `json:"end_audio"`
		}{}
		if d.ConfigJSON != "" {
			json.Unmarshal([]byte(d.ConfigJSON), &cfg)
		}
		room := devToResource[d.ID]
		snMap[d.DeviceSN] = &AudioConfigResp{
			DeviceSN:   d.DeviceSN,
			RoomName:   room.Name,
			ResourceID: room.ID,
			StartAudio: cfg.StartAudio,
			EndAudio:   cfg.EndAudio,
		}
	}

	var result []*AudioConfigResp
	for _, v := range snMap {
		result = append(result, v)
	}
	response.OK(c, result)
}

// UpdateDeviceAudio 更新设备的音频配置（按 SN 更新该 SN 下所有设备）
type UpdateAudioReq struct {
	StartAudio string `json:"start_audio"`
	EndAudio   string `json:"end_audio"`
}

func (h *DeviceHandler) UpdateDeviceAudio(c *gin.Context) {
	deviceSN := c.Param("sn")
	storeID := GetStoreID(c)

	var req UpdateAudioReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var devs []device.Device
	role, _ := c.Get("user_role")
	if role == "super_admin" || role == "admin" {
		h.ctx.DB.Where("device_sn = ?", deviceSN).Find(&devs)
	} else {
		h.ctx.DB.Where("device_sn = ? AND store_id = ?", deviceSN, storeID).Find(&devs)
	}
	if len(devs) == 0 {
		response.FailMsg(c, errcode.ErrNotFound, "未找到该 SN 的设备")
		return
	}

	for _, d := range devs {
		// 合并到现有 config_json
		var existing map[string]interface{}
		if d.ConfigJSON != "" {
			json.Unmarshal([]byte(d.ConfigJSON), &existing)
		} else {
			existing = make(map[string]interface{})
		}
		existing["start_audio"] = req.StartAudio
		existing["end_audio"] = req.EndAudio
		merged, _ := json.Marshal(existing)
		h.ctx.DB.Model(&d).Update("config_json", string(merged))
	}

	response.OK(c, gin.H{"start_audio": req.StartAudio, "end_audio": req.EndAudio})
}

// SpeakerTest 管理后台测试设备播放指定音频
// POST /api/v1/admin/devices/:sn/speaker-test
func (h *DeviceHandler) SpeakerTest(c *gin.Context) {
	deviceSN := c.Param("sn")
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		// 本地后端可能没连 MQTT，返回成功但不上报
		response.OKMsg(c, "MQTT not available, skipped")
		return
	}
	if err := cli.PublishCommand(deviceSN, "speaker_mp3", map[string]interface{}{"url": req.URL}); err != nil {
		response.FailMsg(c, errcode.ErrDeviceCmdFail, err.Error())
		return
	}
	response.OKMsg(c, "已下发播放指令")
}
