package collector

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/model"
	g "github.com/ruraomsk/gosnmp"
)

type FmManager struct{}

var secParamsList = []*g.UsmSecurityParameters{
	{
		UserName:                 "snmpnbuser",
		AuthenticationProtocol:   g.SHA,
		AuthenticationPassphrase: "snmpAuthPasswd",
		PrivacyProtocol:          g.AES,
		PrivacyPassphrase:        "snmpPrivacyPasswd",
	},
}

func (f *FmManager) FmListenTrapSet() {
	log.Println("START AM/FM SNMP LISTNER START")

	tl := g.NewTrapListener()

	usmTable := g.NewSnmpV3SecurityParametersTable(g.NewLogger(log.New(os.Stdout, "", 0)))
	for _, sp := range secParamsList {
		err := usmTable.Add(sp.UserName, sp)
		if err != nil {
			usmTable.Logger.Print(err)
		}
	}

	gs := &g.GoSNMP{
		Port:                        162,
		Transport:                   "udp",
		Version:                     g.Version3, // Always using version3 for traps, only option that works with all SNMP versions simultaneously
		SecurityModel:               g.UserSecurityModel,
		SecurityParameters:          &g.UsmSecurityParameters{AuthoritativeEngineID: "12345"}, // Use for server's engine ID
		TrapSecurityParametersTable: usmTable,
	}

	tl.Params = gs
	tl.Params.Logger = g.NewLogger(log.New(os.Stdout, "", 0))

	tl.OnNewTrap = f.fmHandler

	// 포트 설정을 properties.ini로...
	err := tl.Listen("0.0.0.0:162")
	if err != nil {
		log.Printf("error in listen: %s", err)
	}
}

/*
NSN-SNMP-NBI-COMMONFUNCTIONS-MIB::nbiSequenceId=214
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiAlarmId="441"
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiAlarmType=environmentalAlarm
NSN-SNMP-NBI-COMMONFUNCTIONS-MIB::nbiObjectInstance="PLMN-nbisnmp/RNC-nbisnmp/WBTS-nbisnmp"
NSN-SNMP-NBI-COMMONFUNCTIONS-MIB::nbiEventTime="2015-12-10,4:24:50.1,+2:0"
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiAlarmTime="2015-12-10,4:23:43.0,+2:0"
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiProbableCause=107
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiSpecificProblem="30163|FAULT IN COOLING SYSTEM"
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiPerceivedSeverity=major
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiProposedRepairAction=""
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiAdditionalText="Check if fan is functioning 1|Check if fan is functioning 2|Check if fan is functioning 3|Check if fan is functioning 4|Check if fan is functioning 5|Check if fan is functioning 6|Check if fan is functioning 7"
NSN-SNMP-NBI-FAULTMANAGEMENT-MIB::nbiOptionalInformation="NEName=WBTSNameInNasda|originalAlarmId=1449714223418917808|maintenanceRegion=MRNameInNasda|siteObjName=SITENameInNasda|siteObjDN=SITEC-nbisnmp/SITE-nbisnmp|siteObjAddress=SITEAddressInNasda|controlObjName=RNCNameInNasda|controlObjSiteAddress=SITEAddressInNasda|controlObjMR=M₩RNameInNasda"
*/

