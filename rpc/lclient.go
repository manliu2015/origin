package rpc

import (
	"errors"
	"github.com/duanhf2012/origin/log"
	"github.com/duanhf2012/origin/network"
	"reflect"
	"strings"
	"sync/atomic"
)

//本结点的Client
type LClient struct {
	selfClient *Client
}

func (rc *LClient) Lock(){
}

func (rc *LClient) Unlock(){
}

func (lc *LClient) Run(){
}

func (lc *LClient) OnClose(){
}

func (lc *LClient) IsConnected() bool {
	return true
}

func (lc *LClient) SetConn(conn *network.TCPConn){
}

func (lc *LClient) Close(waitDone bool){
}

func (lc *LClient) Go(rpcHandler IRpcHandler,noReply bool, serviceMethod string, args interface{}, reply interface{}) *Call {
	pLocalRpcServer := rpcHandler.GetRpcServer()()
	//判断是否是同一服务
	findIndex := strings.Index(serviceMethod, ".")
	if findIndex == -1 {
		sErr := errors.New("Call serviceMethod " + serviceMethod + " is error!")
		log.SError(sErr.Error())
		call := MakeCall()
		call.Err = sErr
		return call
	}

	serviceName := serviceMethod[:findIndex]
	if serviceName == rpcHandler.GetName() { //自己服务调用
		//调用自己rpcHandler处理器
		err := pLocalRpcServer.myselfRpcHandlerGo(lc.selfClient,serviceName, serviceMethod, args, requestHandlerNull,reply)
		call := MakeCall()
		if err != nil {
			call.Err = err
			return call
		}

		call.done<-call
		return call
	}

	//其他的rpcHandler的处理器
	return pLocalRpcServer.selfNodeRpcHandlerGo(nil, lc.selfClient, noReply, serviceName, 0, serviceMethod, args, reply, nil)
}


func (rc *LClient) RawGo(rpcHandler IRpcHandler,processor IRpcProcessor, noReply bool, rpcMethodId uint32, serviceName string, rawArgs []byte, reply interface{}) *Call {
	pLocalRpcServer := rpcHandler.GetRpcServer()()

	call := MakeCall()
	call.ServiceMethod = serviceName
	call.Reply = reply

	//服务自我调用
	if serviceName == rpcHandler.GetName() {
		err := pLocalRpcServer.myselfRpcHandlerGo(rc.selfClient,serviceName, serviceName, rawArgs, requestHandlerNull,nil)
		call.Err = err
		call.done <- call

		return call
	}

	//其他的rpcHandler的处理器
	return pLocalRpcServer.selfNodeRpcHandlerGo(processor,rc.selfClient, true, serviceName, rpcMethodId, serviceName, nil, nil, rawArgs)
}


func (lc *LClient) AsyncCall(rpcHandler IRpcHandler, serviceMethod string, callback reflect.Value, args interface{}, reply interface{}) error {
	pLocalRpcServer := rpcHandler.GetRpcServer()()

	//判断是否是同一服务
	findIndex := strings.Index(serviceMethod, ".")
	if findIndex == -1 {
		err := errors.New("Call serviceMethod " + serviceMethod + " is error!")
		callback.Call([]reflect.Value{reflect.ValueOf(reply), reflect.ValueOf(err)})
		log.SError(err.Error())
		return nil
	}

	serviceName := serviceMethod[:findIndex]
	//调用自己rpcHandler处理器
	if serviceName == rpcHandler.GetName() { //自己服务调用
		return pLocalRpcServer.myselfRpcHandlerGo(lc.selfClient,serviceName, serviceMethod, args,callback ,reply)
	}

	//其他的rpcHandler的处理器
	err := pLocalRpcServer.selfNodeRpcHandlerAsyncGo(lc.selfClient, rpcHandler, false, serviceName, serviceMethod, args, reply, callback)
	if err != nil {
		callback.Call([]reflect.Value{reflect.ValueOf(reply), reflect.ValueOf(err)})
	}

	return nil
}

func NewLClient(nodeId int) *Client{
	client := &Client{}
	client.clientId = atomic.AddUint32(&clientSeq, 1)
	client.nodeId = nodeId
	client.maxCheckCallRpcCount = MaxCheckCallRpcCount
	client.callRpcTimeout = DefaultRpcTimeout

	lClient := &LClient{}
	lClient.selfClient = client
	client.IRealClient = lClient
	client.InitPending()
	go client.checkRpcCallTimeout()
	return client
}
