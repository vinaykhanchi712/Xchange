package orderbook

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Match struct {
	Ask        *Order
	Bid        *Order
	SizeFilled float64
	Price      float64
}

type Order struct {
	ID        int64
	UserId    int64
	Size      float64
	Bid       bool
	Limit     *Limit
	Timestamp int64
}

type Orders []*Order

func (o Orders) Len() int           { return len(o) }
func (o Orders) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o Orders) Less(a, b int) bool { return o[a].Timestamp < o[b].Timestamp }

func NewOrder(bid bool, size float64, userId int64) *Order {
	return &Order{
		ID:        int64(rand.Intn(10_000_000)),
		UserId:    userId,
		Size:      size,
		Bid:       bid,
		Timestamp: time.Now().UnixNano(),
	}
}

func (o *Order) IsFilled() bool {
	return o.Size == 0
}

func (o *Order) String() string {
	return fmt.Sprintf("[ bid: %v , size: %.2f]", o.Bid, o.Size)
}

type Limit struct {
	Price       float64
	Orders      Orders
	TotalVolume float64
}

type Limits []*Limit

type ByBestAsk struct {
	Limits
}

func (a ByBestAsk) Len() int { return len(a.Limits) }
func (a ByBestAsk) Swap(i, j int) {
	c := a.Limits[i]
	a.Limits[i] = a.Limits[j]
	a.Limits[j] = c
}
func (a ByBestAsk) Less(i, j int) bool {
	return a.Limits[i].Price < a.Limits[j].Price
}

type ByBestBid struct {
	Limits
}

func (a ByBestBid) Len() int { return len(a.Limits) }
func (a ByBestBid) Swap(i, j int) {
	c := a.Limits[i]
	a.Limits[i] = a.Limits[j]
	a.Limits[j] = c
}

func (a ByBestBid) Less(i, j int) bool {
	return a.Limits[i].Price > a.Limits[j].Price
}

func NewLimit(price float64) *Limit {
	return &Limit{
		Price:  price,
		Orders: []*Order{},
	}
}

func (l *Limit) AddOrder(o *Order) {
	o.Limit = l
	l.Orders = append(l.Orders, o)
	l.TotalVolume += o.Size
}

func (l *Limit) DeleteOrder(o *Order) {

	for i := 0; i < len(l.Orders); i++ {
		if l.Orders[i] == o {
			l.Orders[i] = l.Orders[len(l.Orders)-1]
			l.Orders = l.Orders[:len(l.Orders)-1]
		}
	}
	o.Limit = nil
	l.TotalVolume -= o.Size
}

func (l *Limit) Fill(o *Order) []Match {

	var matches []Match
	var orderToDelete []*Order

	sort.Sort(l.Orders)
	for _, order := range l.Orders {

		if o.IsFilled() {
			break
		}

		orderMatch := l.FillOrder(order, o)
		matches = append(matches, orderMatch)

		l.TotalVolume -= orderMatch.SizeFilled

		if order.IsFilled() {
			orderToDelete = append(orderToDelete, order)
		}

	}

	for _, order := range orderToDelete {
		l.DeleteOrder(order)
	}

	return matches
}

func (l *Limit) FillOrder(a, b *Order) Match {
	var ask, bid *Order
	var sizeFill float64

	//clairfying which is bid and which is ask
	if a.Bid {
		ask = b
		bid = a
	} else {
		ask = a
		bid = b
	}

	if b.Size <= a.Size {
		a.Size -= b.Size
		sizeFill = b.Size
		b.Size = 0.0
	} else {
		b.Size -= a.Size
		sizeFill = a.Size
		a.Size = 0.0
	}

	return Match{
		Ask:        ask,
		Bid:        bid,
		SizeFilled: sizeFill,
		Price:      l.Price,
	}

}

type Trade struct {
	Price     float64
	Bid       bool
	Size      float64
	Timestamp int64
}
type Orderbook struct {
	Asks      Limits
	Bids      Limits
	Trades    []*Trade
	mu        sync.RWMutex
	AskLimits map[float64]*Limit
	BidLimits map[float64]*Limit
	OrderMap  map[int64]*Order
}

func NewOrderBook() *Orderbook {
	return &Orderbook{
		Asks:      []*Limit{},
		Bids:      []*Limit{},
		Trades:    []*Trade{},
		AskLimits: make(map[float64]*Limit),
		BidLimits: make(map[float64]*Limit),
		OrderMap:  make(map[int64]*Order),
	}
}

func (ob *Orderbook) String() string {
	return fmt.Sprintf("bids:[%v], asks:[%v]", ob.Bids, ob.Asks)
}

func (ob *Orderbook) AskTotalVolume() float64 {
	var totalVolume float64

	for i := 0; i < len(ob.Asks); i++ {
		totalVolume += ob.Asks[i].TotalVolume
	}
	return totalVolume
}

