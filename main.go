package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
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
	runtime.GOMAXPROCS(4)
	log.SetOutput(os.Stdout)

	model.GrpcServiceChannel = make(chan model.GrpcServiceMsg, 1000)
	model.GrpcServiceBackupChannel = make(chan model.GrpcServiceMsg, 1000)

	propertyFile := flag.String("c", "properties.ini", "Property File Path")
	flag.Parse()

	// 초기 설정 파일 등록
	configMap := make(map[string]interface{})
	*propertyFile = strings.ReplaceAll(*propertyFile, "~", "")
	*propertyFile = strings.ReplaceAll(*propertyFile, "..", "")

	file, err := ioutil.ReadFile(*propertyFile)

	if err != nil {
		log.Fatal("설정 파일이 없습니다.\n절대 경로에 설정 파일을 배치해주세요.")
	}

	err = json.Unmarshal([]byte(string(file)), &configMap)
	registerConfig("", interface{}(configMap))

	db := &database.RdbConf{}
	db.Initialize(os.Getenv("databse.info.host"), os.Getenv("databse.info.username"), os.Getenv("databse.info.password"), os.Getenv("databse.info.database"))

	// init
	collector.InitCollector(db)
	log.Println("GRPC INIT")
	grpcServer := message.InitializeGrpcServer() // GRPC Connection (데이터 컬렉터 접속)

	log.Println("SNMP TRAP INIT")
	fmManager := collector.FmManager{}
	go fmManager.FmListenTrapSet()

	ueManager := collector.UeManager{}
	go ueManager.UeDataListener(db)

	pmManager := collector.PmManager{}

	// go pmManager.Run()
	log.Println("SCHEDULER INIT")
	s := gocron.NewScheduler(time.UTC)
	s.Every("3m").Do(pmManager.Run, db)
	s.Every("10s").Do(grpcServer.SendPing)

	s.StartAsync()
	// router := mux.NewRouter()

	// router.HandleFunc("/snmp/test", fmManager.Replay).Methods("GET")
	// http.ListenAndServe(":8088", router)

	select {}
}
