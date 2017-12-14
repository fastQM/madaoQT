package websocket

import (
	"encoding/json"
)

type ErrorType int

const CmdTypeSubscribe = "subscribe"
const CmdTypeUnsubscribe = "unsubscribe"
const CmdTypePublish = "publish"

const (
	ErrorTypeNone ErrorType = iota
	ErrorTypeInvalidCmd
	ErrorTypeNum
)

type RequestMsg struct {
	Seq     int         `json:"seq"`
	Cmd     string      `json:"cmd"`
	Channel string      `json:"channel,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ResponseMsg struct {
	Seq int `json:"seq"`
	// Cmd    CmdType     `json:"cmd"`
	Result bool        `json:"result"`
	Error  ErrorType   `json:"error,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func parseRequestMsg(message string) *RequestMsg {
	var data RequestMsg
	err := json.Unmarshal([]byte(message), &data)
	if err != nil {
		Logger.Errorf("Fail to parse message:%v", err)
		return nil
	}
	return &data
}

func packageRequestMsg() {

}

func packageResponseMsg(seq int, result bool, errorCode ErrorType, data interface{}) []byte {
	var rsp ResponseMsg
	if result {
		rsp = ResponseMsg{
			Seq:    seq,
			Result: result,
			Data:   data,
		}
	} else {
		rsp = ResponseMsg{
			Seq:    seq,
			Result: result,
			Error:  errorCode,
			Data:   data,
		}
	}
	msg, err := json.Marshal(rsp)
	if err != nil {
		Logger.Errorf("Fail to packageResponseMsg:%v", err)
		return nil
	}
	return msg
}