const (
	uptime                  = ".1.3.6.1.2.1.1.3.0"
	snmpTrapOID             = ".1.3.6.1.6.3.1.1.4.1.0"
	nbiAlarmId              = ".1.3.6.1.4.1.28458.1.26.3.1.1.5"
	nbiSequenceId           = ".1.3.6.1.4.1.28458.1.26.2.1.3.9"
	nbiAlarmType            = ".1.3.6.1.4.1.28458.1.26.3.1.1.7"
	nbiObjectInstance       = ".1.3.6.1.4.1.28458.1.26.2.1.6.5"
	nbiEventTime            = ".1.3.6.1.4.1.28458.1.26.2.1.6.3"
	nbiAlarmTime            = ".1.3.6.1.4.1.28458.1.26.3.1.1.6"
	nbiProbableCause        = ".1.3.6.1.4.1.28458.1.26.3.1.1.14"
	nbiSpecificProblem      = ".1.3.6.1.4.1.28458.1.26.3.1.1.16"
	nbiPerceivedSeverity    = ".1.3.6.1.4.1.28458.1.26.3.1.1.13"
	nbiProposedRepairAction = ".1.3.6.1.4.1.28458.1.26.3.1.1.15"
	nbiAdditionalText       = ".1.3.6.1.4.1.28458.1.26.3.1.1.4"
	nbiOptionalInformation  = ".1.3.6.1.4.1.28458.1.26.3.1.1.19"
	nbiClearSystemId        = ".1.3.6.1.4.1.28458.1.26.3.1.1.18"
	nbiClearTime            = ".1.3.6.1.4.1.28458.1.26.3.1.1.8"
	nbiClearUser            = ".1.3.6.1.4.1.28458.1.26.3.1.1.9"
	nbiAckState             = ".1.3.6.1.4.1.28458.1.26.3.1.1.1"
	nbiAckSystemId          = ".1.3.6.1.4.1.28458.1.26.3.1.1.17"
	nbiAckTime              = ".1.3.6.1.4.1.28458.1.26.3.1.1.2"
	nbiAckUser              = ".1.3.6.1.4.1.28458.1.26.3.1.1.3"
)

// const (
// 	nbiAlarmNewNotification        = [2]string{".1.3.6.1.4.1.28458.1.26.3.0.1.1", "New"}
// 	nbiAlarmAckChangedNotification = [2]string{".1.3.6.1.4.1.28458.1.26.3.0.1.2", "Ack"}
// 	nbiAlarmClearedNotification    = [2]string{".1.3.6.1.4.1.28458.1.26.3.0.1.5", "Clear"}
// 	nbiAlarmSyncNotification       = [2]string{".1.3.6.1.4.1.28458.1.26.3.0.2.1", "Sync"}
// )

var nbiAlarmMessageType = map[string]string{
	".1.3.6.1.4.1.28458.1.26.3.0.1.1": "OUTSTANDING",
	".1.3.6.1.4.1.28458.1.26.3.0.1.2": "PROCESSING",
	".1.3.6.1.4.1.28458.1.26.3.0.1.5": "CLEARED",
	".1.3.6.1.4.1.28458.1.26.3.0.2.1": "PROCESSING",
}

var nbiAlarmTypeMap = map[int]string{
	1: "communicationsAlarm",
	2: "qualityOfServiceAlarm",
	3: "processingErrorAlarm",
	4: "equipmentAlarm",
	5: "environmentalAlarm",
}

var nbiAlarmSeverityMap = map[int]string{
	0: "Emergency",
	1: "Alert",    // Critical
	2: "Critical", // Major

	3: "Error",   // Minor
	4: "Warning", // Warning
	5: "Notice",
	6: "Informational",
	7: "Debug",
}

type snmpFault struct {
	messageType    string
	uptime         uint32
	id             string // nbiAlarmId 1.3.6.1.4.1.28458.1.26.3.1.1.5
	seqId          int    // nbiSequenceId 1.3.6.1.4.1.28458.1.26.2.1.3.9
	faultType      string // nbiAlarmType 1.3.6.1.4.1.28458.1.26.3.1.1.7
	objectId       string // nbiObjectInstance 1.3.6.1.4.1.28458.1.26.2.1.6.5
	eventTs        int64  // nbiEventTime 1.3.6.1.4.1.28458.1.26.2.1.6.3
	faultTs        int64  // nbiAlarmTime 1.3.6.1.4.1.28458.1.26.3.1.1.6
	cause          int    // nbiProbableCause 1.3.6.1.4.1.28458.1.26.3.1.1.14
	code           int    // nbiSpecificProblem 1.3.6.1.4.1.28458.1.26.3.1.1.16
	problem        string // nbiSpecificProblem 1.3.6.1.4.1.28458.1.26.3.1.1.16
	category       string
	severity       int    // nbiPerceivedSeverity 1.3.6.1.4.1.28458.1.26.3.1.1.13
	proposedAction string // nbiProposedRepairAction 1.3.6.1.4.1.28458.1.26.3.1.1.15
	description    string // nbiAdditionalText 1.3.6.1.4.1.28458.1.26.3.1.1.4
	optInfo        string // nbiOptionalInformation 1.3.6.1.4.1.28458.1.26.3.1.1.19
	clearSystemId  string // nbiClearSystemId 1.3.6.1.4.1.28458.1.26.3.1.1.18
	clearTs        int64  // nbiClearTime 1.3.6.1.4.1.28458.1.26.3.1.1.8
	clearUser      string // nbiClearUser 1.3.6.1.4.1.28458.1.26.3.1.1.9
	ackState       int    // nbiAckState 1.3.6.1.4.1.28458.1.26.3.1.1.1
	actSystemId    string // nbiAckSystemId 1.3.6.1.4.1.28458.1.26.3.1.1.17
	actTs          int64  // nbiAckTime 1.3.6.1.4.1.28458.1.26.3.1.1.2
	actUser        string // nbiAckUser 1.3.6.1.4.1.28458.1.26.3.1.1.3
}

