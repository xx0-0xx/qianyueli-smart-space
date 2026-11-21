package service

import (
	"encoding/json"
	"fmt"
	"log"

	"smart-space-management/backend/config"
	devicePkg "smart-space-management/backend/internal/device"
	cfgModel "smart-space-management/backend/internal/model/config"
	"smart-space-management/backend/internal/model/device"
	"smart-space-management/backend/internal/model/resource"

	"gorm.io/gorm"
)

// DeviceService 设备业务逻辑
type DeviceService struct {
	DB    *gorm.DB
	Voice config.VoiceConfig
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *gorm.DB, voice config.VoiceConfig) *DeviceService {
	return &DeviceService{DB: db, Voice: voice}
}

// GetVoiceConfig 读取管理后台保存的设备语音配置；不回退到旧全局配置。
func GetVoiceConfig(db *gorm.DB, _ config.VoiceConfig) config.VoiceConfig {
	vc := config.VoiceConfig{Devices: []config.DeviceVoiceConfig{}}
	var uc cfgModel.UIConfig
	if db.Where("store_id = 0 AND `key` = ?", "voice_config").First(&uc).Error != nil || uc.Value == "" {
		return vc
	}
	if json.Unmarshal([]byte(uc.Value), &vc) != nil || vc.Devices == nil {
		return config.VoiceConfig{Devices: []config.DeviceVoiceConfig{}}
	}
	return vc
}

func voiceCommandForDeviceEvent(cfg config.DeviceVoiceConfig, event string) string {
	if !cfg.Enabled {
		return ""
	}
	switch event {
	case "session_start":
		return cfg.OnStart
	case "session_end":
		return cfg.OnEnd
	default:
		return ""
	}
}

// SendSessionCmd 向资源关联的设备发送开始/结束指令
func (s *DeviceService) SendSessionCmd(resourceID uint64, cmd, serverURL string) {
	var res resource.Resource
	if s.DB.First(&res, resourceID).Error != nil {
		return
	}
	var ids []uint64
	if res.DeviceIDs != "" {
		var err error
		if ids, err = resource.ParseDeviceIDs(res.DeviceIDs); err != nil {
			log.Printf("[DEVICE] parse device_ids resource=%d: %v", resourceID, err)
			return
		}
	}
	if len(ids) == 0 {
		return
	}

	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		return
	}

	var devices []device.Device
	s.DB.Where("id IN ?", ids).Find(&devices)
	if len(devices) == 0 {
		return
	}

	// 音频文件播报保持原行为。
	s.playAudioForSN(devices[0].DeviceSN, cmd, serverURL)

	// 开机语音改由用户在设备控制页点击开关触发；结束语音仍按订单结束发送。
	if cmd != "session_start" {
		s.sendVoiceForDevices(devices, cmd)
	}

	// 设备电源控制
	for _, dev := range devices {
		deviceCmd := s.resolveDeviceCmd(dev, cmd)
		if deviceCmd == "" {
			continue
		}
		if dev.Protocol == device.ProtocolASCII {
			cli.PublishRaw(dev.DeviceSN, deviceCmd)
		} else {
			cli.PublishCommand(dev.DeviceSN, deviceCmd, nil)
		}
	}
}

// SendVoiceForResource 只向当前房间绑定的语音设备发送语音。
func (s *DeviceService) SendVoiceForResource(resourceID uint64, event string) {
	var res resource.Resource
	if s.DB.First(&res, resourceID).Error != nil {
		return
	}
	ids, err := resource.ParseDeviceIDs(res.DeviceIDs)
	if err != nil {
		return
	}
	var devices []device.Device
	s.DB.Where("id IN ?", ids).Find(&devices)
	s.sendVoiceForDevices(devices, event)
}

func (s *DeviceService) sendVoiceForDevices(devices []device.Device, event string) {
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		return
	}
	vc := GetVoiceConfig(s.DB, s.Voice)
	voiceBySN := make(map[string]config.DeviceVoiceConfig, len(vc.Devices))
	for _, voiceDevice := range vc.Devices {
		voiceBySN[voiceDevice.DeviceSN] = voiceDevice
	}
	seenVoiceSN := make(map[string]bool)
	for _, dev := range devices {
		if dev.Protocol != device.ProtocolASCII || seenVoiceSN[dev.DeviceSN] {
			continue
		}
		seenVoiceSN[dev.DeviceSN] = true
		voiceCmd := voiceCommandForDeviceEvent(voiceBySN[dev.DeviceSN], event)
		if voiceCmd != "" {
			log.Printf("[VOICE] publish sn=%s event=%s cmd=%s", dev.DeviceSN, event, voiceCmd)
			if err := cli.PublishRaw(dev.DeviceSN, voiceCmd); err != nil {
				// 语音失败不阻断房间设备电源控制
				log.Printf("[VOICE] publish failed sn=%s cmd=%s: %v", dev.DeviceSN, voiceCmd, err)
			}
		}
	}
}

// SendVoiceToDevice 按指定 SN 给单台成品控制器发送语音播报。
func (s *DeviceService) SendVoiceToDevice(deviceSN, voiceCmd string) error {
	if deviceSN == "" || voiceCmd == "" {
		return nil
	}
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		return fmt.Errorf("MQTT client 未就绪")
	}
	return cli.PublishRaw(deviceSN, voiceCmd)
}

// resolveDeviceCmd 根据设备类型和 session 指令解析具体设备命令
// 优先使用设备 config_json 里的 cmd_on/cmd_off（成品控制器/自定义映射）
func (s *DeviceService) resolveDeviceCmd(dev device.Device, cmd string) string {
	var cfg struct {
		CmdOn     string `json:"cmd_on"`
		CmdOff    string `json:"cmd_off"`
		AutoClose bool   `json:"auto_close"`
	}
	if dev.ConfigJSON != "" {
		json.Unmarshal([]byte(dev.ConfigJSON), &cfg)
	}

	if cmd == "session_start" {
		if cfg.CmdOn != "" {
			return cfg.CmdOn
		}
		switch dev.Type {
		case "light", "other":
			return "light_on"
		case "ac":
			return "ac_on"
		case "exhaust":
			return "exhaust_on"
		default:
			return ""
		}
	}

	// session_end：仅 auto_close 设备才关闭
	if !cfg.AutoClose {
		return ""
	}
	if cfg.CmdOff != "" {
		return cfg.CmdOff
	}
	switch dev.Type {
	case "light", "other":
		return "light_off"
	case "ac":
		return "ac_off"
	case "exhaust":
		return "exhaust_off"
	case "door_lock":
		return "door_close"
	default:
		return "light_off"
	}
}

// playAudioForSN 通过设备 SN 查询音频配置并下发 MQTT 播报
func (s *DeviceService) playAudioForSN(deviceSN, cmd, serverURL string) {
	var cfgDev device.Device
	if s.DB.Where("device_sn = ?", deviceSN).First(&cfgDev).Error != nil {
		return
	}

	type audioCfg struct {
		StartAudio string `json:"start_audio"`
		EndAudio   string `json:"end_audio"`
	}
	var ac audioCfg
	if cfgDev.ConfigJSON != "" {
		json.Unmarshal([]byte(cfgDev.ConfigJSON), &ac)
	}

	var audioFile string
	if cmd == "session_start" {
		audioFile = ac.StartAudio
	} else {
		audioFile = ac.EndAudio
	}
	if audioFile == "" {
		return
	}

	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		return
	}

	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	audioURL := serverURL + "/api/v1/audio/" + audioFile
	cli.PublishCommand(deviceSN, "speaker_mp3", map[string]interface{}{
		"url": audioURL,
	})
}
