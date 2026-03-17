package collector

import (
	"database/sql"
	"log"
	"maps"
	"os"
	"sync"
	"time"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/database"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
)

type MagnetAgvDeviceMapInfo struct {
	Id          uuid.UUID `json:"id"`
	AgvId       string    `json:"agv_id"`
	WifiIp      string    `json:"wifi_ip"`
	P5gIp       string    `json:"p5g_ip"`
	P5gImsi     string    `json:"p5g_imsi"`
	Location    string    `json:"location"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Timestamp   int64     `json:"timestamp"`
	RegionCode  string    `json:"region_code"`
	FactoryCode string    `json:"factory_code"`
}
type PosCollector struct {
	positionDb             *sql.DB
	mappingDb              *sql.DB
	deviceMap              map[string]MagnetAgvDeviceMapInfo
	deviceMapMutex         *sync.RWMutex
	regionCode             string
	factoryCode            string
	devicePositionMap      map[string]MagnetAgvDevicePositionInfo
	devicePositionMapMutex *sync.RWMutex
}

type MagnetAgvDevicePositionInfo struct {
	RobotId   int    `json:"robot_id"`
	Robot     string `json:"robot"`
	LogDT     int64  `json:"log_dt"`
	ACSMode   int    `json:"acs_mode"`
	Mode      int    `json:"mode"`
	Connected bool   `json:"connected"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	H         int    `json:"h"`
}

func (p *PosCollector) Initialize(positionDb *database.PositionRdbConf, mappingDb *database.RdbConf) {
	p.positionDb = positionDb.C
	p.mappingDb = mappingDb.C
	p.deviceMap = make(map[string]MagnetAgvDeviceMapInfo)
	p.regionCode = os.Getenv("DATA_REGION")
	p.factoryCode = os.Getenv("DATA_FACTORY")
	p.deviceMapMutex = &sync.RWMutex{}
	p.devicePositionMap = make(map[string]MagnetAgvDevicePositionInfo)
	p.devicePositionMapMutex = &sync.RWMutex{}
}

func (p *PosCollector) Run() {
	s := gocron.NewScheduler(time.UTC)
	s.Every("60s").Do(p.inquireDeviceMapList)
	s.Every("60s").Do(p.inquireDevicePositionList)
	s.StartAsync()
}

// 얘는 60초마다 주기적으로 업데이트 해야하며, mutex 처리가 되어야 함
// 맵핑 테이블을 읽어오는 것임.
func (p *PosCollector) inquireDeviceMapList() error {
	query := "SELECT id, agv_id, wifi_ip, p5g_ip, p5g_imsi, location, name, description, timestamp, region_code, factory_code FROM p5g_magnet_agv_info WHERE region_code = ? AND factory_code = ?"
	rows, err := p.mappingDb.Query(query, p.regionCode, p.factoryCode)
	if err != nil {
		log.Println("Error fetching device map:", err)
		return err
	}
	defer rows.Close()

	newDeviceMap := make(map[string]MagnetAgvDeviceMapInfo)
	for rows.Next() {
		var device MagnetAgvDeviceMapInfo
		err := rows.Scan(&device.Id, &device.AgvId, &device.WifiIp, &device.P5gIp, &device.P5gImsi, &device.Location, &device.Name, &device.Description, &device.Timestamp, &device.RegionCode, &device.FactoryCode)
		if err != nil {
			log.Println("Error scanning device map:", err)
			return err
		}
		newDeviceMap[device.AgvId] = device
	}
	p.deviceMapMutex.Lock()
	p.deviceMap = newDeviceMap
	p.deviceMapMutex.Unlock()
	return nil
}

// MSSQL로 이루어진 positiondb에서 전체 장치의 위치 정보를 조회하는 함수
func (p *PosCollector) inquireDevicePositionList() error {
	query := "SELECT RobotId, Robot, LogDT, ACSMode, Mode, Connected, X, Y, H FROM vw_RobotStatus"
	rows, err := p.positionDb.Query(query)
	if err != nil {
		log.Println("Error fetching device position list:", err)
		return err
	}
	defer rows.Close()

	p.deviceMapMutex.RLock()
	deviceMapSnapshot := make(map[string]MagnetAgvDeviceMapInfo, len(p.deviceMap))
	maps.Copy(deviceMapSnapshot, p.deviceMap)
	p.deviceMapMutex.RUnlock()

	positionUpdates := make(map[string]MagnetAgvDevicePositionInfo)
	for rows.Next() {
		var device MagnetAgvDevicePositionInfo
		err := rows.Scan(&device.RobotId, &device.Robot, &device.LogDT, &device.ACSMode, &device.Mode, &device.Connected, &device.X, &device.Y, &device.H)
		if err != nil {
			log.Println("Error scanning device position:", err)
			return err
		}
		if deviceMapInfo, ok := deviceMapSnapshot[device.Robot]; ok {
			// p5g imsi + wifi ip를 키로 사용하여 위치 정보를 저장하는 것임.Aa  AAA
			positionUpdates[deviceMapInfo.P5gImsi+"_"+deviceMapInfo.WifiIp] = device
		}
	}

	p.devicePositionMapMutex.Lock()
	// 일단 위치를 업데이트한다.
	maps.Copy(p.devicePositionMap, positionUpdates)
	p.devicePositionMapMutex.Unlock()
	return nil
}
