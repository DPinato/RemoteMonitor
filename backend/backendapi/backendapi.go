// runs the backend services required for a device to register, check-in and communicate with the central service

package backendapi

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// variables to connect to postgresql database
const (
	pgresHost      = "10.21.0.2"
	pgresPort      = 5432
	pgresUser      = "postgres"
	pgresPass      = "qUA2d22b"
	pgresDBName    = "rm_test"
	pgresTableName = "reg_devices"
)

// ReturnCode is for responses to devices
type ReturnCode struct {
	Code       int    `json:"code"`
	CodeString string `json:"code_string"`
	Comment    string `json:"comment"`
}

var deviceList []Device                  // cache for list of current devices
var returnCodeList map[string]ReturnCode // store codes to return to clients
var dbObj *sql.DB                        // database object
var pCreds map[string]interface{}        // store postgres-related info
var codeListLocation = "../return_codes.json"
var postgresCredFile = "postgres.json"

func SetupBackend() {
	var err error

	// import return codes from JSON file
	err = importReturnCodes(codeListLocation)
	if err != nil {
		log.Panicln(err)
	}
	log.Printf("Loaded %d return codes\n", len(returnCodeList))

	// get postgres credentials from file
	pCreds, err = readPostgresCredentialsFromFile(postgresCredFile)
	if err != nil {
		log.Printf("Failed to read Postgres credentials from %s\n", postgresCredFile)
		log.Panic(err)
	}

	// connect to postgres
	dbObj, err = connectToPostgres(pCreds["host"].(string),
		pCreds["user"].(string),
		pCreds["password"].(string),
		pCreds["database"].(string),
		int(pCreds["port"].(float64)))
	if err != nil {
		log.Panic(err)
	}
	defer dbObj.Close()
	log.Printf("Successfully connected to Postgres at %s:%d\n", pgresHost, pgresPort)
	log.Printf("Using database %s, table %s\n", pgresDBName, pgresTableName)

	// start HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/register", registerDevice).Methods("POST")
	router.HandleFunc("/checkin", checkInDevice).Methods("POST")

	err = http.ListenAndServe(":8000", router)
	if err != nil {
		log.Println("HTTP server terminated, PANIC")
		log.Panic(err)
	}
}

func importReturnCodes(fileLoc string) error {
	// read return codes from fileLoc JSON file, they are used as follows:
	// 1xxx codes are returned during device registration
	// 2xxx codes are returned during device check in / out
	// 3xxx codes are returned when data is being received from the device
	returnCodeList = make(map[string]ReturnCode)

	byteData, err := ioutil.ReadFile(fileLoc)
	if err != nil {
		return err
	}

	// convert bytes read to JSON map
	var tmpCodeMap []map[string]interface{}
	err = json.Unmarshal(byteData, &tmpCodeMap)
	if err != nil {
		return err
	}

	// store it in returnCodeList, the key will be the code
	for _, element := range tmpCodeMap {
		var tmpCode ReturnCode
		tmpCode.Code = int(element["code"].(float64))
		tmpCode.CodeString = element["code_string"].(string)
		tmpCode.Comment = element["comment"].(string)
		returnCodeList[tmpCode.CodeString] = tmpCode
	}

	return nil
}

////////////
// respond to HTTP calls from client
func registerDevice(w http.ResponseWriter, r *http.Request) {
	// allows a device to register with this API
	// the device provides a name, a key is provided to the device which must be used for any other call
	log.Println("New device registration attempt from " + r.Host)
	w.Header().Set("Content-Type", "application/json")

	// check whether the request body has proper JSON and has all the information required
	tmpDev, code := readRegisterRequestBody(r.Body)

	if code != returnCodeList["RequestOK"].Code {
		var response string
		if code == returnCodeList["MissingInformation"].Code {
			response, _ = generateErrorResponse("MissingInformation")
		} else if code == returnCodeList["BadJSON"].Code {
			response, _ = generateErrorResponse("BadJSON")
		}

		log.Printf("Received bad or incomplete request (error %d), %s\n", code, response)
		http.Error(w, response, http.StatusBadRequest)
		return
	}

	// check if device is already in the list
	index := FindDeviceByMac(deviceList, tmpDev)
	if index != -1 {
		log.Println(deviceList[index].Name + " (" + deviceList[index].Mac + ")" + " attempted to register again")
		response, _ := generateErrorResponse("AlreadyRegistered")
		http.Error(w, response, http.StatusBadRequest)
	} else {
		// generate a key for this device
		// TODO: this is not very secure yet, it just takes the MD5 hash of the MAC + Name
		tmpDev.Key = BytesToString(sha256.Sum256([]byte(tmpDev.Mac + tmpDev.Name)))
		log.Printf("Generated key for %s: %s\n", tmpDev.Name, tmpDev.Key)

		// add it to the list and send a response back with the key
		log.Printf("Registered new device, %s (%s)\n", tmpDev.Name, tmpDev.Mac)
		deviceList = append(deviceList, tmpDev)

		// build response to send
		resp, err := generateRegisterResponse(tmpDev)
		if err != nil {
			log.Println(err)
		}

		// err := json.NewEncoder(w).Encode(responseMap)
		w.WriteHeader(200)
		_, err = w.Write([]byte(resp))
		if err != nil {
			log.Println(err)
		}

		// update postgres
		err = newDeviceRegister(tmpDev, pCreds["reg_table"].(string), dbObj)
		if err != nil {
			log.Println(err)
		}

	}

	Debug_dumpDeviceList(deviceList) // just for DEBUG
}

