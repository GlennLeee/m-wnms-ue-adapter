package message

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

type GrpcService struct {
	c     model.WnmsDataCollectorClient
	conn  *grpc.ClientConn
	mutex *sync.Mutex
	host  string
}

var kacp = keepalive.ClientParameters{
	Time:                2 * time.Minute,
	Timeout:             10 * time.Second,
	PermitWithoutStream: false,
}

var grpcMap = map[string]string{
	"UpdateP5gStatsData":   "UpdateP5GStatsData",
	"UpdateP5gUeStatsData": "UpdateP5GUeStatsData",
	"UpdateAlarmData":      "UpdateAlarmData",
	"UpdateEventData":      "UpdateEventData",
	"Ping":                 "PingData",
}

func InitializeGrpcServer() *GrpcService {
	collectorHost := os.Getenv("DATA_COLLECTOR_HOST")

	log.Println("Try to connect Gprc Server [", collectorHost, "] ...")

	grpcConn := &GrpcService{}

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			InitializeGrpcServer()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	defer cancel()

	conn, err := grpc.DialContext(ctx, collectorHost,
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	grpcConn.conn = conn
	grpcConn.c = model.NewWnmsDataCollectorClient(conn)
	grpcConn.mutex = &sync.Mutex{}
	grpcConn.host = collectorHost

	go grpcConn.sendGrpcRequest()
	go grpcConn.checkAndReconnectGrpcServer()

	return grpcConn
}

// reconnect logic
func (g *GrpcService) checkAndReconnectGrpcServer() {
	// 혹시 모를 panic에 대비하여....
	defer func() {
		if err := recover(); err != nil {
			g.checkAndReconnectGrpcServer()
		}
	}()

	// 본 로직 시작
	for {
		// wait 500 ms
		time.Sleep(time.Millisecond * 200)
		if g.conn.GetState() == connectivity.Ready {
			continue
		} else if g.conn.GetState() == connectivity.Idle {
			log.Println("Try to re-connect Gprc Server [", g.host, "] ...")
			g.conn.ResetConnectBackoff()
			g.conn.Connect()
		}
	}
}

func (g *GrpcService) SendPing() {
	rpcRequest := model.RpcRequest{}
	rpcRequest.Message = fmt.Sprintf("PING")

	model.GrpcServiceChannel <- model.GrpcServiceMsg{
		Method:  "Ping",
		Message: &rpcRequest,
	}
}

func (g *GrpcService) sendGrpcRequest() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			go g.sendGrpcRequest()
		}
	}()

	log.Println("Start GRPC Module")

	for {
		if g.conn.GetState() != connectivity.Ready {
			time.Sleep(time.Second)
			continue
		}

		var msg model.GrpcServiceMsg

		if len(model.GrpcServiceBackupChannel) > 0 {
			msg = <-model.GrpcServiceBackupChannel
		} else {
			msg = <-model.GrpcServiceChannel
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		if grpcMethodName, ok := grpcMap[msg.Method]; ok {

			refRet := reflect.ValueOf(g.c).MethodByName(grpcMethodName).
				Call([]reflect.Value{
					reflect.ValueOf(ctx),
					reflect.ValueOf(msg.Message),
				},
				)

			// r := refRet[0].Interface().(*model.RpcReply)
			errIntf := refRet[1].Interface()

			if errIntf != nil {
				err := errIntf.(error)
				if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "connection error") {
					log.Println("Add Backup Channel")
					log.Println(msg)
					model.GrpcServiceBackupChannel <- msg
					panic(err)
				} else {
					// server response
				}
			}
			// log.Printf("Recv Response of %s -> %s: %v", msg.Method, grpcMethodName, r)
		} else {
			log.Println(msg.Method + " not exist.")
		}
		cancel()
	}
}
