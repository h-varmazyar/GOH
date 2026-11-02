package tsemarket

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/tsemarket"
)

type issueFilterReq struct {
	Query    string `json:"q,omitempty" form:"q"`
	Type     string `json:"type,omitempty" form:"type"`
	Severity string `json:"severity,omitempty" form:"severity"`
	Status   string `json:"status,omitempty" form:"status"`
	InsCode  string `json:"ins_code,omitempty" form:"ins_code"`
	MarketID uint   `json:"market_id,omitempty" form:"market_id"`
}

func (r issueFilterReq) toFilter() repositories.TSEIssueFilter {
	return repositories.TSEIssueFilter{
		Query:    r.Query,
		Type:     consts.TSEIssueType(r.Type),
		Severity: consts.TSEIssueSeverity(r.Severity),
		Status:   consts.TSEIssueStatus(r.Status),
		InsCode:  r.InsCode,
		MarketID: r.MarketID,
	}
}

// issueStatusReq changes the status of the issues in IDs, or of every issue that
// matches Filter when IDs is empty.
type issueStatusReq struct {
	Status string          `json:"status" binding:"required,oneof=OPEN IGNORED" example:"IGNORED"`
	Note   *string         `json:"note,omitempty"`
	IDs    []uint          `json:"ids,omitempty"`
	Filter *issueFilterReq `json:"filter,omitempty"`
}

type issueResp struct {
	ID          uint         `json:"id"`
	Type        string       `json:"type"`
	TypeLabel   string       `json:"type_label"`
	Severity    string       `json:"severity"`
	Status      string       `json:"status"`
	Symbol      string       `json:"symbol"`
	Name        string       `json:"name"`
	Detail      string       `json:"detail"`
	InsCodes    []string     `json:"ins_codes"`
	MarketIDs   []uint       `json:"market_ids"`
	Note        string       `json:"note"`
	CreatedAt   time.Time    `json:"created_at"`
	DetectedAt  time.Time    `json:"detected_at"`
	ResolvedAt  *time.Time   `json:"resolved_at"`
	Instruments []marketResp `json:"instruments"`
}

func toIssueResp(view tsemarket.IssueView, members map[uint]int64) issueResp {
	issue := view.Issue
	resp := issueResp{
		ID:          issue.ID,
		Type:        string(issue.Type),
		TypeLabel:   issue.Type.PersianLabel(),
		Severity:    string(issue.Severity),
		Status:      string(issue.Status),
		Symbol:      issue.Symbol,
		Name:        issue.Name,
		Detail:      issue.Detail,
		InsCodes:    nonNil(issue.InsCodes),
		MarketIDs:   nonNil(issue.MarketIDs),
		Note:        issue.Note,
		CreatedAt:   issue.CreatedAt,
		DetectedAt:  issue.DetectedAt,
		ResolvedAt:  issue.ResolvedAt,
		Instruments: make([]marketResp, len(view.Instruments)),
	}
	for i, market := range view.Instruments {
		resp.Instruments[i] = toMarketResp(market, members[market.MarketID])
	}
	return resp
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

type issueTypeResp struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

type issueCountResp struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
	Count    int64  `json:"count"`
}

type issueSummaryResp struct {
	Types  []issueTypeResp  `json:"types"`
	Counts []issueCountResp `json:"counts"`
}

type detectResp struct {
	Found    int   `json:"found"`
	Resolved int64 `json:"resolved"`
}

type issueStatusResp struct {
	Updated int64 `json:"updated"`
}
