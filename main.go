package main

import (
	"context"
	tradingServer "example/proto"
	"fmt"
	"io"
	mrand "math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Item struct {
	id        uint32
	name      string
	price     uint32
	available uint32
}

type TradingServer struct {
	items []*Item
	tradingServer.UnimplementedTradingServer
}

func CreateTradingServer() *TradingServer {
	items := []*Item{
		{
			id:        1,
			name:      "Lord Of Hunger",
			price:     1000,
			available: 200,
		},
		{
			id:        2,
			name:      "Lord Of Evil",
			price:     20000,
			available: 500,
		},
	}

	return &TradingServer{
		items: items,
	}
}

func (s *TradingServer) ListItems(request *tradingServer.ListItemsRequest, serverChannel tradingServer.Trading_ListItemsServer) error {
	for _, item := range s.items {
		serverChannel.Send(&tradingServer.Goods{
			Id:        &item.id,
			Name:      &item.name,
			Price:     &item.price,
			Available: &item.available,
		})
	}

	return nil
}

func (s *TradingServer) Buy(c context.Context, request *tradingServer.BuyRequest) (*tradingServer.BuyResponse, error) {
	isSuccess := false

	for _, item := range s.items {
		if item.id == request.GetId() {
			fmt.Printf("Total Stock %s Is %d\n", item.name, item.available)

			if item.available >= request.GetQty() {
				item.available = item.available - request.GetQty()
				isSuccess = true

				fmt.Printf("Someone purchasing %s (%d)\n", item.name, request.GetQty())

				return &tradingServer.BuyResponse{
					Success: &isSuccess,
				}, nil
			} else {
				fmt.Printf("Can't Purchasing Goods %s (%d)\n", item.name, request.GetQty())

				return &tradingServer.BuyResponse{
					Success: &isSuccess,
				}, nil
			}
		}
	}

	fmt.Println("Good not found")
	return &tradingServer.BuyResponse{
		Success: &isSuccess,
	}, nil
}

func runServer() {
	server := CreateTradingServer()

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", 8888))
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	tradingServer.RegisterTradingServer(grpcServer, server)

	if err := grpcServer.Serve(listen); err != nil {
		panic(err)
	}
}

func doClient() {
	// Sleep first waiting server to run first
	time.Sleep(10 * time.Second)

	var receivedItems = make([]*tradingServer.Goods, 1)

	conn, err := grpc.Dial("localhost:8888", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := tradingServer.NewTradingClient(conn)
	items, err := client.ListItems(context.Background(), &tradingServer.ListItemsRequest{})
	if err != nil {
		panic(err)
	}

	for {
		good, err := items.Recv()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		fmt.Printf("Good %s total available is %d\n", good.GetName(), good.GetAvailable())
		receivedItems = append(receivedItems, good)
	}

	for _, availableGood := range receivedItems {
		for i := uint32(0); i < availableGood.GetAvailable(); i++ {
			fmt.Printf("Attempt to purchase %s\n", availableGood.GetName())

			// Simulate random qty purchasing
			randQty := uint32(mrand.Intn(100))

			response, err := client.Buy(context.Background(), &tradingServer.BuyRequest{
				Id:  availableGood.Id,
				Qty: &randQty,
			})

			if err != nil {
				panic(err)
			}

			fmt.Printf("Purchase %s is %t\n", availableGood.GetName(), response.GetSuccess())
		}
	}
}

func main() {
	go runServer()
	go doClient()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	<-done
}
