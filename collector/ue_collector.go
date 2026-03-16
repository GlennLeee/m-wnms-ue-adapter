package collector

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/database"
	"connectlabs.co.kr/wnms/wnms-ue-adapter/model"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

type UeManager struct {
	loc        *time.Location
	mqttClient MQTT.Client
	C          *sql.DB
}

type PointData struct {
	Cx     int
	Cy     int
	Angle  int
	Status int
	WfDvc  string
	P5gDvc string
}

// 리스너 초기화
func (u *UeManager) UeDataListener(db *database.RdbConf) {
	log.Println("RUN P-5G UE Data Listener")
	log.Println(os.Getenv("TZ"))
	loc, errTz := time.LoadLocation(os.Getenv("TZ"))
	if errTz != nil {
		panic(errTz)
	}
	u.loc = loc
	u.C = db.C

	// crash prevention
	defer func() {
		if err := recover(); err != nil {
			log.Println("Initialization Error. Restart after 5 seconds")
			time.Sleep(5 * time.Second)
			go u.UeDataListener(db)
		}
	}()

	// 3초 delay 하지말고 추후, 변수에 임시 저장 후 넘어오면 임시로 보유한 것만 추가로 저장하는 것으로 하자.
	opts := MQTT.NewClientOptions().AddBroker("tcp://" + os.Getenv("MQTT_HOST"))
	serverNumber := os.Getenv("SERVER_NUMBER")

	opts.OnConnect = func(c MQTT.Client) {
		// 에러나면 재시작을 자동으로 하게 해야함.
		if token := c.Subscribe(os.Getenv("p5g_ue_topic"), 1, u.mqttHandler); token.Wait() && token.Error() != nil {
			log.Fatalln(token.Error())
		} else {
			log.Println("START SUBSCRIBING UE STREAMING TOPIC")
		}
	}

	// opts.SetConnectRetryInterval(time.Second)
	// opts.SetConnectRetry(true)
	opts.SetOrderMatters(false)

	// opts = opts.SetAutoReconnect(true)
	// opts.SetPingTimeout(5 * time.Second)
	// opts = opts.SetKeepAlive(10 * time.Second)
	uuidStr := uuid.NewString()
	uuidArr := strings.Split(uuidStr, "-")

	opts = opts.SetClientID("p5g_ue_da_" + serverNumber + "_" + uuidArr[0])
	opts = opts.SetConnectionLostHandler(func(mqttClient MQTT.Client, err error) {
		log.Printf("!!!!!! mqtt connection lost error: %s\n", err.Error())
	})

	u.mqttClient = MQTT.NewClient(opts)

	if token := u.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		panic(token.Error())
	}

	log.Println("Start MQTT UE Data Listener")
}

