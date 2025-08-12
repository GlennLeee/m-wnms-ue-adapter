package collector

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/database"
	"connectlabs.co.kr/wnms/wnms-ue-adapter/model"
	"connectlabs.co.kr/wnms/wnms-ue-adapter/util"
)

type PmManager struct {
	db        *database.RdbConf
	configKey string
}

func (p *PmManager) Run(db *database.RdbConf) {
	// p.inserted = make(map[string]string)
	p.db = db
	p.checkNetActServer()

	p.configKey = "p5g_netact_sftp_last_" + os.Getenv("server.number")

	fileList, _, err := p.getPmFilePathList()
	if err != nil {
		return
	}

	for _, filePath := range fileList {
		log.Println("READ START", filePath)
		xmlData, err := p.getPmFile(filePath)
		if err != nil {
			log.Println(err)
			continue
		}
		dataResult, fileTimestamp, endTimestamp := p.loadXmlByte(xmlData)
		p.runCalc(dataResult, fileTimestamp, endTimestamp)

		// t := fileTimeList[idx]
		// _, err = p.db.C.Exec("UPDATE wnms_config SET value = ? WHERE key = '"+p.configKey+"'", t)
		// if err != nil {
		// 	log.Println(err)
		// 	continue
		// }

	}
}

func (p *PmManager) insertStatus(status int, desc string) {
	host := os.Getenv("p5g_netact_sftp_host")
	ipAddr := strings.Split(host, ":")[0]

	// db 저장
	millis := time.Now().UnixMilli()

	_, err := p.db.C.Exec("INSERT INTO external_system_info (id, timestamp, ipv4, device_type, status, `desc`, device_name, hostname) VALUES(?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE timestamp=?,ipv4=?, device_type=?, status=?, `desc`=?, device_name=?, hostname=?",
		host, millis, ipAddr, "p5g nms", status, desc, host, host, millis, ipAddr, "p5g nms", status, desc, host, host)

	if err != nil {
		log.Println(err)
	}
}

func (p *PmManager) checkNetActServer() {
	log.Println("START NETACT CHECKER")
	// SFTP 서버 정보
	host := os.Getenv("p5g_netact_sftp_host")
	username := os.Getenv("p5g_netact_sftp_username")
	password := os.Getenv("p5g_netact_sftp_password")

	// SSH 클라이언트 설정
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 보안을 위해 실제 환경에서는 사용하지 않는 것이 좋습니다.
	}

	// SSH 클라이언트 연결
	conn, err := ssh.Dial("tcp", host, config)
	if err != nil {
		fmt.Printf("Failed to connect to SSH server: %v\n", err)
		p.insertStatus(3, "Failed to connect to SSH server")
		return
	}
	defer conn.Close()

	// SFTP 클라이언트 생성
	client, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Printf("Failed to create SFTP client: %v\n", err)
		p.insertStatus(3, "Failed to create SFTP client")
		return
	}
	defer client.Close()

	// SFTP 서버의 디렉토리 목록 가져오기
	_, err = client.ReadDir(os.Getenv("p5g_netact_sftp_path"))
	if err != nil {
		fmt.Printf("Failed to list directory: %v\n", err)
		p.insertStatus(3, "Failed to list directory")
		return
	}

	p.insertStatus(1, "Nokia Private 5G NMS (NetAct)")

}

