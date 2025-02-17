package mm

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vinaykhanchi712/crypto-exchange/client"
	"github.com/vinaykhanchi712/crypto-exchange/server"
)

type Config struct {
	UserId         uint64
	OrderSize      float64
	MinSpread      float64
	SeddOffSet     float64
	ExchangeClient *client.Client
	MakeInterval   time.Duration
	PriceOffSet    float64
}

type MarketMaker struct {
	userId         uint64
	orderSize      float64
	minSpread      float64
	seedOffSet     float64
	exchangeClient *client.Client
	makeInterval   time.Duration
	PriceOffSet    float64
}

func NewMarketMaker(config Config) *MarketMaker {
	return &MarketMaker{
		userId:         config.UserId,
		orderSize:      config.OrderSize,
		minSpread:      config.MinSpread,
		seedOffSet:     config.SeddOffSet,
		makeInterval:   config.MakeInterval,
		exchangeClient: config.ExchangeClient,
		PriceOffSet:    config.PriceOffSet,
	}
}

func (mm *MarketMaker) Start() {
	logrus.WithFields(logrus.Fields{
		"id":           mm.userId,
		"orderSize":    mm.orderSize,
		"makeInterval": mm.makeInterval,
		"minSpread":    mm.minSpread,
		"priceOffSet":  mm.PriceOffSet,
	}).Info("Starting Market Maker.....")
	go mm.makerLoop()
}

func (mm *MarketMaker) makerLoop() {
	ticker := time.NewTicker(mm.makeInterval)
	for {
		bestAsk, err := mm.exchangeClient.GetBestAskClient()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("[Maker] Error occured while fetching bestAsk from the client")
			break
		}
		bestBid, err := mm.exchangeClient.GetBestBidClient()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("[Maker] Error occured while fetching bestBid from the client")
			break
		}

		if bestAsk.Price == 0 && bestBid.Price == 0 {
			if err := mm.seedMarket(); err != nil {
				logrus.Error(err)
				break
			}
			continue
		}

		if bestBid.Price == 0 {
			bestBid.Price = bestAsk.Price - mm.PriceOffSet*2
		}

		if bestAsk.Price == 0 {
			bestAsk.Price = bestBid.Price - mm.PriceOffSet*2
		}
		// logrus.WithFields(logrus.Fields{
		// 	"bestAsk": bestAsk.Price,
		// 	"bestBid": bestBid.Price,
		// }).Info("Best Ask and Best Bid are")

		spread := bestAsk.Price - bestBid.Price

		// logrus.WithFields(logrus.Fields{
		// 	"CurrentSpread": spread,
		// }).Info("[maker]Spread is ")

		if spread <= mm.minSpread {
			continue
		}

		_, er := mm.PlaceOrder(true, bestBid.Price+mm.PriceOffSet)
		if er != nil {
			logrus.Error(er)
			break
		}

		_, errr := mm.PlaceOrder(false, bestAsk.Price-mm.PriceOffSet)
		if errr != nil {
			logrus.Error(errr)
			break
		}

		<-ticker.C
	}
}
func (mm *MarketMaker) seedMarket() error {
	currentPrice := simulateFetchCurrentETHPrice() // async call to fetch the price

	//ask order placement
	_, err := mm.PlaceOrder(false, float64(currentPrice)+mm.seedOffSet)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":  err,
			"price":  float64(currentPrice) - mm.seedOffSet,
			"userId": mm.userId,
			"size":   mm.orderSize,
		}).Error("[Main] Error occured while placing Limit order")
		return err
	}

	//bid order placement
	_, er := mm.PlaceOrder(true, float64(currentPrice)-mm.seedOffSet)
	if er != nil {
		logrus.WithFields(logrus.Fields{
			"error":  er,
			"price":  float64(currentPrice) - mm.seedOffSet,
			"userId": mm.userId,
			"size":   mm.orderSize,
		}).Error("[Main] Error occured while placing Limit order")
		return err
	}

	return nil
}

// this will simulate a call to other exchange
func simulateFetchCurrentETHPrice() float64 {
	return 1000.0
}

func (mm *MarketMaker) PlaceOrder(bid bool, price float64) (*server.PlaceOrderResponse, error) {
	orderRequest := &server.PlaceOrderRequest{
		UserId: int64(mm.userId),
		Type:   server.LimitOrder,
		Bid:    bid,
		Price:  price,
		Size:   mm.orderSize,
		Market: server.MarketETH,
	}
	return mm.exchangeClient.PlaceLimitOrder(orderRequest)

}
