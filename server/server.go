package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/vinaykhanchi712/crypto-exchange/orderbook"
)

const (
	MarketOrder      OrderType = "MARKET"
	LimitOrder       OrderType = "LIMIT"
	MarketETH        Market    = "ETH"
	privateKeyString string    = "59a45aa6f473957623db1123f322e2c126ff7183fdcbf940b16569952ebb6695"
)

type (
	Market    string
	OrderType string

	ConnectionManager struct {
		Conn map[*websocket.Conn]bool
		mu   sync.RWMutex
	}

	Ticker struct {
		Price       float64
		Spread      float64
		TotalVolume float64
	}

	Exchange struct {
		Client      *ethclient.Client
		ConnManager *ConnectionManager
		mu          sync.RWMutex
		Users       map[int64]*User
		Orders      map[int64][]*orderbook.Order //userId->order[] map
		PrivateKey  *ecdsa.PrivateKey
		orderbooks  map[Market]*orderbook.Orderbook
	}

	PlaceOrderRequest struct {
		UserId int64
		Type   OrderType
		Bid    bool
		Size   float64
		Price  float64
		Market Market
	}

	Order struct {
		UserId    int64
		ID        int64
		Price     float64
		Size      float64
		Bid       bool
		Timestamp int64
	}

	MatchedOrder struct {
		UserId int64
		Price  float64
		Size   float64
		ID     int64
	}

	OrderbookData struct {
		TotalAskVolume float64
		TotalBidVolume float64
		Asks           []*Order
		Bids           []*Order
	}
)

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		Conn: make(map[*websocket.Conn]bool),
	}
}

func (cm *ConnectionManager) AddConnection(conn *websocket.Conn) {
	cm.mu.Lock()
	cm.Conn[conn] = true
	cm.mu.Unlock()
}

func (cm *ConnectionManager) RemoveConnection(conn *websocket.Conn) {
	cm.mu.Lock()
	delete(cm.Conn, conn)
	cm.mu.Unlock()
}

func (cm *ConnectionManager) Broadcast(msg []byte) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for conn := range cm.Conn {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("Error while broadcasting:", err)
			cm.RemoveConnection(conn)
			return
		}
	}
}

func StartServer() {
	//new instance
	e := echo.New()

	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("http://127.0.0.1:7545")
	if err != nil {
		log.Fatal(err)
	}
	//new exchange instance
	ex, err := NewExchange(privateKeyString, client)
	if err != nil {
		fmt.Println(err)
	}
	//define new users
	ex.AddUserInExchange("24e4f2e2bcd525e899b9d08a0282e875848f414b5267fd3d08e91641053e235b", 100)
	ex.AddUserInExchange("141ab6c2897f70fab6c4743b636b85236bd5672176b1d32e2a5f128c9f2c9b33", 200)
	ex.AddUserInExchange("4fb086cda5a684e916c1899f6e3498af4325c5d948eab8853402b894b5af0d23", 666)

	// Routes
	e.GET("/trades/:market", ex.handleGetTrades)
	e.POST("/order", ex.PlaceOrderHandler)
	e.GET("/order/:userId", ex.handleGetOrders)
	e.GET("/book/:market", ex.handleGetBook)
	e.DELETE("/order/:market/:id", ex.CancelOrderHandler)
	e.GET("/book/bids", ex.handleGetBestBid)
	e.GET("/book/asks", ex.handleGetBestAsk)

	e.GET("/ws", ex.websocketHandler)

	// Start server

	if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}
}

type User struct {
	ID         int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(pk string, id int64) *User {
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		panic(err)
	}
	return &User{
		ID:         id,
		PrivateKey: privateKey,
	}
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (ex *Exchange) websocketHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("Error while upgrading to websocket: ", err)
		return err
	}
	defer conn.Close()
	ex.ConnManager.AddConnection(conn)
	logrus.Info("New connection added")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message: ", err)
			return err
		}

		logrus.Info("Received message: ", string(msg))

	}
	return nil
}

