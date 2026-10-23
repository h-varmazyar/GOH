package consts

type MarketStatus string

const (
	MarketStatusOpen                 MarketStatus = "OPEN"
	MarketStatusClose                MarketStatus = "CLOSE"
	MarketStatusDeactivateByExchange MarketStatus = "DEACTIVATE_BY_EXCHANGE"
	MarketStatusDeactivateBySystem   MarketStatus = "DEACTIVATE_BY_SYSTEM"
)

type TSEMarketDataMode string

const (
	TSEMarketDataToday TSEMarketDataMode = "today"
	TSEMarketDataDate  TSEMarketDataMode = "date"
	TSEMarketDataRange TSEMarketDataMode = "range"
	TSEMarketDataAll   TSEMarketDataMode = "all"
)
