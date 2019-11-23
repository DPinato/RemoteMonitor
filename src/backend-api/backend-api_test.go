// runs the backend services required for a device to register, check-in and communicate with the central service

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"
)

func Test_ReadRegisterRequestBody(t *testing.T) {
	assertCorrect := func(t *testing.T, got, want uint32) {
		if got != want {
			t.Errorf("Got %v, want %v", got, want)
		}
	}

	t.Run("Properly formatted request body", func(t *testing.T) {
		var testDev Device
		testDev.Name = "Sample name"
		testDev.Mac = "00:01:02:03:04:05"
		byteData, _ := json.Marshal(testDev)
		r := ioutil.NopCloser(bytes.NewReader(byteData))
		_, got := readRegisterRequestBody(r)
		want := RequestOK
		assertCorrect(t, got, want)
	})

	t.Run("Missing device name", func(t *testing.T) {
		var testDev Device
		testDev.Name = ""
		testDev.Mac = "00:01:02:03:04:05"
		byteData, _ := json.Marshal(testDev)
		r := ioutil.NopCloser(bytes.NewReader(byteData))
		_, got := readRegisterRequestBody(r)
		want := MissingInformation
		assertCorrect(t, got, want)
	})

	t.Run("Missing device MAC", func(t *testing.T) {
		var testDev Device
		testDev.Name = "Sample name"
		testDev.Mac = ""
		byteData, _ := json.Marshal(testDev)
		r := ioutil.NopCloser(bytes.NewReader(byteData))
		_, got := readRegisterRequestBody(r)
		want := MissingInformation
		assertCorrect(t, got, want)
	})

	t.Run("Malformed JSON request body", func(t *testing.T) {
		testJson := "{\"name\":\"Sample name\", \"mac\":\"00:00:00:00:00:00}"
		r := ioutil.NopCloser(bytes.NewReader([]byte(testJson)))
		_, got := readRegisterRequestBody(r)
		want := BadJSON
		assertCorrect(t, got, want)
	})

	t.Run("Device provides bad MAC address", func(t *testing.T) {
		var testDev Device
		testDev.Name = "Sample name"
		testDev.Mac = "00:011::03:04:5"
		byteData, _ := json.Marshal(testDev)
		r := ioutil.NopCloser(bytes.NewReader(byteData))
		_, got := readRegisterRequestBody(r)
		want := BadDeviceMac
		assertCorrect(t, got, want)
	})

}

func Test_BytesToString(t *testing.T) {
	assertCorrect := func(t *testing.T, got, want string) {
		if got != want {
			t.Errorf("Got %v, want %v", got, want)
		}
	}

	t.Run("Proper [32]byte as an input", func(t *testing.T) {
		var sample = [32]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
		got := BytesToString(sample)
		want := "0001020304050607000102030405060700010203040506070001020304050607"
		assertCorrect(t, got, want)
	})

}
