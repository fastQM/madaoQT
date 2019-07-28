package exchange

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	Websocket "github.com/gorilla/websocket"
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
	URL string
}

// var dll *syscall.LazyDLL

type KlineSort []KlineValue

func (p KlineSort) Len() int           { return len(p) }
func (p KlineSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p KlineSort) Less(i, j int) bool { return p[i].OpenTime < p[j].OpenTime }

func send(conn *Websocket.Conn, datas map[string]interface{}) {
	cmd, _ := json.Marshal(datas)
	log.Printf("[%s]Cmd:%v", time.Now().Format("2006-01-02 15:04:05.999999999"), string(cmd))
	conn.WriteMessage(Websocket.TextMessage, cmd)
}

func (p *CTPDll) GetKlines(contract string, intervalMinutes int, count int, randomString string) []KlineValue {

	if contract == "" || intervalMinutes == 0 || count == 0 || randomString == "" {
		logger.Errorf("Invalid parameters")
		return nil
	}

	var klines KlineSort
	duration := float64(1000000000 * intervalMinutes * 60)
	dialer := Websocket.DefaultDialer

	connection, _, err := dialer.Dial(p.URL, nil)

	if err != nil {
		logger.Errorf("Fail to dial:%v", err)
		return nil
	}

	// var contractString string
	// for _, contract := range contracts {
	// 	if contractString == "" {
	// 		contractString = contract
	// 	} else {
	// 		contractString = contractString + "," + contract
	// 	}
	// }

	defer connection.Close()
	step := 0

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			logger.Errorf("Fail to read:%v", err)
			return nil
		}

		var response map[string]interface{}
		if err = json.Unmarshal([]byte(message), &response); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}

		log.Printf("[%s]Reseponse", time.Now().Format("2006-01-02 15:04:05.999999999"))
		if step == 2 || step == 3 {
			data := response["data"].([]interface{})
			if data != nil {

				if data[0].(map[string]interface{})["klines"] == nil {
					logger.Errorf("Fail to get klines")
					return nil
				}

				data := data[0].(map[string]interface{})["klines"].(map[string]interface{})
				if data != nil {
					values := data[contract].(map[string]interface{})
					if values != nil {
						dutrationStr := strconv.Itoa(int(duration))
						datas := values[dutrationStr].(map[string]interface{})
						if datas["data"] != nil {
							for _, data := range datas["data"].(map[string]interface{}) {
								tmp := data.(map[string]interface{})
								klines = append(klines, KlineValue{
									Time:     time.Unix(int64(tmp["datetime"].(float64)/1000000000.0), 0).Format("2006-01-02 15:04:05"),
									OpenTime: tmp["datetime"].(float64) / 1000000000.0,
									Open:     tmp["open"].(float64),
									High:     tmp["high"].(float64),
									Low:      tmp["low"].(float64),
									Close:    tmp["close"].(float64),
									Volumn:   tmp["volume"].(float64),
								})
							}
							sort.Sort(klines)
							return klines
						}
					}
				}
			}

			if step == 3 {
				return nil
			}
		}
		command := make(map[string]interface{})

		switch step {
		case 0:
			step++
			command = map[string]interface{}{
				"aid": "peek_message",
			}
			send(connection, command)
		case 1:
			step++
			command = map[string]interface{}{
				"aid":        "set_chart",  // 必填, 请求图表数据
				"chart_id":   randomString, // 必填, 图表id, 服务器只会维护每个id收到的最后一个请求的数据
				"ins_list":   contract,     // 必填, 填空表示删除该图表，多个合约以逗号分割，第一个合约是主合约，所有id都是以主合约为准
				"duration":   duration,     // 必填, 周期，单位ns, tick:0, 日线: 3600 * 24 * 1000 * 1000 * 1000
				"view_width": count,        // 必填, 图表宽度, 请求最新N个数据，并保持滚动(新K线生成会移动图表)
			}
			send(connection, command)

			time.Sleep(100 * time.Microsecond)

			command = map[string]interface{}{
				"aid": "peek_message",
			}
			send(connection, command)
		case 2:
			step++
			time.Sleep(50 * time.Microsecond)
			command = map[string]interface{}{
				"aid": "peek_message",
			}
			send(connection, command)
		}

	}

	return nil
}