func (ob *Orderbook) BidsTotalVolume() float64 {
	var totalVolume float64

	for i := 0; i < len(ob.Bids); i++ {
		totalVolume += ob.Bids[i].TotalVolume
	}
	return totalVolume
}

func (ob *Orderbook) ClearLimit(bid bool, l *Limit) {

	if bid {
		delete(ob.BidLimits, l.Price)
		for i := 0; i < len(ob.Bids); i++ {
			if ob.Bids[i] == l {
				ob.Bids[i] = ob.Bids[len(ob.Bids)-1]
				ob.Bids = ob.Bids[:len(ob.Bids)-1]
			}
		}
	} else {
		delete(ob.AskLimits, l.Price)
		for i := 0; i < len(ob.Asks); i++ {
			if ob.Asks[i] == l {
				ob.Asks[i] = ob.Asks[len(ob.Asks)-1]
				ob.Asks = ob.Asks[:len(ob.Asks)-1]
			}
		}
	}

}

// place market order
func (ob *Orderbook) PlaceMarketOrder(o *Order) []Match {

	matches := []Match{}
	var limitsToBeCleared Limits

	if o.Bid {
		if o.Size > ob.AskTotalVolume() {
			panic(fmt.Errorf("insufficient volume. present volume:[%.2f] and order size : [%.2f]", ob.AskTotalVolume(), o.Size))
		}

		for _, limit := range ob.GetBestAsks() {

			if o.IsFilled() {
				break
			}

			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)

			if len(limit.Orders) == 0 {
				limitsToBeCleared = append(limitsToBeCleared, limit)
			}

		}
		//delete clear limits.
		for i := 0; i < len(limitsToBeCleared); i++ {
			ob.ClearLimit(false, limitsToBeCleared[i])
		}

	} else {
		if o.Size > ob.BidsTotalVolume() {
			panic(fmt.Errorf("insufficient volume. present volume:[%.2f] and order size : [%.2f]", ob.BidsTotalVolume(), o.Size))
		}

		for _, limit := range ob.GetBestBids() {

			if o.IsFilled() {
				break
			}

			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)

			if len(limit.Orders) == 0 {
				limitsToBeCleared = append(limitsToBeCleared, limit)
			}
		}
		//delete clear limits.
		for i := 0; i < len(limitsToBeCleared); i++ {
			ob.ClearLimit(true, limitsToBeCleared[i])
		}
	}
	for _, m := range matches {
		trade := &Trade{
			Price:     m.Price,
			Size:      m.SizeFilled,
			Timestamp: time.Now().UnixNano(),
			Bid:       o.Bid,
		}
		ob.Trades = append(ob.Trades, trade)
	}

	logrus.WithFields(logrus.Fields{
		"currentPrice": ob.Trades[len(ob.Trades)-1].Price,
	}).Info("[Orderbook]")

	return matches
}

// place limit order
func (ob *Orderbook) PlaceLimitOrder(price float64, o *Order) {

	ob.mu.Lock()
	defer ob.mu.Unlock()
	var l *Limit

	if o.Bid {
		l = ob.BidLimits[price]
	} else {
		l = ob.AskLimits[price]
	}

	if l == nil {
		l = NewLimit(price)

		if o.Bid {
			ob.Bids = append(ob.Bids, l)
			ob.BidLimits[price] = l
		} else {
			ob.Asks = append(ob.Asks, l)
			ob.AskLimits[price] = l
		}
	}

	logrus.WithFields(logrus.Fields{
		"Price":     l.Price,
		"Type":      o.Type(),
		"Size":      o.Size,
		"Timestamp": o.Timestamp,
		"UserId":    o.UserId,
	}).Info("New Limit Order")

	l.AddOrder(o)
	ob.OrderMap[o.ID] = o
}

// cancel order
func (ob *Orderbook) CancelOrder(o *Order) {
	l := o.Limit
	delete(ob.OrderMap, o.ID)
	l.DeleteOrder(o)

	//clearing limit as no orders left in the limit
	if len(l.Orders) == 0 {
		ob.ClearLimit(o.Bid, l)
	}
}

func (ob *Orderbook) CancelOrderById(id int64) {
	o := ob.OrderMap[id]
	l := o.Limit
	delete(ob.OrderMap, id)
	l.DeleteOrder(o)

	//clearing limit as no orders left in the limit
	if len(l.Orders) == 0 {
		ob.ClearLimit(o.Bid, l)
	}
}

func (ob *Orderbook) GetBestBids() Limits {
	if ob.Bids == nil {
		return Limits{}
	}
	sort.Sort(ByBestBid{ob.Bids})

	return ob.Bids
}

func (ob *Orderbook) GetBestAsks() Limits {
	if ob.Asks == nil {
		return Limits{}
	}
	sort.Sort(ByBestAsk{ob.Asks})

	return ob.Asks
}

func (o *Order) Type() string {

	str := "ASK"
	if o.Bid {
		str = "BID"
	}
	return str
}
