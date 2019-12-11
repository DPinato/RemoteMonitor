// runs the service on the node to report back to backend-api

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Device struct {
	Name        string    `json:"name"`         // name a device identified itself with
	Key         string    `json:"key"`          // key the device will use to authenticate itself with backend-api
	Mac         string    `json:"mac"`          // MAC address of any interface provided by the device
	LastCheckin time.Time `json:"last_checkin"` // time when the device last checked in
}

var myInfo Device // information device uses to identify itself

func main() {
	serverURL := "http://localhost:8000"
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	myInfo = getMyInfo()
	log.Printf("myInfo: %v\n", myInfo)

	// start by registering with the backend
	registerReq := make(map[string]interface{})
	registerReq["name"] = myInfo.Name
	registerReq["mac"] = myInfo.Mac

	var err error
	var resp *http.Response
	resp, err = register(serverURL+"/register", registerReq, client)
	if err != nil {
		log.Println(err)
	}

	// process register response
	err = processRegisterResponse(resp)
	if err != nil {
		log.Fatal(err)
	}

	// no point in continuing if the key was not obtained, registration should probably be attempted again
	if myInfo.Key == "" {
		log.Fatalf("I don't have a key, exiting ...")
	}

	// start checkin goroutine
	for {
		resp, err = checkin(serverURL+"/checkin", client)
		if err != nil {
			log.Println(err)
		}

		err = processCheckinResponse(resp)
		if err != nil {
			log.Println(err)
		}
		time.Sleep(10 * time.Second)

	}

}

func register(url string, reqBody map[string]interface{}, clientObj *http.Client) (*http.Response, error) {
	// register with the server and obtain key required for any other API call
	requestJson, _ := json.Marshal(reqBody)
	request, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestJson))
	request.Header.Set("Content-type", "application/json")

	resp, err := clientObj.Do(request)
	if err != nil {
		log.Fatalln(err)
	}
	// defer resp.Body.Close()
	return resp, err
}

func checkin(url string, clientObj *http.Client) (*http.Response, error) {
	// check in with the backend
	// only parameter required is the key obtained during registration
	reqBody := map[string]string{"key": myInfo.Key}
	requestJson, _ := json.Marshal(reqBody)
	request, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestJson))
	request.Header.Set("Content-type", "application/json")

	resp, err := clientObj.Do(request)
	if err != nil {
		log.Fatalln(err)
	}
	// defer resp.Body.Close()
	return resp, err
}

///////////////////////
// helper functions
func getMyInfo() Device {
	// collect device name and mac address
	tmpDevice := Device{}
	name, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}

	ifas, err := net.Interfaces()
	tmpMac := ""
	if err != nil {
		log.Panic(err)
	}
	for _, ifa := range ifas {
		tmpMac = ifa.HardwareAddr.String()
		if tmpMac != "" {
			break
		}
	}

	tmpDevice.Mac = tmpMac // just take the MAC address of the first interface in the list
	tmpDevice.Name = name
	return tmpDevice
}

func processRegisterResponse(resp *http.Response) error {

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("processRegisterResponse failed to process response")
		return err
	}
	log.Println("Response body: " + string(body))

	var respMap map[string]interface{}
	err = json.Unmarshal([]byte(body), &respMap)

	// check the code received in the response
	code := int(respMap["code"].(float64))
	if code == 100 {
		myInfo.Key = respMap["key"].(string)
	} else if code == 101 {
	} else {
		log.Printf("processRegisterResponse does not know about this response code, %d\n", respMap["code"])
		return fmt.Errorf("processRegisterResponse does not know about this response code, %d\n", respMap["code"])
	}

	return nil // everything went well
}

func processCheckinResponse(resp *http.Response) error {

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("processCheckinResponse failed to process response")
		return err
	}
	log.Println("Response body: " + string(body))

	var respMap map[string]interface{}
	err = json.Unmarshal([]byte(body), &respMap)

	// check the code received in the response
	code := int(respMap["code"].(float64))
	if code == 200 {
	} else {
		log.Printf("processCheckinResponse does not know about this response code, %d\n", respMap["code"])
		return fmt.Errorf("processCheckinResponse does not know about this response code, %d\n", respMap["code"])
	}

	return nil
}
