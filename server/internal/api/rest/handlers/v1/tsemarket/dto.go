package tsemarket

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
)

// --- Request DTOs ---

type createMarketReq struct {
	Symbol  string `json:"symbol" binding:"required" example:"فولاد"`
	InsCode string `json:"ins_code" binding:"required" example:"46348559193224090"`
}

type flagsReq struct {
	DataCollectionEnabled *bool `json:"data_collection_enabled,omitempty" example:"true"`
	EvaluationEnabled     *bool `json:"evaluation_enabled,omitempty" example:"false"`
}

type filterReq struct {
	Query             string `json:"q,omitempty" form:"q"`
	MarketID          uint   `json:"market_id,omitempty" form:"market_id"`
	MarketType        string `json:"market_type,omitempty" form:"market_type"`
	Kind              string `json:"kind,omitempty" form:"kind"`
	IndustryGroupCode string `json:"group,omitempty" form:"group"`
	Delisted          *bool  `json:"delisted,omitempty" form:"delisted"`
	DataCollection    *bool  `json:"data_collection,omitempty" form:"data_collection"`
	Evaluation        *bool  `json:"evaluation,omitempty" form:"evaluation"`
}

func (r filterReq) toFilter() repositories.TSEMarketFilter {
	return repositories.TSEMarketFilter{
		Query:                 r.Query,
		MarketID:              r.MarketID,
		MarketType:            consts.TSEMarketType(r.MarketType),
		Kind:                  consts.TSEInstrumentKind(r.Kind),
		IndustryGroupCode:     r.IndustryGroupCode,
		Delisted:              r.Delisted,
		DataCollectionEnabled: r.DataCollection,
		EvaluationEnabled:     r.Evaluation,
	}
}

type delistedReq struct {
	Delisted *bool `json:"delisted" binding:"required" example:"true"`
}

type linkReq struct {
	// MarketLabel is the label of an existing market, or of a new market to create.
	MarketLabel string `json:"market_label" binding:"required" example:"زنجان"`
}

type deleteResp struct {
	DeletedInsCodes int `json:"deleted_ins_codes"`
	DeletedMarkets  int `json:"deleted_markets"`
}

// bulkFlagsReq changes the flags of the markets in IDs, or of every market that
// matches Filter when IDs is empty.
type bulkFlagsReq struct {
	flagsReq
	IDs    []uint     `json:"ids,omitempty"`
	Filter *filterReq `json:"filter,omitempty"`
}

// --- Response DTOs ---

type errorResp struct {
	Error string `json:"error"`
}

type bulkFlagsResp struct {
	Updated int64 `json:"updated"`
}

type industryGroupResp struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type marketResp struct {
	ID                    uint       `json:"id"`
	MarketID              uint       `json:"market_id"`
	InsCode               string     `json:"ins_code"`
	InstrumentID          string     `json:"instrument_id"`
	CompanyISIN           string     `json:"company_isin"`
	Symbol                string     `json:"symbol"`
	MarketLabel           string     `json:"market_label"`
	MarketMembers         int64      `json:"market_members"`
	Name                  string     `json:"name"`
	EnglishName           string     `json:"english_name"`
	MarketType            string     `json:"market_type"`
	MarketTypeLabel       string     `json:"market_type_label"`
	Kind                  string     `json:"kind"`
	KindLabel             string     `json:"kind_label"`
	InstrumentType        int        `json:"instrument_type"`
	BoardCode             string     `json:"board_code"`
	IndustryGroupCode     string     `json:"industry_group_code"`
	IndustryGroupName     string     `json:"industry_group_name"`
	IndustrySubCode       string     `json:"industry_sub_code"`
	Delisted              bool       `json:"delisted"`
	ManuallyAdded         bool       `json:"manually_added"`
	InsertDate            *time.Time `json:"insert_date"`
	ReleaseDate           *time.Time `json:"release_date"`
	ReleaseDateSource     string     `json:"release_date_source"`
	AcceptanceDate        *time.Time `json:"acceptance_date"`
	TotalShares           *float64   `json:"total_shares"`
	BaseVolume            *float64   `json:"base_volume"`
	Nav                   *float64   `json:"nav"`
	EPS                   *float64   `json:"eps"`
	SectorPE              *float64   `json:"sector_pe"`
	PSR                   *float64   `json:"psr"`
	InfoUpdatedAt         *time.Time `json:"info_updated_at"`
	DataCollectionEnabled bool       `json:"data_collection_enabled"`
	EvaluationEnabled     bool       `json:"evaluation_enabled"`
}

// toMarketResp converts a market; members is the number of instruments of its market.
func toMarketResp(market *models.TSEMarket, members int64) marketResp {
	resp := marketResp{
		ID:                    market.ID,
		MarketID:              market.MarketID,
		MarketMembers:         members,
		InsCode:               market.InsCode,
		InstrumentID:          market.InstrumentID,
		CompanyISIN:           market.CompanyISIN,
		Symbol:                market.Label,
		Name:                  market.Name,
		EnglishName:           market.EnglishName,
		MarketType:            string(market.MarketType),
		MarketTypeLabel:       market.MarketType.PersianLabel(),
		Kind:                  string(market.Kind),
		KindLabel:             market.Kind.PersianLabel(),
		InstrumentType:        market.InstrumentType,
		BoardCode:             market.BoardCode,
		IndustryGroupCode:     market.IndustryGroupCode,
		IndustryGroupName:     market.IndustryGroupName,
		IndustrySubCode:       market.IndustrySubCode,
		Delisted:              market.Delisted,
		ManuallyAdded:         market.ManuallyAdded,
		InsertDate:            market.InsertDate,
		ReleaseDate:           market.IPODate,
		ReleaseDateSource:     market.IPODateSource,
		AcceptanceDate:        market.AcceptanceDate,
		TotalShares:           market.TotalShares,
		BaseVolume:            market.BaseVolume,
		Nav:                   market.Nav,
		EPS:                   market.EPS,
		SectorPE:              market.SectorPE,
		PSR:                   market.PSR,
		InfoUpdatedAt:         market.InfoUpdatedAt,
		DataCollectionEnabled: market.DataCollectionEnabled,
		EvaluationEnabled:     market.EvaluationEnabled,
	}
	if market.Market != nil {
		resp.MarketLabel = market.Market.Label
	}
	return resp
}
