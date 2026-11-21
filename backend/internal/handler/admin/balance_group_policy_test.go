package admin

import (
	"encoding/json"
	"testing"
)

func TestFlexIDUnmarshal(t *testing.T) {
	var f flexID
	if err := json.Unmarshal([]byte(`"4677101"`), &f); err != nil || uint64(f) != 4677101 {
		t.Fatalf("字符串形式失败: %v err=%v", f, err)
	}
	f = 0
	if err := json.Unmarshal([]byte(`4677101`), &f); err != nil || uint64(f) != 4677101 {
		t.Fatalf("数字形式失败: %v err=%v", f, err)
	}
	f = 99
	if err := json.Unmarshal([]byte(`0`), &f); err != nil || uint64(f) != 0 {
		t.Fatalf("零值失败: %v err=%v", f, err)
	}
}

func TestStoreDeleteBlockedByOrphanGroup(t *testing.T) {
	// 组内仅此门店且组内有余额 → 禁止删除
	if !storeDeleteBlockedByBalance(true, true) {
		t.Fatalf("仅此门店且有余额时应禁止删除")
	}
	// 组内还有其它门店 → 允许（余额不变）
	if storeDeleteBlockedByBalance(false, true) {
		t.Fatalf("组内还有其它门店时应允许删除")
	}
	// 无余额 → 允许
	if storeDeleteBlockedByBalance(true, false) {
		t.Fatalf("无余额时应允许删除")
	}
}
