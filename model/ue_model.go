package model

import (
	"fmt"
	"strconv"
	"strings"
)

type UeMessage struct {
	Data UeData `json:"data"`
}

type UeData struct {
	Date       string `json:"date"`
	Timestamp  int64
	DeviceName string   `json:"deviceName,omitempty"`
	ModelName  string   `json:"modelName"`
	GroupName  string   `json:"groupName"`
	System     UeSystem `json:"system"`
	Lan1       Lan      `json:"lan1"`
	Lan2       Lan      `json:"lan2"`
	Wlan0      Wlan     `json:"wlan0"`
	Qmimux0    Qmimux   `json:"qmimux0"`
	Pos        Position `json:"position,omitempty"`
	Dtc        []string `json:"DTC,omitempty"`
}

type UeSystem struct {
	CpuStr         string `json:"cpu,omitempty"`
	Cpu            float32
	Memory         float32 `json:"memory,omitempty"`
	Storage        float32 `json:"storage,omitempty"`
	Temp           float32 `json:"temp,omitempty"`
	Fwver          string  `json:"fwver,omitempty"`
	StartTime      string  `json:"uptime,omitempty"`
	StartTimeEpoch int64
	Runtime        string `json:"runtime,omitempty"`
	RuntimeSecs    int64
	Commway        string `json:"commway,omitempty"`
	Lan1Act        int    `json:"lan1_act,omitempty"`
	Lan2Act        int    `json:"lan2_act,omitempty"`
	Wlan0Act       int    `json:"wlan0_act,omitempty"`
	Qmimux0Act     int    `json:"qmimux0_act,omitempty"`
}

type Lan struct {
	BpsTx    string `json:"bps_tx,omitempty"`
	BpsTxAvg int64  `json:"bps_tx_avg,omitempty"`
	BpsTxSum int64  `json:"bps_tx_sum,omitempty"`
	BpsRx    string `json:"bps_rx,omitempty"`
	BpsRxAvg int64  `json:"bps_rx_avg,omitempty"`
	BpsRxSum int64  `json:"bps_rx_sum,omitempty"`
}

type Wlan struct {
	Ip         string `json:"ip,omitempty"`
	Mac        string `json:"mac,omitempty"`
	BpsTx      string `json:"bps_tx,omitempty"`
	BpsTxAvg   int64  `json:"bps_tx_avg,omitempty"`
	BpsTxSum   int64  `json:"bps_tx_sum,omitempty"`
	BpsRx      string `json:"bps_rx,omitempty"`
	BpsRxAvg   int64  `json:"bps_rx_avg,omitempty"`
	BpsRxSum   int64  `json:"bps_rx_sum,omitempty"`
	Rssi       int    `json:"rssi,omitempty"`
	NoiseLevel int    `json:"noise_level,omitempty"`
	AntA       int    `json:"antA,omitempty"`
	AntB       int    `json:"antB,omitempty"`
	McsTx      int64  `json:"mcstx,omitempty"`
	McsRx      int64  `json:"mcsrx,omitempty"`
}

type Qmimux struct {
	Ip       string `json:"ip,omitempty"`
	Imei     string `json:"imei,omitempty"`
	Imsi     string `json:"imsi,omitempty"`
	Pcid     int    `json:"pcid,omitempty"`
	Tac      int    `json:"tac,omitempty"`
	Rsrp     int    `json:"rsrp,omitempty"`
	Rsrq     int    `json:"rsrq,omitempty"`
	Rssi     int    `json:"rssi,omitempty"`
	Cqi      int    `json:"cqi,omitempty"`
	Ri       int    `json:"ri,omitempty"`
	Rb       int    `json:"rb,omitempty"`
	Mcs      int64  `json:"mcs,omitempty"`
	McsTx    int64  `json:"mcstx,omitempty"`
	McsRx    int64  `json:"mcsrx,omitempty"`
	BpsTx    string `json:"bps_tx,omitempty"`
	BpsTxAvg int64  `json:"bps_tx_avg,omitempty"`
	BpsTxSum int64  `json:"bps_tx_sum,omitempty"`
	BpsRx    string `json:"bps_rx,omitempty"`
	BpsRxAvg int64  `json:"bps_rx_avg,omitempty"`
	BpsRxSum int64  `json:"bps_rx_sum,omitempty"`

	Ant0Power float32 `json:"ant0_power,omitempty"`
	Ant0Ecio  float32 `json:"ant0_ecio,omitempty"`
	Ant0Rsrp  float32 `json:"ant0_rsrp,omitempty"`

	Ant1Power float32 `json:"ant1_power,omitempty"`
	Ant1Ecio  float32 `json:"ant1_ecio,omitempty"`
	Ant1Rsrp  float32 `json:"ant1_rsrp,omitempty"`

	Ant2Power float32 `json:"ant2_power,omitempty"`
	Ant2Ecio  float32 `json:"ant2_ecio,omitempty"`
	Ant2Rsrp  float32 `json:"ant2_rsrp,omitempty"`

	Ant3Power float32 `json:"ant3_power,omitempty"`
	Ant3Ecio  float32 `json:"ant3_ecio,omitempty"`
	Ant3Rsrp  float32 `json:"ant3_rsrp,omitempty"`
}

type Position struct {
	PosX   string `json:"pos_x,omitempty"`
	PosY   string `json:"pos_y,omitempty"`
	Angle  string `json:"heading,omitempty"`
	Status string `json:"status,omitempty"`
}

func ParseTimeToSeconds(timeStr string) (int64, error) {
	parts := strings.Fields(timeStr) // 문자열을 공백을 기준으로 분리

	var totalSeconds int64
	for _, part := range parts {
		valueTmp, err := strconv.Atoi(part[:len(part)-1]) // 숫자 부분을 추출하고 정수로 변환
		if err != nil {
			return 0, err
		}
		value := int64(valueTmp)

		switch part[len(part)-1] { // d, h, m, s를 기준으로 단위를 선택
		case 'd':
			totalSeconds += value * 24 * 60 * 60 // 일 단위를 초 단위로 변환
		case 'h':
			totalSeconds += value * 60 * 60 // 시간 단위를 초 단위로 변환
		case 'm':
			totalSeconds += value * 60 // 분 단위를 초 단위로 변환
		case 's':
			totalSeconds += value // 초 단위
		default:
			return 0, fmt.Errorf("Unknown time unit: %s", string(part[len(part)-1]))
		}
	}

	return totalSeconds, nil
}
