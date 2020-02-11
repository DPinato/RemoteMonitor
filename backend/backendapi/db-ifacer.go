// communicate with database
// devices are registered in postgresql database

package backendapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	_ "github.com/lib/pq"
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
