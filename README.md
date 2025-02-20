# Stock Xchange

A fully functional stock exchange clone where users can:
- Place market and limit orders
- Fetch real-time order books
- Cancel orders
- Implement their own market-making strategies

## Features
✅ Real-time order book updates  
✅ Market and limit order support  
✅ Order cancellation  
✅ API-driven architecture  
✅ Subscription/websocket mechanism for real-time notifications 
✅ Custom market-making strategies  

## Tech Stack
- **Backend:** Golang
- **Frontend:** Next.js (optional, for UI)

## Installation & Setup

### Prerequisites
- Golang installed
- Node.js (if using the frontend)

### Clone the Repository
```sh
git clone https://github.com/vinaykhanchi712/Xchange.git
cd Xchange
```

### Backend Setup
```sh
go mod tidy
go run main.go
```

### Frontend Setup (Optional)
```sh
cd frontend
npm install
npm start
```

## API Endpoints

### 1. Get Order Book
```http
GET /book/ETH
```
_Response:_
```json
{
    "TotalAskVolume": 0.0,
	"TotalBidVolume": 0.0,
	"Asks":
    { 
        [
            "UserId" : 123,
            "ID"        : 745722,
            "Price"     : 1245.2,
            "Size"     :150
            "Bid"       :false
            "Timestamp" : 12580000,
        ] 
    }         
	"Bids":
     { 
        [
            "UserId" : 123,
            "ID"        : 745722,
            "Price"     : 1245.2,
            "Size"     :150
            "Bid"       :true
            "Timestamp" : 12580000,
        ] 
    }
}
```

### 2. Place Order
```http
POST /order
```
_Body:_
```json
{
    "UserId": 100,
	"Type":   "LIMIT/MARKET",
	"Bid" :   true,
	"Size":   150,
	"Price":  123.43,
	"Market": "ETH"
}
```

### 3. Cancel Order
```http
POST /order/ETH/{orderId}
```
### 4. Get Best Ask
```http
GET /book/asks
```
_Response:_
```json
{"Price": 123.2}
```

### 5. Get Best Bid
```http
GET /book/bids
```
_Response:_
```json
{"Price": 123.2}
```

### 6. Get Trades
```http
GET /trades/ETH
```
_Response:_
```json
[
    { 
        "Price":  123.2,
        "Size"  :150,
        "Bid"  : "ask/bid",
        "Timestamp": 12350000
    },
    .....
]
```


### 6. Establish Websocket Connection
```http
GET ws:{FQDN}/ws
```
_Response:_
```json
{ 
    "Price":1000,
    "Spread":10,
    "TotalVolume":74
}
```

## Market-Making Strategies
Users can build their own market-making strategies by integrating custom trading algorithms and subscribing to order book updates.

## Contribution
We welcome contributions! Please follow these steps:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Open a pull request

## License
MIT License

## Contact
For any issues, feel free to open an issue or reach out at [your email].

