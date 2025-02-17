package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/vinaykhanchi712/crypto-exchange/server"
)

const Endpoint string = "http://localhost:3000"

type Client struct {
	*http.Client
}

func NewClient() *Client {
	return &Client{
		Client: http.DefaultClient,
	}
}

func (c *Client) CancelOrderClient(orderId int64) error {
	e := fmt.Sprintf("%s/order/%s/%d", Endpoint, server.MarketETH, orderId)
	request, err := http.NewRequest(http.MethodDelete, e, nil)
	if err != nil {
		return err
	}

	_, err = c.Do(request)
	if err != nil {
		return nil
	}
	return nil
}

func (c *Client) PlaceLimitOrder(pr *server.PlaceOrderRequest) (*server.PlaceOrderResponse, error) {
	params := &server.PlaceOrderRequest{
		UserId: pr.UserId,
		Type:   server.LimitOrder,
		Bid:    pr.Bid,
		Size:   pr.Size,
		Price:  pr.Price,
		Market: server.MarketETH,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	e := Endpoint + "/order"

	req, err := http.NewRequest(http.MethodPost, e, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)

	if err != nil {
		return nil, err
	}

	r := &server.PlaceOrderResponse{}

	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) PlaceMarketOrderClient(userId int64, bid bool, size float64) (*server.PlaceOrderResponse, error) {
	params := &server.PlaceOrderRequest{
		UserId: userId,
		Type:   server.MarketOrder,
		Bid:    bid,
		Size:   size,
		Price:  0.0,
		Market: server.MarketETH,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	e := Endpoint + "/order"

	req, err := http.NewRequest(http.MethodPost, e, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)

	if err != nil {
		return nil, err
	}
	fmt.Println(resp)
	return &server.PlaceOrderResponse{
		OrderId: 1,
	}, nil
}

func (c *Client) GetBestAskClient() (server.BestBidAskResponse, error) {
	e := Endpoint + "/book/asks"

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("[Client] Error occured while making request for fetching Asks from orderbook")
	}

	resp, err := c.Do(req)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("[Client] Error occured while fetching Asks from orderbook")
	}

	r := server.BestBidAskResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, err
	}
	return r, nil

}
func (c *Client) GetBestBidClient() (server.BestBidAskResponse, error) {
	e := Endpoint + "/book/bids"

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("[Client] Error occured while making request for fetching Bids from orderbook")
	}

	resp, err := c.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("[Client] Error occured while fetching Asks from orderbook")
	}
	r := server.BestBidAskResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, err
	}
	return r, nil

}

func (c *Client) GetOrders(userId int64) (server.HandleGetOrderResponse, error) {
	e := fmt.Sprintf("%s/order/%d", Endpoint, userId)

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		return server.HandleGetOrderResponse{}, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return server.HandleGetOrderResponse{}, err
	}
	r := server.HandleGetOrderResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, nil
	}
	return r, nil
}

func (c *Client) GetTrades(market string) ([]server.TradesResponse, error) {

	e := fmt.Sprintf("%s/trades/%s", Endpoint, market)

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		fmt.Println("unable to fetch get trades")
		return []server.TradesResponse{}, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return []server.TradesResponse{}, err
	}
	r := &[]server.TradesResponse{}
	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return *r, nil
	}
	return *r, nil
}