func NewExchange(privateKey string, client *ethclient.Client) (*Exchange, error) {
	ob := make(map[Market]*orderbook.Orderbook)
	ob[MarketETH] = orderbook.NewOrderBook()

	manager := NewConnectionManager()

	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &Exchange{
		Client:      client,
		Users:       make(map[int64]*User),
		Orders:      make(map[int64][]*orderbook.Order),
		PrivateKey:  key,
		orderbooks:  ob,
		ConnManager: manager,
	}, nil
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderbookData{
		TotalAskVolume: 0.0,
		TotalBidVolume: 0.0,
		Asks:           []*Order{},
		Bids:           []*Order{},
	}

	orderbookData.TotalAskVolume = ob.AskTotalVolume()
	orderbookData.TotalBidVolume = ob.BidsTotalVolume()
	for _, limit := range ob.Asks {
		for _, order := range limit.Orders {
			orderData := Order{
				UserId:    order.UserId,
				ID:        order.ID,
				Price:     order.Limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Asks = append(orderbookData.Asks, &orderData)

		}
	}
	for _, limit := range ob.Bids {
		for _, order := range limit.Orders {
			orderData := Order{
				UserId:    order.UserId,
				ID:        order.ID,
				Price:     order.Limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Bids = append(orderbookData.Bids, &orderData)

		}
	}

	return c.JSON(http.StatusOK, orderbookData)

}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrder) {
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrder, len(matches))

	isBid := false
	if order.Bid {
		isBid = true
	}
	totalSizeFilled := 0.0
	sumPrice := 0.0
	for i := 0; i < len(matchedOrders); i++ {

		userID := matches[i].Bid.UserId
		id := matches[i].Bid.ID
		if isBid {
			id = matches[i].Ask.ID
			userID = matches[i].Ask.UserId
		}

		matchedOrders[i] = &MatchedOrder{
			UserId: userID,
			ID:     id,
			Size:   matches[i].SizeFilled,
			Price:  matches[i].Price,
		}
		totalSizeFilled += matches[i].SizeFilled
		sumPrice += matches[i].Price

	}
	avgPrice := sumPrice / float64(len(matches))

	logrus.WithFields(logrus.Fields{
		"size":      totalSizeFilled,
		"avg price": avgPrice,
		"type":      order.Type(),
		"userId":    order.UserId,
	}).Info("Filled Market Order")
	//broadcast to all connected clients

	go broadcastToAllConnections(avgPrice, ob, ex)

	newOrderMap := make(map[int64][]*orderbook.Order)

	ex.mu.RLock()
	for userId, orderBookOrders := range ex.Orders {
		for i := 0; i < len(orderBookOrders); i++ {
			if !orderBookOrders[i].IsFilled() {
				newOrderMap[userId] = append(newOrderMap[userId], orderBookOrders[i])
			}
		}
	}
	ex.mu.RUnlock()
	ex.mu.Lock()
	ex.Orders = newOrderMap
	ex.mu.Unlock()
	return matches, matchedOrders
}

func broadcastToAllConnections(avgPrice float64, ob *orderbook.Orderbook, ex *Exchange) {
	tickerExport := Ticker{
		Price:       avgPrice,
		TotalVolume: ob.AskTotalVolume() + ob.BidsTotalVolume(),
		Spread:      ob.GetBestAsks()[0].Price - ob.GetBestBids()[0].Price,
	}

	msg, marshallError := json.Marshal(tickerExport)
	if marshallError != nil {
		logrus.Error("Error while marshalling ticker data:", marshallError)
	}
	ex.ConnManager.Broadcast(msg)
}

type PlaceOrderResponse struct {
	OrderId int64
}

type BestBidAskResponse struct {
	Price float64
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[Market(market)]
	ob.PlaceLimitOrder(price, order)
	ex.mu.Lock()
	ex.Orders[order.UserId] = append(ex.Orders[order.UserId], order)
	ex.mu.Unlock()
	log.Printf("new Limit order=> type[%t] price[%.2f] | size [%.2f]", order.Bid, order.Limit.Price, order.Size)
	return nil
}

