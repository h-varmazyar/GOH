package models

type TSETradeHistory struct {
	MarketID   uint64 `json:"market_id"`
	InsCode    string `json:"ins_code"`
	TradeDate  string `json:"trade_date"`
	TradeIndex uint32 `json:"trade_index"`
	Canceled   bool   `json:"canceled"`
	TradeTime  int64  `json:"trade_time"`
	Price      int64  `json:"price"`
	Quantity   int64  `json:"quantity"`
	Version    uint64 `json:"version"`
}

func (TSETradeHistory) TableName() string { return "tse_trade_history" }
