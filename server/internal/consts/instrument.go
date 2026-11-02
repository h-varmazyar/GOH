package consts

// TSEMarketType is the exchange board an instrument is traded on.
type TSEMarketType string

const (
	TSEMarketTypeBourse       TSEMarketType = "BOURSE"
	TSEMarketTypeIFBFirst     TSEMarketType = "IFB_FIRST"
	TSEMarketTypeIFBSecond    TSEMarketType = "IFB_SECOND"
	TSEMarketTypeIFBBase      TSEMarketType = "IFB_BASE"
	TSEMarketTypeIFBNoAfarin  TSEMarketType = "IFB_NOAFARIN"
	TSEMarketTypeIFBAgreement TSEMarketType = "IFB_AGREEMENT"
	// TSEMarketTypeIFB is an IFB instrument whose board is not listed by IFB,
	// such as a delisted symbol, a fund or a symbol before its first trade.
	TSEMarketTypeIFB TSEMarketType = "IFB"
)

var tseMarketTypePersianLabel = map[TSEMarketType]string{
	TSEMarketTypeBourse:       "بورس",
	TSEMarketTypeIFBFirst:     "بازار اول فرابورس",
	TSEMarketTypeIFBSecond:    "بازار دوم فرابورس",
	TSEMarketTypeIFBBase:      "بازار پایه فرابورس",
	TSEMarketTypeIFBNoAfarin:  "بازار نوآفرین فرابورس",
	TSEMarketTypeIFBAgreement: "بازار توافقی فرابورس",
	TSEMarketTypeIFB:          "فرابورس",
}

func (t TSEMarketType) PersianLabel() string {
	if label, exists := tseMarketTypePersianLabel[t]; exists {
		return label
	}
	return ""
}

// TSEInstrumentKind is what an instrument represents.
type TSEInstrumentKind string

const (
	TSEInstrumentKindStock TSEInstrumentKind = "STOCK"
	TSEInstrumentKindFund  TSEInstrumentKind = "FUND"
	TSEInstrumentKindRight TSEInstrumentKind = "RIGHT"
)

var tseInstrumentKindPersianLabel = map[TSEInstrumentKind]string{
	TSEInstrumentKindStock: "سهام",
	TSEInstrumentKindFund:  "صندوق قابل معامله",
	TSEInstrumentKindRight: "حق تقدم",
}

func (k TSEInstrumentKind) PersianLabel() string {
	if label, exists := tseInstrumentKindPersianLabel[k]; exists {
		return label
	}
	return ""
}