func checkInDevice(w http.ResponseWriter, r *http.Request) {
	// check-in endpoint for device, replies with 204 code if successful check-in
	// device has to be already registered and provide its key, for the check-in to be considered valid
	var responseMap = make(map[string]interface{}) // map used to reply to client
	log.Println("New check-in attempt from " + r.Host)
	var tmpDev *Device // reference to device checking in
	tmpDev, code := readCheckinRequestBody(r.Body)

	if code != returnCodeList["CheckinOK"].Code {
		var response string // response string to send to the device in case of an error
		if code == returnCodeList["BadKey"].Code {
			response, _ = generateErrorResponse("BadKey")
		} else if code == returnCodeList["MalformedCheckin"].Code {
			response, _ = generateErrorResponse("MalformedCheckin")
		}

		log.Printf("Received bad checkin (error %d), %s\n", code, response)
		http.Error(w, response, http.StatusBadRequest)
		return
	}

	// update last check-in status of the device
	// reply back with the last time the device checked in as a confirmation, i.e. now
	log.Printf("Received valid checkin from %s\n", tmpDev.Name)
	tmpDev.LastCheckin = time.Now()

	// build response to send
	responseMap["code"] = returnCodeList["CheckinOK"].Code
	responseMap["last_checkin"] = tmpDev.LastCheckin.String()
	json.NewEncoder(w).Encode(responseMap)

	Debug_dumpDeviceList(deviceList) // just for DEBUG

}

/////////////
// helpful functions for API calls
func readRegisterRequestBody(body io.ReadCloser) (Device, int) {
	// check if the HTTP request body received from registerDevice has all the necessary parameters
	// return an error if either JSON is bad or if MAC / name of device is missing
	var tmpDev Device
	var err error
	err = json.NewDecoder(body).Decode(&tmpDev)
	if err == nil {
		// request not malformed, check if all the necessary parameters are there
		if tmpDev.Name == "" {
			return tmpDev, returnCodeList["MissingInformation"].Code
		}

		if tmpDev.Mac == "" {
			err = fmt.Errorf("Missing device MAC")
			return tmpDev, returnCodeList["MissingInformation"].Code
		}
	} else {
		// request malformed
		return tmpDev, returnCodeList["BadJSON"].Code
	}

	return tmpDev, returnCodeList["RequestOK"].Code
}

func readCheckinRequestBody(body io.ReadCloser) (*Device, int) {
	// check if a check-in request body is valid
	// if valid, return a reference to the device performing the check-in
	var tmpDev *Device
	var err error
	err = json.NewDecoder(body).Decode(&tmpDev)
	if err == nil {
		// check if a known key is found
		index := FindDeviceByKey(deviceList, *tmpDev)
		if index == -1 {
			// not found
			return nil, returnCodeList["BadKey"].Code
		} else {
			tmpDev = &deviceList[index]
		}

	} else {
		// request malformed
		return nil, returnCodeList["MalformedCheckin"].Code
	}

	return tmpDev, returnCodeList["CheckinOK"].Code

}

func generateRegisterResponse(dev Device) (string, error) {
	// generate a proper response message to return to a device after receiving a successful register request
	var responseMap = make(map[string]interface{}) // map used to reply to client
	responseMap["code"] = returnCodeList["RegisterOK"].Code
	responseMap["comment"] = returnCodeList["RegisterOK"].Comment
	responseMap["code_string"] = returnCodeList["RegisterOK"].CodeString
	responseMap["key"] = dev.Key
	responseMap["mac"] = dev.Mac

	jsonData, err := json.Marshal(responseMap)
	if err != nil {
		log.Println("generateRegisterResponse failed to marshal JSON")
		return "", err
	}

	return string(jsonData), nil

}

func generateErrorResponse(codeStr string) (string, error) {
	// generate a proper error response message in JSON format
	jsonData, err := json.Marshal(returnCodeList[codeStr])
	if err != nil {
		log.Println("generateErrorResponse failed to marshal JSON")
		log.Println(returnCodeList[codeStr].Comment)
		log.Printf("code: %d\n", returnCodeList[codeStr].Code)
		return "", err
	}

	return string(jsonData), nil
}

/////////////
/////////////
// generic helper functions
func BytesToString(data [32]byte) string {
	return fmt.Sprintf("%x", data)
}
