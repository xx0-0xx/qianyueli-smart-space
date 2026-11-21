package handler

import (
	"testing"

	"smart-space-management/backend/internal/model/device"
)

func TestDeviceStatusResponseFallsBackToDatabaseOnlineStatus(t *testing.T) {
	dev := device.Device{Name: "Door", Type: device.TypeDoorLock, Status: device.StatusOnline}
	got := deviceStatusResponse(dev, map[string]interface{}{"rssi": float64(-55)})
	if got["online"] != true {
		t.Fatalf("online = %#v, want true from database status", got["online"])
	}
}

func TestDeviceStatusResponsePrefersCurrentDatabaseStatusOverStaleCache(t *testing.T) {
	dev := device.Device{Name: "Door", Type: device.TypeDoorLock, Status: device.StatusOnline}
	got := deviceStatusResponse(dev, map[string]interface{}{"online": false})
	if got["online"] != true {
		t.Fatalf("online = %#v, want current database status true", got["online"])
	}

	dev.Status = device.StatusOffline
	got = deviceStatusResponse(dev, map[string]interface{}{"online": true})
	if got["online"] != false {
		t.Fatalf("online = %#v, want current database status false", got["online"])
	}
}
