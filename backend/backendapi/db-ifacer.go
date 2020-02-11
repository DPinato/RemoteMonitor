// communicate with databases
// Postgres is used to maintain information on currently registered devices, including their authentication key

package backendapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	_ "github.com/lib/pq" // used for postgres driver
)

func connectToPostgres(host, user, password, dbname string, port int) (*sql.DB, error) {
	// connect to database, return pointer to db object
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	// check connection to database
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func readPostgresCredentialsFromFile(fileLoc string) (map[string]interface{}, error) {
	// read credentials for postgres from JSON file in fileLoc
	output := make(map[string]interface{})

	jsonFile, err := os.Open(fileLoc)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(byteValue), &output)

	return output, nil

}

func newDeviceRegister(dev Device, table string, dbObj *sql.DB) error {
	// add device to postgres
	now := time.Now()

	sqlStatement := fmt.Sprintf("INSERT INTO %s ", table)
	sqlStatement += `(key, name, os, mac, first_register_ts, last_register_ts, last_checkin_ts, last_checkout_ts)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	values := []interface{}{dev.Key,
		dev.Name,
		dev.OS,
		dev.Mac,
		now,
		nil,
		nil,
		nil}
	_, err := dbObj.Exec(sqlStatement, values...)
	if err != nil {
		return err
	}

	return nil
}