func convertTime(str string) int64 {
	spl := strings.Split(str, ",")
	dateStr := spl[0]
	timeStr := spl[1]
	tzStr := spl[2]

	dateSpl := strings.Split(dateStr, "-")
	year, _ := strconv.Atoi(dateSpl[0])
	month, _ := strconv.Atoi(dateSpl[1])
	day, _ := strconv.Atoi(dateSpl[2])

	timeStr = strings.ReplaceAll(timeStr, ".", ":")
	timeSpl := strings.Split(timeStr, ":")
	hour, _ := strconv.Atoi(timeSpl[0])
	mins, _ := strconv.Atoi(timeSpl[1])
	secs, _ := strconv.Atoi(timeSpl[2])
	msecs, _ := strconv.Atoi(timeSpl[3])

	tzSql := strings.Split(tzStr, ":")
	tzh, _ := strconv.Atoi(tzSql[0])
	tzF := ""
	if tzh >= 0 {
		tzF = fmt.Sprintf("+%02d:00", tzh)
	} else {
		tzF = fmt.Sprintf("-%02d:00", -1*tzh)
	}

	convTimeStr := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%d00 %s", year, month, day, hour, mins, secs, msecs, tzF)

	eventTime, err := time.Parse("2006-01-02 15:04:05.999 -07:00", convTimeStr)
	if err != nil {
		log.Println(err)
		return -1
	}
	return eventTime.UnixMilli()
}

func (f *FmManager) fmHandler(packet *g.SnmpPacket, addr *net.UDPAddr) {
	fault := snmpFault{}
	for _, v := range packet.Variables {
		switch v.Name {
		case uptime:
			value := v.Value.(uint32)
			fault.uptime = value
		case snmpTrapOID:
			value := v.Value.(string)
			if mType, ok := nbiAlarmMessageType[value]; ok {
				fault.messageType = mType
			}
		case nbiAlarmId:
			value := string(v.Value.([]byte))
			fault.id = value
		case nbiSequenceId:
			value := v.Value.(int)
			fault.seqId = value
		case nbiAlarmType:
			value := v.Value.(int)
			if faultType, ok := nbiAlarmTypeMap[value]; ok {
				fault.faultType = faultType
			} else {
				fault.faultType = fmt.Sprintf("%d", value)
			}
		case nbiObjectInstance:
			value := string(v.Value.([]byte))
			fault.objectId = value
		case nbiEventTime:
			value := string(v.Value.([]byte))
			fault.eventTs = convertTime(value)
		case nbiAlarmTime:
			value := string(v.Value.([]byte))
			fault.faultTs = convertTime(value)
		case nbiProbableCause:
			value := v.Value.(int)
			fault.cause = value
		case nbiSpecificProblem:
			value := string(v.Value.([]byte))
			problems := strings.Split(value, "|")
			code, err := strconv.Atoi(problems[0])
			if err != nil {
				fault.code = 0
			} else {
				fault.code = code
			}

			if len(problems) == 2 {
				// fault.problem = problems[1]
				fault.category = problems[1]
			}
		case nbiPerceivedSeverity:
			value := v.Value.(int)
			fault.severity = value
		case nbiProposedRepairAction:
			value := string(v.Value.([]byte))
			fault.proposedAction = value
		case nbiAdditionalText:
			value := string(v.Value.([]byte))
			fault.description = value
		case nbiOptionalInformation:
			value := string(v.Value.([]byte))
			fault.optInfo = value
		case nbiClearSystemId:
			value := string(v.Value.([]byte))
			fault.clearSystemId = value
		case nbiClearTime:
			value := string(v.Value.([]byte))
			fault.clearTs = convertTime(value)
		case nbiClearUser:
			value := string(v.Value.([]byte))
			fault.clearUser = value
		case nbiAckState:
			value := v.Value.(int)
			fault.ackState = value
		case nbiAckSystemId:
			value := string(v.Value.([]byte))
			fault.actSystemId = value
		case nbiAckTime:
			value := string(v.Value.([]byte))
			fault.actTs = convertTime(value)
		case nbiAckUser:
			value := string(v.Value.([]byte))
			fault.actUser = value
		}
	}

	if fault.id == "" || fault.objectId == "" {
		return
	}

	if fault.messageType == "OUTSTANDING" && (fault.code == 0 || fault.category == "") {
		return
	}

	if fault.messageType == "PROCESSING" {
		fault.problem = "U:" + fault.actUser + "_____D:" + fault.category
	} else if fault.messageType == "CLEARED" {
		fault.problem = "U:" + fault.clearUser + "_____D:" + fault.category
	} else if fault.messageType == "OUTSTANDING" {
		fault.problem = fault.category
	}

	if fault.ackState == 2 {
		fault.messageType = "OUTSTANDING"
	}

	if strings.Contains(fault.problem, "U:cleanup") {
		// fault.messageType = "CLEARED"
		return
	}

	sendFaultData(fault)
}