func (ex *Exchange) PlaceOrderHandler(c echo.Context) error {

	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}
	market := Market(placeOrderData.Market)

	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserId)
	if placeOrderData.Type == MarketOrder {
		matches, _ := ex.handlePlaceMarketOrder(market, order)

		if err := ex.handleMatches(matches); err != nil {
			return err
		}

		return c.JSON(200, map[string]any{"msg": "Market Order placed with length of matches", "matches": matches})

	} else if placeOrderData.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
			return err
		}

		resp := &PlaceOrderResponse{
			OrderId: order.ID,
		}
		return c.JSON(200, resp)
	}

	return c.JSON(200, map[string]any{"msg": "Order placed"})

}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserId]
		if !ok {
			return fmt.Errorf("error while fetchig user from match")
		}

		toUser, ok := ex.Users[match.Bid.UserId]
		if !ok {
			return fmt.Errorf("error while fetchig user from match")
		}

		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)
		amount := big.NewInt((int64(match.SizeFilled)))

		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
	}
	return nil
}

func (ex *Exchange) CancelOrderHandler(c echo.Context) error {
	IDstr := c.Param("id")
	ID, _ := strconv.Atoi(IDstr)
	market := c.Param("market")
	ob := ex.orderbooks[Market(market)]
	ob.CancelOrderById(int64(ID))
	return c.JSON(200, map[string]any{"Order deleted successfully of orderId": ID})
}

func (ex *Exchange) handleGetBestAsk(c echo.Context) error {

	ob := ex.orderbooks[MarketETH]
	if ob == nil {
		return c.JSON(200, BestBidAskResponse{
			Price: 0,
		})
	}
	resp := ob.GetBestAsks()
	if len(resp) == 0 {
		return c.JSON(200, BestBidAskResponse{
			Price: 0,
		})
	}
	bestAsk := BestBidAskResponse{
		Price: resp[0].Price,
	}
	return c.JSON(200, bestAsk)
}

func (ex *Exchange) handleGetBestBid(c echo.Context) error {
	ob := ex.orderbooks[MarketETH]

	if ob == nil {
		return c.JSON(200, BestBidAskResponse{
			Price: 0,
		})
	}
	resp := ob.GetBestBids()
	if len(resp) == 0 {
		return c.JSON(200, BestBidAskResponse{
			Price: 0,
		})
	}
	bestBid := BestBidAskResponse{
		Price: resp[0].Price,
	}
	return c.JSON(200, bestBid)
}

func transferETH(client *ethclient.Client, from *ecdsa.PrivateKey, to common.Address, amount *big.Int) error {
	ctx := context.Background()
	publicKey := from.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

	chainID := big.NewInt(1337)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), from)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (ex *Exchange) AddUserInExchange(pk string, id int64) {
	user := NewUser(pk, id)
	ex.Users[user.ID] = user
	logrus.WithFields(logrus.Fields{
		"ID": id,
	}).Info("New user added")
}

type HandleGetOrderResponse struct {
	Asks []Order
	Bids []Order
}

func (ex *Exchange) handleGetOrders(c echo.Context) error {
	userIdStr := c.Param("userId")
	userId, _ := strconv.Atoi(userIdStr)

	ex.mu.RLock()

	orders := ex.Orders[int64(userId)]
	orderResp := HandleGetOrderResponse{
		Asks: []Order{},
		Bids: []Order{},
	}

	for _, o := range orders {
		// to avoid NPE when Limit got empty ! TO DO := change later
		if o.Limit == nil {
			continue
		}
		orderStruct := Order{
			UserId:    o.UserId,
			ID:        o.ID,
			Price:     o.Limit.Price,
			Size:      o.Size,
			Bid:       o.Bid,
			Timestamp: o.Timestamp,
		}
		if orderStruct.Bid {
			orderResp.Bids = append(orderResp.Bids, orderStruct)
		} else {
			orderResp.Asks = append(orderResp.Asks, orderStruct)
		}
	}
	ex.mu.RUnlock()
	return c.JSON(200, orderResp)
}

type TradesResponse struct {
	Price     float64
	Size      float64
	Bid       bool
	Timestamp int64
}

func (ex *Exchange) handleGetTrades(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(500, map[any]any{"error": "error in fetching market"})
	}

	resp := []TradesResponse{}

	for _, o := range ob.Trades {
		resp = append(resp, TradesResponse{
			Price:     o.Price,
			Size:      o.Size,
			Timestamp: o.Timestamp,
			Bid:       o.Bid,
		})
	}

	return c.JSON(200, resp)
}
