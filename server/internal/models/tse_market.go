package models

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
)

// Sources of TSEMarket.IPODate.
const (
	TSEIPODateSourceListing      = "listing"
	TSEIPODateSourcePriceHistory = "price_history"
)

type TSEMarket struct {
	ID       uint    `gorm:"primaryKey"`
	MarketID uint    `gorm:"not null;index"`
	Market   *Market `gorm:"foreignKey:MarketID"`
	InsCode  string  `gorm:"column:ins_code;unique"`
	// Label is the TSETMC symbol.
	Label string

	// InstrumentID is the instrument ISIN that starts with "IR".
	InstrumentID   string                   `gorm:"column:instrument_id;index"`
	CompanyISIN    string                   `gorm:"column:company_isin"`
	Name           string                   `gorm:"column:name"`
	EnglishName    string                   `gorm:"column:english_name"`
	MarketType     consts.TSEMarketType     `gorm:"column:market_type;index"`
	Kind           consts.TSEInstrumentKind `gorm:"column:kind;index"`
	InstrumentType int                      `gorm:"column:instrument_type"` // TSETMC yVal
	Flow           int                      `gorm:"column:flow"`
	BoardCode      string                   `gorm:"column:board_code"`
	// Delisted is true when TSETMC no longer lists the instrument.
	Delisted          bool   `gorm:"column:delisted;not null;default:false;index"`
	ManuallyAdded     bool   `gorm:"column:manually_added;not null;default:false"`
	IndustryGroupCode string `gorm:"column:industry_group_code"`
	IndustryGroupName string `gorm:"column:industry_group_name"`
	IndustrySubCode   string `gorm:"column:industry_sub_code"`

	// InsertDate is the date the symbol was inserted on the board (تاریخ درج).
	InsertDate *time.Time `gorm:"column:insert_date"`
	// IPODate is the initial public offering date (تاریخ عرضه).
	IPODate       *time.Time `gorm:"column:ipo_date"`
	IPODateSource string     `gorm:"column:ipo_date_source"`
	// AcceptanceDate is the listing acceptance date (تاریخ پذیرش).
	AcceptanceDate *time.Time `gorm:"column:acceptance_date"`

	TotalShares   *float64   `gorm:"column:total_shares"`
	BaseVolume    *float64   `gorm:"column:base_volume"`
	Nav           *float64   `gorm:"column:nav"`
	EPS           *float64   `gorm:"column:eps"`
	SectorPE      *float64   `gorm:"column:sector_pe"`
	PSR           *float64   `gorm:"column:psr"`
	InfoUpdatedAt *time.Time `gorm:"column:info_updated_at"`

	// Panel controlled flags; the instrument sync never changes them.
	DataCollectionEnabled bool `gorm:"column:data_collection_enabled;not null;default:true;index"`
	EvaluationEnabled     bool `gorm:"column:evaluation_enabled;not null;default:false;index"`
}

func (TSEMarket) TableName() string { return "tse_markets" }
