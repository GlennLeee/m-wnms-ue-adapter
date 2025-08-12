package model

type ChMsgInfo struct {
	Topic   string
	Message interface{}
	Qos     byte
}

type GrpcServiceMsg struct {
	Method  string
	Message *RpcRequest
}

var GpubChannel chan ChMsgInfo
var GrpcServiceChannel chan GrpcServiceMsg
var GrpcServiceBackupChannel chan GrpcServiceMsg
