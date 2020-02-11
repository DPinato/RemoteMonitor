package backendapi

import (
	"encoding/json"
	"log"
	"time"
)

// Device is for maintaining devices currently being handled by this backend process
type Device struct {
	Name        string    `json:"name"`         // name a device identified itself with
	Key         string    `json:"key"`          // key the device will use to authenticate itself with backend-api
	Mac         string    `json:"mac"`          // MAC address of any interface provided by the device
	LastCheckin time.Time `json:"last_checkin"` // time when the device last checked in
}

func FindDeviceByMac(list []Device, dev Device) int {
	// find device dev in the list, check for matching MAC addresses
	for i := 0; i < len(list); i++ {
		if list[i].Mac == dev.Mac {
			return i
		}
	}
	return -1 // not in list
}

func FindDeviceByKey(list []Device, dev Device) int {
	// find device in list, check for matching key
	for i := 0; i < len(list); i++ {
		if list[i].Key == dev.Key {
			return i
		}
	}
	return -1 // not in list
}

/////////////
/////////////
// debug functions
func Debug_dumpDeviceList(list []Device, index ...int) {
	// dump the contents of the deviceList slice
	// if index is provided, only show the indexes
	var jsonData []byte
	if len(index) == 0 {
		jsonData, _ = json.MarshalIndent(list, "", "    ")
	} else {
		var tmpList []Device
		for i := 0; i < len(index); i++ {
			tmpList = append(tmpList, list[index[i]])
		}
		jsonData, _ = json.MarshalIndent(tmpList, "", "    ")
	}

	log.Println(string(jsonData))
}