// 등록할 핸들러임
func (u *UeManager) mqttHandler(client MQTT.Client, msg MQTT.Message) {
	// Downstream gRPC queues are bounded; handle messages in-place so queue pressure
	// throttles MQTT processing instead of spawning unbounded goroutines.
	um := model.UeMessage{}
	if err := json.Unmarshal(msg.Payload(), &um); err != nil {
		log.Printf("invalid UE payload: %v\n", err)
		return
	}

	um.Data.Lan1.BpsRxAvg, um.Data.Lan1.BpsRxSum = splitBps(um.Data.Lan1.BpsRx)
	um.Data.Lan1.BpsTxAvg, um.Data.Lan1.BpsTxSum = splitBps(um.Data.Lan1.BpsTx)
	um.Data.Lan2.BpsRxAvg, um.Data.Lan2.BpsRxSum = splitBps(um.Data.Lan2.BpsRx)
	um.Data.Lan2.BpsTxAvg, um.Data.Lan2.BpsTxSum = splitBps(um.Data.Lan2.BpsTx)

	um.Data.Wlan0.BpsRxAvg, um.Data.Wlan0.BpsRxSum = splitBps(um.Data.Wlan0.BpsRx)
	um.Data.Wlan0.BpsTxAvg, um.Data.Wlan0.BpsTxSum = splitBps(um.Data.Wlan0.BpsTx)

	um.Data.Qmimux0.BpsRxAvg, um.Data.Qmimux0.BpsRxSum = splitBps(um.Data.Qmimux0.BpsRx)
	um.Data.Qmimux0.BpsTxAvg, um.Data.Qmimux0.BpsTxSum = splitBps(um.Data.Qmimux0.BpsTx)

	um.Data.System.RuntimeSecs, _ = model.ParseTimeToSeconds(um.Data.System.Runtime)

	cpuRate, err := strconv.ParseFloat(um.Data.System.CpuStr, 32)
	if err != nil {
		log.Println(err)
	}
	um.Data.System.Cpu = float32(cpuRate)

	uptimeTm, err := time.ParseInLocation("2006-01-02 15:04:05", um.Data.System.StartTime, u.loc)
	if err != nil {
		log.Println(err)
	}
	um.Data.System.StartTimeEpoch = uptimeTm.Unix()

	tm, err := time.ParseInLocation("[2006-01-02, 15:04:05.999]", um.Data.Date, u.loc)
	if err != nil {
		log.Println(err)
	}
	um.Data.Timestamp = tm.UnixMilli()

	imei := um.Data.Qmimux0.Imei
	mac := um.Data.Wlan0.Mac

	// IMEI가 정상적일때만 처리한다.
	if imei == "" || imei == "0" || len(imei) == 15 {
		sendRpcData(&um)
		if imei == "" || imei == "0" {
			imei = u.findImeiValue(mac)
		}
		u.sendDevicePosition(imei, mac, um.Data.Pos)
	}
}

func (u *UeManager) findImeiValue(wifiMac string) string {
	query := "SELECT imei FROM awnms.p5g_ue_info WHERE wifi_mac = ?"
	qRes, err := u.C.Query(query, wifiMac)
	if err != nil {
		log.Println("Find P5G IMEI error", err)
	}

	defer qRes.Close() //반드시 닫는다 (지연하여 닫기)
	var imei string
	for qRes.Next() {
		qRes.Scan(&imei)
		if imei != "" && imei != "0" {
			break
		}
	}

	return imei
}

func (u *UeManager) sendDevicePosition(p5gImei string, wifiMac string, pos model.Position) {
	if pos.Angle == "" || pos.PosX == "" || pos.PosY == "" {
		log.Println("POS IS NULL")
		return
	}

	pX, err := strconv.ParseFloat(pos.PosX, 64)
	if err != nil {
		log.Println(err)
	}

	pY, err := strconv.ParseFloat(pos.PosY, 64)
	if err != nil {
		log.Println(err)
	}

	angle, err := strconv.ParseFloat(pos.Angle, 64)
	if err != nil {
		log.Println(err)
	}

	status, err := strconv.ParseInt(pos.Status, 10, 64)
	if err != nil {
		log.Println(err)
	}

	p := PointData{
		Cx:     int(pX),
		Cy:     int(pY) * (-1),
		Angle:  int(angle),
		Status: int(status),
		WfDvc:  strings.ToUpper(wifiMac),
		P5gDvc: p5gImei,
	}

	jsonData, err := json.Marshal(p)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return
	}

	u.mqttClient.Publish("client-point", 0, false, jsonData)
}

