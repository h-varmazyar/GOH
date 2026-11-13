package consts

// TSETradeDayStatus is the result of fetching the trade history of one TSE
// market on one day.
type TSETradeDayStatus string

const (
	// TSETradeDayFetched: trades were received and stored.
	TSETradeDayFetched TSETradeDayStatus = "FETCHED"
	// TSETradeDayEmpty: the request succeeded but returned no trades. The symbol
	// was likely not traded or closed that day, or TSETMC returned nothing.
	TSETradeDayEmpty TSETradeDayStatus = "EMPTY"
	// TSETradeDayFailed: the request or storing the trades failed.
	TSETradeDayFailed TSETradeDayStatus = "FAILED"
	// TSETradeDayMissing is never stored: it is a day that has not been fetched yet.
	TSETradeDayMissing TSETradeDayStatus = "MISSING"
	// TSETradeDayMismatched is never stored: a fetched or empty day whose trade
	// count or volume, canceled trades left out, differs from its closing price.
	TSETradeDayMismatched TSETradeDayStatus = "MISMATCHED"
)

// TSETradeDayStatuses are the statuses that days can be selected by.
var TSETradeDayStatuses = []TSETradeDayStatus{
	TSETradeDayMissing, TSETradeDayFailed, TSETradeDayEmpty, TSETradeDayFetched, TSETradeDayMismatched,
}

func (s TSETradeDayStatus) Valid() bool {
	for _, status := range TSETradeDayStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// TSETradeHistoryJobStatus is the state of a queued trade history collection.
type TSETradeHistoryJobStatus string

const (
	TSETradeHistoryJobPending  TSETradeHistoryJobStatus = "PENDING"
	TSETradeHistoryJobRunning  TSETradeHistoryJobStatus = "RUNNING"
	TSETradeHistoryJobDone     TSETradeHistoryJobStatus = "DONE"
	TSETradeHistoryJobFailed   TSETradeHistoryJobStatus = "FAILED"
	TSETradeHistoryJobCanceled TSETradeHistoryJobStatus = "CANCELED"
)

// TSETradeHistoryJobActive are the statuses of jobs that are not finished.
var TSETradeHistoryJobActive = []TSETradeHistoryJobStatus{TSETradeHistoryJobPending, TSETradeHistoryJobRunning}
