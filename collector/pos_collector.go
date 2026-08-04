package collector

import (
	"database/sql"
	"log"
	"maps"
	"os"
	"strings"
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

func (p *PosCollector) GetDevicePosition(key string) (MagnetAgvDevicePositionInfo, bool) {
	p.devicePositionMapMutex.RLock()
	defer p.devicePositionMapMutex.RUnlock()
	val, ok := p.devicePositionMap[key]
	return val, ok
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
	if positionDb != nil {
		p.positionDb = positionDb.C
	}
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
	s.Every("60s").Do(p.inquireDevicePositionList)
	s.StartAsync()
}

// 얘는 60초마다 주기적으로 업데이트 해야하며, mutex 처리가 되어야 함
// 맵핑 테이블을 읽어오는 것임.
func (p *PosCollector) inquireDeviceMapList() error {
	query := "SELECT id, agv_id, wifi_ip, p5g_ip, p5g_imsi, location, name, description, region_code, factory_code FROM p5g_magnet_agv_info WHERE region_code = ? AND factory_code = ?"
	rows, err := p.mappingDb.Query(query, p.regionCode, p.factoryCode)
	if err != nil {
		log.Println("Error fetching device map:", err)
		return err
	}
	defer rows.Close()

	newDeviceMap := make(map[string]MagnetAgvDeviceMapInfo)
	for rows.Next() {
		var device MagnetAgvDeviceMapInfo
		var agvId, wifiIp, p5gIp, p5gImsi, location, name, description, regionCode, factoryCode sql.NullString
		err := rows.Scan(&device.Id, &agvId, &wifiIp, &p5gIp, &p5gImsi, &location, &name, &description, &regionCode, &factoryCode)
		if err != nil {
			log.Println("Error scanning device map:", err)
			return err
		}
		device.AgvId = agvId.String
		device.WifiIp = wifiIp.String
		device.P5gIp = p5gIp.String
		device.P5gImsi = p5gImsi.String
		device.Location = location.String
		device.Name = name.String
		device.Description = description.String
		device.RegionCode = regionCode.String
		device.FactoryCode = factoryCode.String
		if device.AgvId == "" {
			continue
		}
		newDeviceMap[device.AgvId] = device
	}
	p.deviceMapMutex.Lock()
	p.deviceMap = newDeviceMap
	p.deviceMapMutex.Unlock()
	return nil
}

// MSSQL의 VIF_RobotState view에서 전체 장치의 위치 정보를 조회하는 함수
func (p *PosCollector) inquireDevicePositionList() error {
	err := p.inquireDeviceMapList()
	if err != nil {
		log.Println("Error fetching device map:", err)
		return err
	}

	if p.positionDb == nil {
		return nil
	}

	query := "SELECT RobotId, Robot, LogDT, ACSMode, Mode, Connected, X, Y, H FROM VIF_RobotState"
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
		var robotId, logDT, acsMode, mode, x, y, h sql.NullInt64
		var robot sql.NullString
		var connected sql.NullBool
		err := rows.Scan(&robotId, &robot, &logDT, &acsMode, &mode, &connected, &x, &y, &h)
		if err != nil {
			log.Println("Error scanning device position:", err)
			return err
		}
		device.RobotId = int(robotId.Int64)
		device.Robot = strings.TrimSpace(robot.String)
		device.LogDT = logDT.Int64
		device.ACSMode = int(acsMode.Int64)
		device.Mode = int(mode.Int64)
		device.Connected = connected.Bool
		device.X = int(x.Int64)
		device.Y = int(y.Int64)
		device.H = int(h.Int64)
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
