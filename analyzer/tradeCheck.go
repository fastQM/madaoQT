// package main

// import (
// 	"flag"
// 	"log"
// 	"net"

// 	"madaoqt/redis"
// 	tradeCheck "madaoqt/tradeCheck"

// 	"golang.org/x/net/context"
// 	"google.golang.org/grpc"
// )

// type Service struct {
// }

// func (S *Service) GetInfo(ctx context.Context, handle *tradeCheck.TradeHandle) (*tradeCheck.TradeCheckInfo, error) {

// 	log.Printf("Call from Handle:%v", handle)
// 	return &tradeCheck.TradeCheckInfo{Info: "This is the template"}, nil
// }

// // 根据现有走势数组判断买卖点
// func (S *Service) TradeCheck(ctx context.Context, name *tradeCheck.TokenName) (*tradeCheck.TradeExpect, error) {

// 	conn := new(redis.ChartsHistory)
// 	err := conn.LoadCharts(name.Name, 1)

// 	if err == nil {
// 		analyzer := new(tradeCheck.Analyzer)
// 		analyzer.Init(name.Name, conn.Charts, 0)
// 		analyzer.Analyze()
// 	}

// 	return &tradeCheck.TradeExpect{}, nil
// }

// func main() {
// 	log.Printf("Server starting...")

// 	port := flag.String("port", ":5000", "RPC listen port")
// 	lis, err := net.Listen("tcp", *port)
// 	if err != nil {
// 		log.Fatalf("failed to listen: %v", err)
// 	}
// 	s := grpc.NewServer()
// 	tradeCheck.RegisterTradeCheckServiceServer(s, &Service{})
// 	s.Serve(lis)
// }

