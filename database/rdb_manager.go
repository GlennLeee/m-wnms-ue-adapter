package database

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	_ "github.com/go-sql-driver/mysql"
)

type RdbConf struct {
	username string
	password string
	url      string
	schema   string
	C        *sql.DB
}

var retryCnt = 0

func (d *RdbConf) connection() {

	defer func() {
		if err := recover(); err != nil {
			time.Sleep(1000 * time.Millisecond)
			retryCnt++
			log.Println("[Connection Error] ", retryCnt, " Retry")
			d.connection()
		}
	}()

	dataSource := d.username + ":" + d.password + "@tcp(" + d.url + ")/" + d.schema
	conn, err := sql.Open("mysql", dataSource)

	if err != nil {
		log.Fatal(err)
	}

	// connection check
	if err = conn.Ping(); err != nil {
		conn.Close()
		panic("RDB Connection Error")
	}

	log.Println("DB Opened [", d.url, "]")
	d.C = conn
}

func (d *RdbConf) Initialize(url string, username string, password string, schema string) {
	d.username = username
	d.password = password
	d.schema = schema
	d.url = url

	d.connection()
	d.getSettingInformations()
}

func NumericParamChecker(paramName string, defaultValue int, minValue int, maxValue int) int {
	var retVal int
	if val, ok := os.LookupEnv(paramName); ok {
		tempVal, err := strconv.Atoi(val)
		if err != nil {
			retVal = defaultValue
			log.Printf("%s error, use default value (%v)\n", paramName, defaultValue)
		} else {
			if tempVal < minValue || tempVal > maxValue {
				retVal = defaultValue
				log.Printf("%s error, use default value (%v)\n", paramName, defaultValue)
			} else {
				retVal = tempVal
			}
		}
	} else {
		retVal = defaultValue
		log.Printf("%s is not set, use default value (%v)\n", paramName, defaultValue)
	}
	return retVal
}

// 특정 시간마다 설정값 업데이트
func (d *RdbConf) getSettingInformations() {
	log.Println("GET SETTING INFORMATION")

	d.updateSettingInformations()

	// 설정 정보 읽어오기
	s := gocron.NewScheduler(time.UTC)
	s.Every("30s").Do(d.updateSettingInformations)
	s.StartAsync()

}
func (d *RdbConf) updateSettingInformations() {
	query := "SELECT `key`, `value` FROM wnms_config"

	rows, err := d.C.Query(query)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		// if !strings.HasPrefix(k, "p5g") {
		// 	continue
		// }
		// log.Println(k, v)
		os.Setenv(k, v)
	}
}
