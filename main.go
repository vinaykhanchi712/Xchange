package main

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vinaykhanchi712/crypto-exchange/client"
	"github.com/vinaykhanchi712/crypto-exchange/mm"
	"github.com/vinaykhanchi712/crypto-exchange/server"
)

var tick = 2 * time.Second

// func marketOrderPlacer(c *client.Client) {
// 	ticker := time.NewTicker(5 * time.Second)
// 	for {

// 		_, err := c.PlaceMarketOrderClient(666, true, 1000)
// 		if err != nil {
// 			fmt.Println("error placement failed")
// 		}
// 		fmt.Printf("MARKET ORDER PLACED !!!!!!!!!!")
// 		<-ticker.C
// 	}
// }

// func marketMaker(c *client.Client) {
// 	ticker := time.NewTicker(tick)

// 	for {

// 		orders100, err := c.GetOrders(100)
// 		if err != nil {
// 			fmt.Println("get order call failed")
// 		}
// 		fmt.Println("----------USER-100-ORDERS--------------------")
// 		fmt.Println(orders100)
// 		fmt.Println("-----------END--------------------")

// 		<-ticker.C

// 		bestAsk, _ := c.GetBestAskClient()
// 		bestBid, _ := c.GetBestBidClient()
// 		spread := math.Abs(bestBid.Price - bestAsk.Price)
// 		if len(orders100.Bids) < 3 {
// 			bidLimit := &server.PlaceOrderRequest{
// 				UserId: 100,
// 				Bid:    true,
// 				Price:  bestBid.Price + 100,
// 				Type:   server.LimitOrder,
// 				Size:   10_000,
// 				Market: server.MarketETH,
// 			}

// 			bidResp, _ := c.PlaceLimitOrder(bidLimit)

// 			fmt.Println("bid order placed with ID", bidResp.OrderId)
// 		}

// 		if len(orders100.Asks) < 3 {
// 			askLimit := &server.PlaceOrderRequest{
// 				UserId: 100,
// 				Bid:    false,
// 				Price:  bestAsk.Price - 100,
// 				Type:   server.LimitOrder,
// 				Size:   10_000,
// 				Market: server.MarketETH,
// 			}

// 			askResp, _ := c.PlaceLimitOrder(askLimit)

// 			fmt.Println("ask order placed with ID", askResp.OrderId)
// 		}
// 		fmt.Println("exchange spread", spread)

// 		fmt.Println("best bid", bestBid)
// 		fmt.Println("best ask", bestAsk)

// 	}

// }

// func getTrades(c *client.Client) {
// 	ticker := time.NewTicker(tick)
// 	for {
// 		<-ticker.C

// 		resp, _ := c.GetTrades("ETH")
// 		fmt.Println("---------Trades------------")
// 		fmt.Println(resp)
// 		fmt.Println("---------Trades End---------")
// 	}
// }

// func seedMarket(c *client.Client) {

// 	user1 := &server.PlaceOrderRequest{
// 		UserId: 100,
// 		Bid:    true,
// 		Price:  9000,
// 		Size:   1000,
// 		Market: server.MarketETH,
// 		Type:   server.LimitOrder,
// 	}
// 	//place one bid order
// 	c.PlaceLimitOrder(user1)

// 	user2 := &server.PlaceOrderRequest{
// 		UserId: 200,
// 		Bid:    false,
// 		Price:  10_000,
// 		Size:   1000,
// 		Market: server.MarketETH,
// 		Type:   server.LimitOrder,
// 	}
// 	//place on ask order
// 	c.PlaceLimitOrder(user2)

// }

func main() {
	go server.StartServer()
	// c := client.NewClient()
	// seedMarket(c)

	// getTrades(c)

	// time.Sleep(3 * time.Second)
	// go makeMarketV2(c)
	// time.Sleep(15 * time.Second)
	// marketOrderPlacerV2(c)
	//----------------Quant trading---------------------
	time.Sleep(3 * time.Second)
	c := client.NewClient()
	cfg := mm.Config{
		OrderSize:      10,
		MinSpread:      20,
		MakeInterval:   1 * time.Second,
		SeddOffSet:     40,
		ExchangeClient: c,
		UserId:         100,
		PriceOffSet:    10,
	}

	maker := mm.NewMarketMaker(cfg)
	maker.Start()
	time.Sleep(5 * time.Second)
	marketOrderPlacerV2(c)
	select {}

}

/*
-----------------------------------Quant Trading series-----------------------
*/
const ETH_PRICE = 1281

func makeMarketV2(c *client.Client) {
	ticker := time.NewTicker(tick)
	for {
		bestAsk, err := c.GetBestAskClient()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("[Main] Error occured while fetching bestAsk from the client")
		}
		bestBid, err := c.GetBestBidClient()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("[Main] Error occured while fetching bestBid from the client")
		}

		if bestAsk.Price == 0 && bestBid.Price == 0 {
			seedMarketv2(c)
			continue
		}

		logrus.WithFields(logrus.Fields{
			"bestAsk": bestAsk.Price,
			"bestBid": bestBid.Price,
		}).Info("Best Ask and Best Bid are")

		<-ticker.C

	}
}

func seedMarketv2(c *client.Client) {
	currentPrice := ETH_PRICE // async call to fetch the price
	priceOffset := 100.0
	//ask order placement
	askRequest := &server.PlaceOrderRequest{
		UserId: 100,
		Type:   server.LimitOrder,
		Bid:    false,
		Price:  float64(currentPrice) + priceOffset,
		Size:   10,
		Market: server.MarketETH,
	}

	_, err := c.PlaceLimitOrder(askRequest)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"Order": askRequest,
		}).Error("[Main] Error occured while placing Limit order")
	}
	if err == nil {
		logrus.Info("[Main] Response of API call to place Limit order successfull")
	}
	//bid order placement
	bidRequest := &server.PlaceOrderRequest{
		UserId: 100,
		Type:   server.LimitOrder,
		Bid:    true,
		Price:  float64(currentPrice) - priceOffset,
		Size:   10,
		Market: server.MarketETH,
	}
	_, er := c.PlaceLimitOrder(bidRequest)
	if er != nil {
		logrus.WithFields(logrus.Fields{
			"error": er,
			"Order": bidRequest,
		}).Error("[Main] Error occured while placing Limit order")
	}
	if er == nil {
		logrus.Info("[Main] Response of API call to place Limit order successfull")
	}

}

func marketOrderPlacerV2(c *client.Client) {
	ticket := time.NewTicker(500 * time.Millisecond)
	for {

		randInt := rand.Intn(10)
		bid := true
		if randInt > 5 {
			bid = false
		}

		resp, buyErr := c.PlaceMarketOrderClient(200, bid, 1)
		if buyErr != nil {
			logrus.WithFields(logrus.Fields{
				"error": buyErr,
			}).Error("[Main] Error occured while placing market order")

		}
		if buyErr == nil {
			logrus.WithFields(logrus.Fields{
				"error": resp,
			}).Info("[Main] Market Order placed successfully")
		}
		<-ticket.C
	}
}