func sendFaultData(fault snmpFault) {
	rpcRequest := model.RpcRequest{}
	rpcRequest.Timestamp = fault.eventTs
	version := uint32(1)

	// error 이상 등급이거나,
	// PROCESSING 및 CLEARED라서 ALARM을 TRACKING 해야하는 경우
	if fault.severity <= 2 || fault.messageType == "PROCESSING" || fault.messageType == "CLEARED" {
		seqId := uint32(fault.seqId)

		alarmReport := model.AlarmReport{
			AlarmCode:     uint32(fault.code),
			AlarmUuid:     &fault.id,
			AlarmSeverity: nbiAlarmSeverityMap[fault.severity],
			AlarmType:     &fault.faultType,
			MainCategory:  "Private 5G",
			SubCategory:   &fault.category,
			BladeId:       &fault.objectId,
			Timestamp:     uint64(rpcRequest.Timestamp),
			Description:   fault.problem,
			Reason:        &fault.description,
			Version:       &seqId,
			Vendor:        "nokia",
			NetworkType:   "p5g",
		}

		if fault.messageType == "OUTSTANDING" {
			alarmReport.AlarmState = 0
		} else if fault.messageType == "PROCESSING" {
			alarmReport.AlarmState = 2
		} else if fault.messageType == "CLEARED" {
			alarmReport.AlarmState = 1
		}

		rpcRequest.AlarmReport = &alarmReport

		model.GrpcServiceChannel <- model.GrpcServiceMsg{
			Method:  "UpdateAlarmData",
			Message: &rpcRequest,
		}
	} else {
		if fault.messageType != "OUTSTANDING" {
			return
		}

		desc := fmt.Sprintf("%s [%s]", fault.category, fault.description)

		ev := model.EventMessage{
			EventCode:    uint32(fault.code),
			EventType:    &fault.faultType,
			Severity:     nbiAlarmSeverityMap[fault.severity],
			ApMac:        &fault.objectId,
			MainCategory: "Private 5G",
			SubCategory:  &fault.category,
			Timestamp:    uint64(rpcRequest.Timestamp),
			Description:  desc,
			Reason:       &fault.description,
			Version:      version,
			Vendor:       "nokia",
			NetworkType:  "p5g",
		}

		el := make([]*model.EventMessage, 0)
		el = append(el, &ev)
		eventReport := model.EventReport{
			EventList: el,
			EventNum:  1,
			Timestamp: fault.eventTs,
		}

		rpcRequest.EventReport = &eventReport
		model.GrpcServiceChannel <- model.GrpcServiceMsg{
			Method:  "UpdateEventData",
			Message: &rpcRequest,
		}
	}
}
