package consts

// TSEIssueType is a problem of the TSE catalog that the panel user decides on.
type TSEIssueType string

const (
	// TSEIssueNameMismatch: insCodes of one market whose names differ, such as two
	// companies that used the same symbol.
	TSEIssueNameMismatch TSEIssueType = "NAME_MISMATCH"
	// TSEIssueSameName: markets of different symbols with the same company name,
	// such as a symbol renamed on moving between boards.
	TSEIssueSameName TSEIssueType = "SAME_NAME"
	// TSEIssueMultipleActive: several listed insCodes of one market collect data;
	// prices of the same day would share one candle.
	TSEIssueMultipleActive TSEIssueType = "MULTIPLE_ACTIVE"
	// TSEIssuePriceOverlap: insCodes of one market have prices on the same days.
	TSEIssuePriceOverlap TSEIssueType = "PRICE_OVERLAP"
	// TSEIssueKindConflict: a symbol is used by instruments of different kinds.
	TSEIssueKindConflict TSEIssueType = "KIND_CONFLICT"
	// TSEIssueLabelConflict: a symbol was renamed to the label of another market.
	TSEIssueLabelConflict TSEIssueType = "LABEL_CONFLICT"
	// TSEIssueSkipped: an instrument of the instrument files could not be read from TSETMC.
	TSEIssueSkipped TSEIssueType = "SKIPPED"
	// TSEIssueSymbolChanged: TSETMC reports another symbol than the instrument files.
	TSEIssueSymbolChanged TSEIssueType = "SYMBOL_CHANGED"
	// TSEIssueIFBBoardUnmatched: an IFB first or second market symbol has no active TSETMC instrument.
	TSEIssueIFBBoardUnmatched TSEIssueType = "IFB_BOARD_UNMATCHED"
	// TSEIssueIFBNameMismatch: the IFB name of a board symbol differs from its TSETMC name.
	TSEIssueIFBNameMismatch TSEIssueType = "IFB_NAME_MISMATCH"
)

var tseIssueTypePersianLabel = map[TSEIssueType]string{
	TSEIssueNameMismatch:      "نام متفاوت در یک مارکت",
	TSEIssueSameName:          "نام یکسان با نماد متفاوت",
	TSEIssueMultipleActive:    "چند insCode فعال در یک مارکت",
	TSEIssuePriceOverlap:      "تداخل قیمت در یک روز",
	TSEIssueKindConflict:      "نماد مشترک با نوع متفاوت",
	TSEIssueLabelConflict:     "تغییر نماد به برچسب مارکت دیگر",
	TSEIssueSkipped:           "دریافت‌نشده از TSETMC",
	TSEIssueSymbolChanged:     "تفاوت نماد فایل‌ها و TSETMC",
	TSEIssueIFBBoardUnmatched: "نماد فرابورس بدون نماد فعال در TSETMC",
	TSEIssueIFBNameMismatch:   "تفاوت نام فرابورس و TSETMC",
}

func (t TSEIssueType) PersianLabel() string { return tseIssueTypePersianLabel[t] }

// TSEIssueSyncTypes are found while the catalog is built; the others are found by
// checking the stored markets.
var TSEIssueSyncTypes = []TSEIssueType{
	TSEIssueKindConflict, TSEIssueLabelConflict, TSEIssueSkipped, TSEIssueSymbolChanged,
	TSEIssueIFBBoardUnmatched, TSEIssueIFBNameMismatch,
}

var TSEIssueCheckTypes = []TSEIssueType{
	TSEIssueNameMismatch, TSEIssueSameName, TSEIssueMultipleActive, TSEIssuePriceOverlap,
}

type TSEIssueSeverity string

const (
	TSEIssueSeverityError   TSEIssueSeverity = "ERROR"
	TSEIssueSeverityWarning TSEIssueSeverity = "WARNING"
	TSEIssueSeverityInfo    TSEIssueSeverity = "INFO"
)

// TSEIssueStatus is the state of an issue. A resolved issue that is found again is
// reopened; an ignored one stays ignored.
type TSEIssueStatus string

const (
	TSEIssueStatusOpen TSEIssueStatus = "OPEN"
	// TSEIssueStatusIgnored is set in the panel when the user accepts the situation.
	TSEIssueStatusIgnored TSEIssueStatus = "IGNORED"
	// TSEIssueStatusResolved is set when the issue is no longer found.
	TSEIssueStatusResolved TSEIssueStatus = "RESOLVED"
)
