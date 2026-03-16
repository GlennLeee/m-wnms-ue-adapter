package main

import (
	"log"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/go-co-op/gocron"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/collector"
	"connectlabs.co.kr/wnms/wnms-ue-adapter/database"
	"connectlabs.co.kr/wnms/wnms-ue-adapter/message"
	"connectlabs.co.kr/wnms/wnms-ue-adapter/model"
)

func registerConfig(key string, valIntf interface{}) {
	valMap := valIntf.(map[string]interface{})

	for newKey, val := range valMap {
		var newKeyName string

		if "" == key {
			newKeyName = newKey
		} else {
			newKeyName = key + "." + newKey
		}

		if reflect.Map == reflect.TypeOf(val).Kind() {
			registerConfig(newKeyName, val)
		} else {
			os.Setenv(newKeyName, val.(string))
		}
	}
}

func main() {
	runtime.GOMAXPROCS(2)
	log.SetOutput(os.Stdout)

	model.GrpcServiceChannel = make(chan model.GrpcServiceMsg, 1000)
	model.GrpcServiceBackupChannel = make(chan model.GrpcServiceMsg, 1000)

	// timezone check
	_, tzFlag := os.LookupEnv("TZ")
	if !tzFlag {
		log.Fatalln("서버의 타임존을 설정해주세요")
	}

	db := &database.RdbConf{}
	db.Initialize(os.Getenv("MARIADB_HOST"), os.Getenv("MARIADB_USERNAME"), os.Getenv("MARIADB_PASSWORD"), os.Getenv("MARIADB_SCHEMA"))
	pdb := &database.PositionRdbConf{}
	pdb.Initialize(os.Getenv("POSITION_DB_HOST"), os.Getenv("POSITION_DB_USERNAME"), os.Getenv("POSITION_DB_PASSWORD"), os.Getenv("POSITION_DB_SCHEMA"), os.Getenv("POSITION_DB_PORT"))

	// init
	log.Println("GRPC INIT")
	grpcServer := message.InitializeGrpcServer() // GRPC Connection (데이터 컬렉터 접속)

	ueManager := collector.UeManager{}
	go ueManager.UeDataListener(db)

	// go pmManager.Run()
	log.Println("SCHEDULER INIT")
	s := gocron.NewScheduler(time.UTC)
	s.Every("60s").Do(grpcServer.SendPing)

	s.StartAsync()
	select {}
}