func sendRpcData(data *model.UeMessage) {
	system := &model.P5GUeSystem{
		CpuRate:         data.Data.System.Cpu,
		MemoryRate:      data.Data.System.Memory,
		StorageRate:     data.Data.System.Storage,
		Temperature:     data.Data.System.Temp,
		FirmwareVersion: data.Data.System.Fwver,
		Handle:          "",
		UptimeSecs:      data.Data.System.RuntimeSecs,
		StartEpoch:      data.Data.System.StartTimeEpoch,
		Commway:         data.Data.System.Commway,
		Lan1Act:         int32(data.Data.System.Lan1Act),
		Lan2Act:         int32(data.Data.System.Lan2Act),
		Wlan0Act:        int32(data.Data.System.Wlan0Act),
		Qmimux0Act:      int32(data.Data.System.Qmimux0Act),
	}

	lan1 := &model.P5GUeLan{
		BpsTxAvg: data.Data.Lan1.BpsTxAvg,
		BpsTxSum: data.Data.Lan1.BpsTxSum,
		BpsRxAvg: data.Data.Lan1.BpsRxAvg,
		BpsRxSum: data.Data.Lan1.BpsRxSum,
	}

	lan2 := &model.P5GUeLan{
		BpsTxAvg: data.Data.Lan2.BpsTxAvg,
		BpsTxSum: data.Data.Lan2.BpsTxSum,
		BpsRxAvg: data.Data.Lan2.BpsRxAvg,
		BpsRxSum: data.Data.Lan2.BpsRxSum,
	}

	wlan := &model.P5GUeWlan{
		IpAddress:  data.Data.Wlan0.Ip,
		MacAddress: data.Data.Wlan0.Mac,
		BpsTxAvg:   data.Data.Wlan0.BpsTxAvg,
		BpsTxSum:   data.Data.Wlan0.BpsTxSum,
		BpsRxAvg:   data.Data.Wlan0.BpsRxAvg,
		BpsRxSum:   data.Data.Wlan0.BpsRxSum,
		Rssi:       int32(data.Data.Wlan0.Rssi),
		NoiseLevel: int32(data.Data.Wlan0.NoiseLevel),
		AntA:       int32(data.Data.Wlan0.AntA),
		AntB:       int32(data.Data.Wlan0.AntB),
		McsTx:      data.Data.Wlan0.McsTx,
		McsRx:      data.Data.Wlan0.McsRx,
	}

	// ant0Ecio, _ := strconv.ParseFloat(data.Data.Qmimux0.Ant0Ecio, 32)
	// ant1Ecio, _ := strconv.ParseFloat(data.Data.Qmimux0.Ant1Ecio, 32)
	// ant2Ecio, _ := strconv.ParseFloat(data.Data.Qmimux0.Ant2Ecio, 32)
	// ant3Ecio, _ := strconv.ParseFloat(data.Data.Qmimux0.Ant3Ecio, 32)

	qmimux := &model.P5GUeQmimux{
		IpAddress: data.Data.Qmimux0.Ip,
		Imei:      data.Data.Qmimux0.Imei,
		Imsi:      data.Data.Qmimux0.Imsi,
		Pcid:      fmt.Sprintf("%d", data.Data.Qmimux0.Pcid),
		Tac:       int32(data.Data.Qmimux0.Tac),
		Rsrp:      int32(data.Data.Qmimux0.Rsrp),
		Rsrq:      int32(data.Data.Qmimux0.Rsrp),
		Rssi:      int32(data.Data.Qmimux0.Rssi),
		Cqi:       int32(data.Data.Qmimux0.Cqi),
		Ri:        int32(data.Data.Qmimux0.Ri),
		Rb:        int32(data.Data.Qmimux0.Rb),
		McsTx:     data.Data.Qmimux0.McsTx,
		McsRx:     data.Data.Qmimux0.McsRx,
		BpsTxAvg:  data.Data.Qmimux0.BpsTxAvg,
		BpsTxSum:  data.Data.Qmimux0.BpsTxSum,
		BpsRxAvg:  data.Data.Qmimux0.BpsRxAvg,
		BpsRxSum:  data.Data.Qmimux0.BpsRxSum,
		Ant0Power: float32(data.Data.Qmimux0.Ant0Power),
		Ant0Ecio:  float32(data.Data.Qmimux0.Ant0Ecio),
		Ant0Rsrp:  float32(data.Data.Qmimux0.Ant0Rsrp),
		Ant1Power: float32(data.Data.Qmimux0.Ant1Power),
		Ant1Ecio:  float32(data.Data.Qmimux0.Ant1Ecio),
		Ant1Rsrp:  float32(data.Data.Qmimux0.Ant1Rsrp),
		Ant2Power: float32(data.Data.Qmimux0.Ant2Power),
		Ant2Ecio:  float32(data.Data.Qmimux0.Ant2Ecio),
		Ant2Rsrp:  float32(data.Data.Qmimux0.Ant2Rsrp),
		Ant3Power: float32(data.Data.Qmimux0.Ant3Power),
		Ant3Ecio:  float32(data.Data.Qmimux0.Ant3Ecio),
		Ant3Rsrp:  float32(data.Data.Qmimux0.Ant3Rsrp),
	}

	if data.Data.Pos.PosX == "" || data.Data.Pos.PosY == "" || data.Data.Pos.Angle == "" || data.Data.Pos.Status == "" {
		data.Data.Pos.PosX = "0.0"
		data.Data.Pos.PosY = "0.0"
		data.Data.Pos.Angle = "0.0"
		data.Data.Pos.Status = "-9999"
	}

	posX, err := strconv.ParseFloat(data.Data.Pos.PosX, 64)
	if err != nil {
		log.Println("P5G Data Conv", err)
	}
	posY, err := strconv.ParseFloat(data.Data.Pos.PosY, 64)
	if err != nil {
		log.Println("P5G Data Conv", err)
	}
	angle, err := strconv.ParseFloat(data.Data.Pos.Angle, 64)
	if err != nil {
		log.Println("P5G Data Conv", err)
	}
	status, err := strconv.ParseInt(data.Data.Pos.Status, 10, 64)
	if err != nil {
		log.Println("P5G Data Conv", err)
	}

	location := &model.P5GUeLoc{
		PosX:   float32(posX),
		PosY:   float32(posY) * (-1),
		Angle:  float32(angle),
		Status: int32(status),
	}

	p5gUeStats := model.P5GUeStats{
		Timestamp:   data.Data.Timestamp,
		DeviceName:  data.Data.DeviceName,
		ModelName:   data.Data.ModelName,
		GroupName:   data.Data.GroupName,
		System:      system,
		Lan1:        lan1,
		Lan2:        lan2,
		Wlan0:       wlan,
		Qmimux0:     qmimux,
		Location:    location,
		Dtc:         data.Data.Dtc,
		RegionCode:  os.Getenv("DATA_REGION"),
		FactoryCode: os.Getenv("DATA_FACTORY"),
	}

	p5gUeStatsReport := model.P5GUeStatsReport{
		Timestamp: uint64(data.Data.Timestamp),
	}
	p5gUeStatsReport.P5GUeStats = append(p5gUeStatsReport.P5GUeStats, &p5gUeStats)

	rpcRequest := model.RpcRequest{}
	rpcRequest.Timestamp = data.Data.Timestamp
	rpcRequest.P5GUeStatsReport = &p5gUeStatsReport

	model.GrpcServiceChannel <- model.GrpcServiceMsg{
		Method:  "UpdateP5gUeStatsData",
		Message: &rpcRequest,
	}
}

func splitBps(data string) (int64, int64) {
	data = strings.ReplaceAll(data, " ", "")
	if data == "" {
		return 0, 0
	}
	ret := strings.Split(data, ",")
	ret1, err := strconv.ParseInt(ret[0], 10, 64)
	if err != nil {
		tmp, err := strconv.ParseFloat(ret[0], 32)
		if err != nil {
			log.Println(err)
		}

		ret1 = int64(math.Round(tmp))
	}
	ret2, err := strconv.ParseInt(ret[1], 10, 64)
	if err != nil {
		tmp, err := strconv.ParseFloat(ret[1], 32)
		if err != nil {
			log.Println(err)
		}

		ret2 = int64(math.Round(tmp))
	}
	return ret1, ret2
}