// MAX five contracts for 200 klines
func (p *CTPDll) GetMultipleKlines(contracts []string, intervalMinutes int, count int, randomString string) map[string][]KlineValue {

	if contracts == nil || len(contracts) == 0 || intervalMinutes == 0 || count == 0 || randomString == "" {
		logger.Errorf("Invalid parameters")
		return nil
	}

	klines := make(map[string][]KlineValue)
	duration := float64(1000000000 * intervalMinutes * 60)
	dialer := Websocket.DefaultDialer

	connection, _, err := dialer.Dial(p.URL, nil)

	if err != nil {
		logger.Errorf("Fail to dial:%v", err)
		return nil
	}

	var contractString string
	for _, contract := range contracts {
		if contractString == "" {
			contractString = contract
		} else {
			contractString = contractString + "," + contract
		}
	}

	defer connection.Close()
	step := 0
	var buffer []interface{}

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			logger.Errorf("Fail to read:%v", err)
			return nil
		}

		var response map[string]interface{}

		if err = json.Unmarshal([]byte(message), &response); err != nil {
			logger.Errorf("Fail to Unmarshal:%v", err)
			return nil
		}

		log.Printf("[%s]Reseponse", time.Now().Format("2006-01-02 15:04:05.999999999"))
		if step == 2 || step == 3 {
			data := response["data"].([]interface{})
			// log.Printf("LENGTH:%v", len(data))

			if data != nil && len(data) > 0 { // if == 2, the datas are not ready in the server

				if data[len(data)-2].(map[string]interface{})["charts"] != nil &&
					data[len(data)-2].(map[string]interface{})["charts"].(map[string]interface{})[randomString].(map[string]interface{})["left_id"].(float64) == -1 {
					log.Printf("Invalid data, resend peek")
					command := map[string]interface{}{
						"aid": "peek_message",
					}
					send(connection, command)
					continue
				} else if data[len(data)-2].(map[string]interface{})["mdhis_more_data"] == true {

					log.Printf("more datas reserved, resend peek")
					command := map[string]interface{}{
						"aid": "peek_message",
					}
					send(connection, command)

					for _, tmp := range data[0 : len(data)-2] {
						tmpString, _ := json.Marshal(tmp)
						if strings.Contains(string(tmpString), "binding") {
							continue
						}
						if strings.Count(string(tmpString), "datetime") < 10 { // how about 10?
							continue
						}

						buffer = append(buffer, tmp)
					}

					continue
				} else {
					for _, tmp := range data[0 : len(data)-2] {
						tmpString, _ := json.Marshal(tmp)
						if strings.Contains(string(tmpString), "binding") {
							continue
						}
						if strings.Count(string(tmpString), "datetime") < 10 {
							continue
						}

						buffer = append(buffer, tmp)
					}

					// for key, value := range buffer {
					// 	log.Printf("Key:%v value:%v", key, value)
					// }

					// return nil

					for i, contract := range contracts {
						// log.Printf("[Index %d] %v", i, data[i])
						data := buffer[i].(map[string]interface{})["klines"].(map[string]interface{})
						if data != nil {
							// log.Printf("data:%v contract:%v", data, contract)
							if data[contract] != nil {
								values := data[contract].(map[string]interface{})
								if values != nil {
									dutrationStr := strconv.Itoa(int(duration))
									datas := values[dutrationStr].(map[string]interface{})
									if datas["data"] != nil {
										for _, data := range datas["data"].(map[string]interface{}) {
											tmp := data.(map[string]interface{})
											klines[contract] = append(klines[contract], KlineValue{
												Time:     time.Unix(int64(tmp["datetime"].(float64)/1000000000.0), 0).Format("2006-01-02 15:04:05"),
												OpenTime: tmp["datetime"].(float64) / 1000000000.0,
												Open:     tmp["open"].(float64),
												High:     tmp["high"].(float64),
												Low:      tmp["low"].(float64),
												Close:    tmp["close"].(float64),
												Volumn:   tmp["volume"].(float64),
											})
										}
										sort.Sort(KlineSort(klines[contract]))
									}
								}
							} else {
								return nil
							}

						}
					}

					return klines
				}
			}

		}
		command := make(map[string]interface{})

		switch step {
		case 0:
			step++
			command = map[string]interface{}{
				"aid": "peek_message",
			}
			send(connection, command)
		case 1:
			step++
			command = map[string]interface{}{
				"aid":        "set_chart",    // 必填, 请求图表数据
				"chart_id":   randomString,   // 必填, 图表id, 服务器只会维护每个id收到的最后一个请求的数据
				"ins_list":   contractString, // 必填, 填空表示删除该图表，多个合约以逗号分割，第一个合约是主合约，所有id都是以主合约为准
				"duration":   duration,       // 必填, 周期，单位ns, tick:0, 日线: 3600 * 24 * 1000 * 1000 * 1000
				"view_width": count,          // 必填, 图表宽度, 请求最新N个数据，并保持滚动(新K线生成会移动图表)
			}
			send(connection, command)

			time.Sleep(100 * time.Microsecond)

			command = map[string]interface{}{
				"aid": "peek_message",
			}
			send(connection, command)
		case 2:
			step++
			time.Sleep(100 * time.Microsecond)
			command = map[string]interface{}{
				"aid": "peek_message",
			}
			send(connection, command)
		}

	}

	return nil
}

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
			logger.Errorf("Fail to Unmarshal:%v buffer:%v", err, string(buffer[:result]))
			return nil
		}
		logger.Infof("MarketOpenPosition:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) MarketClosePosition(instrument string, volume int, price int, isBuy int, isMarket int, isToday int) map[string]interface{} {
	buffer := make([]byte, 1024)
	function := p.Dll.NewProc("MarketClosePosition")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])),
		uintptr(volume),
		uintptr(price),
		uintptr(isBuy),
		uintptr(isMarket),
		uintptr(isToday),
		uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v, buffer:%v", err, string(buffer[:result]))
			return nil
		}
		logger.Infof("MarketClosePosition:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) CancelOrder(instrument string, exchangeID string, orderSysID string) map[string]interface{} {
	buffer := make([]byte, 1024)
	para3 := []byte(orderSysID + "\x00")
	function := p.Dll.NewProc("CancelOrder")
	result, _, err := function.Call(uintptr(unsafe.Pointer(&[]byte(instrument)[0])),
		uintptr(unsafe.Pointer(&[]byte(exchangeID)[0])),
		uintptr(unsafe.Pointer(&para3[0])),
		uintptr(unsafe.Pointer(&buffer[0])))
	if err != nil {
		// log.Printf("error:%v result:%v", err, result)
	}

	if result != 0 {
		// return buffer[:result]
		var values map[string]interface{}

		if err = json.Unmarshal(buffer[:result], &values); err != nil {
			logger.Errorf("Fail to Unmarshal:%v, buffer:%v", err, string(buffer[:result]))
			return nil
		}
		logger.Infof("CancelOrder:%v", values)
		return values
	}

	return nil
}

func (p *CTPDll) GetStatus() uintptr {
	function := p.Dll.NewProc("GetStatus")
	result, _, _ := function.Call()
	return uintptr(result)
}