func (p *PmManager) getPmFilePathList() ([]string, []string, error) {
	result := make([]string, 0)
	resultTime := make([]string, 0)

	// 마지막 기록 읽어오기
	lastTime := 0
	rows, err := p.db.C.Query("SELECT `key`, `value` FROM wnms_config WHERE `key` = '" + p.configKey + "'")
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		lastTime, err = strconv.Atoi(v)
		if err != nil {
			log.Println(err)
			return nil, nil, err
		}
	}

	// SFTP 서버 연결 정보 설정
	config := &ssh.ClientConfig{
		User: os.Getenv("p5g_netact_sftp_username"),
		Auth: []ssh.AuthMethod{
			ssh.Password(os.Getenv("p5g_netact_sftp_password")),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// SSH 클라이언트 연결
	sshClient, err := ssh.Dial("tcp", os.Getenv("p5g_netact_sftp_host"), config)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	defer sshClient.Close()

	// SFTP 클라이언트 생성
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		log.Println(err)
		return nil, nil, err

	}
	defer sftpClient.Close()

	directories, err := sftpClient.ReadDir(os.Getenv("p5g_netact_sftp_path"))

	if err != nil {
		log.Println("--> ", err)
		return nil, nil, err
	}

	directoryList := make([]string, 0)
	for _, directory := range directories {
		if !directory.IsDir() {
			continue
		}

		directoryList = append(directoryList, directory.Name())
	}

	// asc sort directory list
	sort.Slice(directoryList, func(i, j int) bool {
		return directoryList[i] < directoryList[j]
	})

	// get all directories
	for _, directory := range directoryList {
		directoryNameSpl := strings.Split(directory, "_")
		if len(directoryNameSpl) != 2 {
			continue
		}
		directoryName := directoryNameSpl[0]
		dirTime, err := strconv.Atoi(directoryName)
		if err != nil {
			continue
		}

		if dirTime <= lastTime {
			continue
		}

		targetPath := os.Getenv("p5g_netact_sftp_path") + "/" + directory
		files, err := sftpClient.ReadDir(targetPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if strings.HasSuffix(file.Name(), ".xml.gz") && strings.Contains(file.Name(), "MRBTS") {
				filePath := targetPath + "/" + file.Name()
				result = append(result, filePath)
				resultTime = append(resultTime, directoryName)
			}
		}
	}

	return result, resultTime, nil
}

