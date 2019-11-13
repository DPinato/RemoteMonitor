// runs the backend services required for a device to register, check-in and communicate with the central service

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Device struct {
	Name string `json:"name"` // name a device identified itself with
	Key  string `json:"key"`  // key the device will use to authenticate itself with backend-api
	Mac  string `json:"mac"`  // MAC address of any interface provided by the device
}

type ErrorMsg struct {
	MsgString string `json:"error"`
	Code      uint32 `json:"code"`
}

// list of generic response codes / errors that may be returned after any HTTP request
const (
	RequestOK uint32 = 0 // request received was ok
	BadJSON   uint32 = 1 // device provided bad JSON
)

// list of errors codes that may be returned when the device registers
const (
	AlreadyRegistered  uint32 = 101
	MissingInformation uint32 = 102 // device did not provide all the necessary information when registering
	BadDeviceName      uint32 = 103 // device should not attempt to register with this name
	BadDeviceMac       uint32 = 104 // device should not attempt to register with this MAC
	TooManyDevices     uint32 = 105 // device should not attempt to register anymore
)

// list of errors codes that may be returned when the device registers
const (
	BadKey uint32 = 1
)

var deviceList []Device

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/register", registerDevice).Methods("POST")
	router.HandleFunc("/checkin", checkInDevice).Methods("POST")

	http.ListenAndServe(":8000", router)
}

////////////
// respond to HTTP calls from client
func registerDevice(w http.ResponseWriter, r *http.Request) {
	// allows a device to register with this API
	// the device provides a name, a key is provided to the device which must be used for any other call
	w.Header().Set("Content-Type", "application/json")

	// check whether the request body has proper JSON and has all the information required
	tmpDev, code := readRegisterRequestBody(r.Body)

	if code != RequestOK {
		var response string
		if code == MissingInformation {
			response, _ = generateErrorResponse("Information missing", MissingInformation)
		} else if code == BadJSON {
			response, _ = generateErrorResponse("Bad JSON", BadJSON)
		}

		log.Println("Received bad or incomplete request, " + response)
		http.Error(w, response, http.StatusBadRequest)
		return
	}

	// check if device is already in the list
	index := findDeviceInList(deviceList, tmpDev)
	if index != -1 {
		log.Println(deviceList[index].Name + " (" + deviceList[index].Mac + ")" + " attempted to register again")
		response, _ := generateErrorResponse("Device already registered", AlreadyRegistered)
		http.Error(w, response, http.StatusBadRequest)
	} else {
		// generate a key for this device
		// TODO: this is not very secure yet, it just takes the MD5 hash of the MAC + Name
		tmpDev.Key = BytesToString(sha256.Sum256([]byte(tmpDev.Mac + tmpDev.Name)))
		log.Println("Key for " + tmpDev.Name + "\t" + tmpDev.Key)

		// add it to the list and send a response back with the key
		deviceList = append(deviceList, tmpDev)
		json.NewEncoder(w).Encode(&tmpDev)

	}

	debug_dumpDeviceList(deviceList) // just for DEBUG
}

func checkInDevice(w http.ResponseWriter, r *http.Request) {
	// check-in endpoint for device
	// device has to provide its key as an authentication mechanism
	w.Header().Set("Content-Type", "application/json")
	// tmpDev, code := readRegisterRequestBody(r.Body)

}

/////////////
// helpful functions for API calls
func readRegisterRequestBody(body io.ReadCloser) (Device, uint32) {
	// check if the HTTP request body received from registerDevice has all the necessary parameters
	// return an error if either JSON is bad or if MAC / name of device is missing
	var tmpDev Device
	var err error
	err = json.NewDecoder(body).Decode(&tmpDev)
	if err == nil {
		// request not malformed, check if all the necessary parameters are there
		if tmpDev.Name == "" {
			return tmpDev, MissingInformation
		}

		if tmpDev.Mac == "" {
			err = fmt.Errorf("Missing device MAC")
			return tmpDev, MissingInformation
		}
	} else {
		// request malformed
		return tmpDev, BadJSON
	}

	return tmpDev, RequestOK
}

// func readCheckinRequestBody(body io.ReadCloser) (Device, uint32) {
//
//
// 	var tmpDev Device
// 	var err error
// 	err = json.NewDecoder(body).Decode(&tmpDev)
// 	if err == nil {
//
// 	} else {
// 		// request malformed
// 		return tmpDev,
// 	}
//
// }

func generateErrorResponse(msg string, code uint32) (string, error) {
	// generate a proper response message in JSON format
	var tmpError ErrorMsg
	tmpError.MsgString = msg
	tmpError.Code = code

	jsonData, err := json.Marshal(tmpError)
	if err != nil {
		log.Println("generateErrorResponse failed to marshal JSON")
		log.Println("msg: " + msg)
		log.Println("code: " + strconv.Itoa(int(code)))
		return "", nil
	}

	return string(jsonData), err
}

/////////////
/////////////
// generic helper functions
func findDeviceInList(list []Device, dev Device) int {
	// find device dev in the list, check for matching MAC addresses
	for i := 0; i < len(list); i++ {
		if list[i].Mac == dev.Mac {
			return i
		}
	}
	return -1 // not in list
}

func BytesToString(data [32]byte) string {
	return fmt.Sprintf("%x", data)
}

/////////////
/////////////
// debug functions
func debug_dumpDeviceList(list []Device, index ...int) {
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
