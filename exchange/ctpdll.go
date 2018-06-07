package exchange

import (
	"encoding/json"
	"log"
	"syscall"
	"unsafe"
)

// char -->  C.char -->  byte
// signed char -->  C.schar -->  int8
// unsigned char -->  C.uchar -->  uint8
// short int -->  C.short -->  int16
// short unsigned int -->  C.ushort -->  uint16
// int -->  C.int -->  int
// unsigned int -->  C.uint -->  uint32
// long int -->  C.long -->  int32 or int64
// long unsigned int -->  C.ulong -->  uint32 or uint64
// long long int -->  C.longlong -->  int64
// long long unsigned int -->  C.ulonglong -->  uint64
// float -->  C.float -->  float32
// double -->  C.double -->  float64
// wchar_t -->  C.wchar_t  -->
// void * -> unsafe.Pointer

const (
	CTPStatusNone = 0
	CTPStatusReady
	CTPStatusProcess
	CTPStatusDone
	CTPStatusDisconnect
	CTPStatusError
)

type CTPDll struct {
	Dll *syscall.LazyDLL
}

// var dll *syscall.LazyDLL

func (p *CTPDll) SetConfig(config string) bool {
	function := p.Dll.NewProc("Config")
	if function == nil {
		log.Printf("Invaid functions")
		return false
	}
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(config)[0])))
	if err != nil {
		// log.Printf("Err:%v", err)
	}

	if result == 1 {
		return true
	}

	return false
}

func (p *CTPDll) InitMarket() bool {
	function := p.Dll.NewProc("InitMarket")
	result, _, err := function.Call()
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}
	if result == 1 {
		return true
	}
	return false
}

func (p *CTPDll) InitTrade() bool {
	function := p.Dll.NewProc("InitTrade")
	result, _, err := function.Call()
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}
	if result == 1 {
		return true
	}
	return false
}

func (p *CTPDll) CloseMarket() bool {
	function := p.Dll.NewProc("CloseMarket")
	result, _, err := function.Call()
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}
	if result == 1 {
		return true
	}
	return false
}

func (p *CTPDll) CloseTrade() bool {
	function := p.Dll.NewProc("CloseTrade")
	result, _, err := function.Call()
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}
	if result == 1 {
		return true
	}
	return false
}

func (p *CTPDll) GetDepth(instrument string) map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("GetDepth")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])), uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}

		// logger.Infof("GetDepth:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) GetInstrumentInfo(instrument string) map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("GetInstrumentInfo")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])), uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}
		logger.Infof("GetInstrumentInfo:%v", values)
		return values
	}

	logger.Errorf("无效商品信息:%s", instrument)
	return nil
}

func (p *CTPDll) GetPositionInfo(instrument string) map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("GetPositionInfo")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])), uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}
		logger.Infof("GetPositionInfo:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) GetBalance() map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("GetBalance")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}
		logger.Infof("GetBalance:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) MarketOpenPosition(instrument string, volume int, price int, isBuy int, isMarket int) map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("MarketOpenPosition")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])),
		uintptr(volume),
		uintptr(price),
		uintptr(isBuy),
		uintptr(isMarket),
		uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}
		logger.Infof("MarketOpenPosition:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) MarketClosePosition(instrument string, volume int, price int, isBuy int, isMarket int) map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("MarketClosePosition")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])),
		uintptr(volume),
		uintptr(price),
		uintptr(isBuy),
		uintptr(isMarket),
		uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}
		logger.Infof("MarketClosePosition:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) GetStatus() uintptr {
	function := p.Dll.NewProc("GetStatus")
	result, _, _ := function.Call()
	return uintptr(result)
}