func (p *PmManager) getPmFile(filePath string) ([]byte, error) {
	// SFTP 서버 연결 정보 설정
	config := &ssh.ClientConfig{
		User: os.Getenv("p5g_netact_sftp_username"),
		Auth: []ssh.AuthMethod{
			ssh.Password(os.Getenv("p5g_netact_sftp_password")),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// SSH 클라이언트 연결
	sshClient, err := ssh.Dial("tcp", os.Getenv("p5g_netact_sftp_host"), config)
	if err != nil {
		log.Println(err)
		return nil, err

	}
	defer sshClient.Close()

	// SFTP 클라이언트 생성
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		log.Println(err)
		return nil, err

	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Open(filePath)
	if err != nil {
		log.Println(filePath, "->", err)
		return nil, err

	}
	defer remoteFile.Close()
	fileBytes, err := io.ReadAll(remoteFile)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	reader := bytes.NewReader(fileBytes)
	gzreader, e1 := gzip.NewReader(reader)
	if e1 != nil {
		fmt.Println(e1) // Maybe panic here, depends on your error handling.
		return nil, e1
	}

	output, e2 := io.ReadAll(gzreader)
	if e2 != nil {
		fmt.Println(e2)
		return nil, e2
	}

	// result := string(output)
	// log.Println(result)
	return output, nil
}

func (p *PmManager) loadXmlByte(data []byte) (map[string]Data, int64, int64) {
	dataResult := make(map[string]Data)

	// xml 디코딩
	var measCollecFile model.MeasCollecFile
	xmlerr := xml.Unmarshal(data, &measCollecFile)
	if xmlerr != nil {
		panic(xmlerr)
	}

	measData := measCollecFile.MeasData
	// targetTimestamp := int64(0)

	measCollec := measCollecFile.FileHeader.MeasCollec
	timeString := measCollec.BeginTime
	// timeString := measCollec.EndTime
	targetDate, err := time.Parse("2006-01-02T15:04:05-07:00", timeString)
	if err != nil {
		log.Println(err)
	}

	targetTimestamp := targetDate.UnixMilli()
	endTimestamp := int64(0)

	for _, measInfo := range measData.MeasInfo {
		// granPeriod := measInfo.GranPeriod
		// if targetTimestamp == 0 && granPeriod.Duration == "PT900S" {
		// 	timeString := granPeriod.EndTime
		// 	targetDate, err := time.Parse("2006-01-02T15:04:05-07:00", timeString)
		// 	if err != nil {
		// 		log.Println(err)
		// 	}

		// 	targetTimestamp = targetDate.UnixMilli()
		// }

		timeString := measInfo.GranPeriod.EndTime
		targetDate, err := time.Parse("2006-01-02T15:04:05-07:00", timeString)
		if err != nil {
			log.Println(err)
		}
		endTimestamp = targetDate.UnixMilli()

		dvcNameSpl := strings.Split(measInfo.MeasValue.MeasObjLdn, ",")
		if len(dvcNameSpl) == 0 {
			continue
		}
		// dvcName := measInfo.MeasValue.MeasObjLdn
		dvcName := dvcNameSpl[0]
		values := measInfo.MeasValue.MeasResults
		keys := measInfo.MeasTypes

		keyArr := strings.Split(keys, " ")
		valueArr := strings.Split(values, " ")

		if len(keyArr) != len(valueArr) {
			panic("데이터 길이가 다름")
		}

		if v, ok := dataResult[dvcName]; ok {
			for idx, dptr := range keyArr {
				if _, ok := neededValue[dptr]; ok {
					if _, ok := v[dptr]; ok {
						oldV, err := strconv.ParseFloat(v[dptr].(string), 32)
						if err == nil {
							newV, err := strconv.ParseFloat(valueArr[idx], 32)
							if err == nil {
								v[dptr] = fmt.Sprintf("%f", oldV+newV)
							}
						}
					} else {
						v[dptr] = valueArr[idx]
					}
				}
			}
		} else {
			v := make(Data)
			for idx, dptr := range keyArr {
				if _, ok := neededValue[dptr]; ok {
					if _, ok := v[dptr]; ok {
						oldV, err := strconv.ParseFloat(v[dptr].(string), 32)
						if err == nil {
							newV, err := strconv.ParseFloat(valueArr[idx], 32)
							if err == nil {
								v[dptr] = fmt.Sprintf("%f", oldV+newV)
							}
						}
					} else {
						v[dptr] = valueArr[idx]
					}
				}
			}
			dataResult[dvcName] = v
		}
	}
	// log.Println(dataResult)
	log.Println("LOAD XML FILE - START TIMESTAMP: ", targetTimestamp, ", END TIME: ", endTimestamp)
	return dataResult, targetTimestamp, endTimestamp
}

func (p *PmManager) runCalc(dataResult map[string]Data, timestamp int64, endTimestamp int64) {
	rpcRequest := model.RpcRequest{}
	rpcRequest.Timestamp = timestamp

	mc := regexp.MustCompile("[+/*-/(/)]")
	// fomulaChecker := regexp.MustCompile("[a-zA-Z]")
	p5gStatsReport := model.P5GStatsReport{}
	p5gStatsReport.Timestamp = uint64(rpcRequest.GetTimestamp())

	for k, v := range dataResult {
		p5gStats := model.P5GStats{}
		p5gStats.Id = k

		formulaResult := make(map[string]string)

		for fk, fv := range nFormula {
			t := mc.Split(fv, -1)
			trimmedFormula := fv

			for _, target := range t {
				if target == "" {
					continue
				}

				if targetValue, ok := v[target]; ok {
					trimmedFormula = strings.ReplaceAll(trimmedFormula, target, targetValue.(string))
				} else {
					if targetValue, ok := numberMap[target]; ok {
						trimmedFormula = strings.ReplaceAll(trimmedFormula, target, targetValue)
					} else {
						trimmedFormula = strings.ReplaceAll(trimmedFormula, target, "0")
					}
				}
			}

			result, err := util.Calcuate(trimmedFormula)

			if err != nil {
				result = 0
			}

			formulaResult[fk] = fmt.Sprintf("%v", result)
		}
		formulaResult["vendor"] = "NOKIA"
		formulaResult["period"] = fmt.Sprintf("%d", endTimestamp-timestamp)
		p5gStats.Data = formulaResult
		p5gStatsReport.P5GStats = append(p5gStatsReport.P5GStats, &p5gStats)
	}

	rpcRequest.P5GStatsReport = &p5gStatsReport

	model.GrpcServiceChannel <- model.GrpcServiceMsg{
		Method:  "UpdateP5gStatsData",
		Message: &rpcRequest,
	}
}
