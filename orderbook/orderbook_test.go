package orderbook

import (
	"fmt"
	"reflect"
	"testing"
)

func assert(t *testing.T, a, b any) {

	if !reflect.DeepEqual(a, b) {
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestLimit(t *testing.T) {
	l := NewLimit(10_000)
	buyOrderA := NewOrder(true, 5, 1)
	buyOrderB := NewOrder(true, 8, 2)
	sellOrderC := NewOrder(false, 10, 3)

	l.AddOrder(buyOrderA)
	l.AddOrder(buyOrderB)
	l.AddOrder(sellOrderC)

	fmt.Printf("%-v", l)

}

func TestPlaceLimitOrder(t *testing.T) {
	ob := NewOrderBook()

	buyOrderA := NewOrder(true, 10, 1)
	buyOrderB := NewOrder(true, 5, 2)
	sellOrderC := NewOrder(false, 8, 3)

	ob.PlaceLimitOrder(10_000, buyOrderA)
	ob.PlaceLimitOrder(10_000, buyOrderB)
	ob.PlaceLimitOrder(10_000, sellOrderC)

	assert(t, len(ob.Bids), 1)
	assert(t, len(ob.Asks), 1)

}

func TestPlaceMarketOrder(t *testing.T) {
	ob := NewOrderBook()

	buyOrderA := NewOrder(true, 7, 1)
	buyOrderB := NewOrder(true, 5, 2)
	sellOrderC := NewOrder(false, 8, 3)

	ob.PlaceLimitOrder(10_000, buyOrderA)
	ob.PlaceLimitOrder(10_000, buyOrderB)
	ob.PlaceMarketOrder(sellOrderC)

	fmt.Printf("[ob:%+v]", len(ob.Bids))

	assert(t, ob.BidsTotalVolume(), float64(4))

}
