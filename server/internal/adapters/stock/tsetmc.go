package stock

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/sirupsen/logrus"
// 	"github.com/xuri/excelize/v2"
// )

// const (
// 	baseURL = "https://members.tsetmc.com/tsev2/excel/MarketWatchPlus.aspx"
// )

// // StockRecord represents a single row of stock market data
// type StockRecord struct {
// 	InsCode      string
// 	Symbol       string  // نماد
// 	Name         string  // نام
// 	TradeCount   int64   // تعداد
// 	Volume       int64   // حجم
// 	Value        int64   // ارزش
// 	Yesterday    float64 // دیروز
// 	First        float64 // اولین
// 	Last         float64 // آخرین معامله - مقدار
// 	LastChange   float64 // آخرین معامله - تغییر
// 	LastPercent  float64 // آخرین معامله - درصد
// 	Close        float64 // قیمت پایانی - مقدار
// 	CloseChange  float64 // قیمت پایانی - تغییر
// 	ClosePercent float64 // قیمت پایانی - درصد
// 	Low          float64 // کمترین
// 	High         float64 // بیشترین
// }

// // Adapter is the TSETMC stock market adapter
// type Adapter struct {
// 	httpClient *http.Client
// 	logger     *logrus.Entry
// }

// // NewAdapter creates a new TSETMC stock adapter
// func NewAdapter() *Adapter {
// 	return &Adapter{
// 		httpClient: &http.Client{
// 			Timeout: 60 * time.Second,
// 		},
// 		logger: logrus.WithField("adapter", "tsetmc"),
// 	}
// }

// // FetchMarketWatch fetches and parses the market watch data for a given Shamsi date (e.g. "1404/07/12")
// func (a *Adapter) FetchMarketWatch(ctx context.Context, shamsiDate string) ([]StockRecord, error) {
// 	url := fmt.Sprintf("%s?d=%s", baseURL, shamsiDate)

// 	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %w", err)
// 	}

// 	resp, err := a.httpClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch market watch: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read response body: %w", err)
// 	}

// 	return a.parseExcel(body)
// }

// // FetchMarketWatch fetches and parses the market watch data for a given Shamsi date (e.g. "1404/07/12")
// func (a *Adapter) FetchMarketWatchHtml(ctx context.Context, shamsiDate string) ([]StockRecord, error) {
// 	url := fmt.Sprintf("%s?d=%s", baseURL, shamsiDate)

// 	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %w", err)
// 	}

// 	resp, err := a.httpClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch market watch: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read response body: %w", err)
// 	}

// 	return a.parseExcel(body)
// }

// // parseExcel parses the Excel content and returns stock records
// func (a *Adapter) parseExcel(data []byte) ([]StockRecord, error) {
// 	f, err := excelize.OpenReader(bytes.NewReader(data))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to open excel: %w", err)
// 	}
// 	defer f.Close()

// 	sheetName := f.GetSheetName(0)
// 	rows, err := f.GetRows(sheetName)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get rows: %w", err)
// 	}

// 	records := make([]StockRecord, 0, len(rows))

// 	// The Excel structure from TSETMC:
// 	// Row 1: Title "دیده بان بازار"
// 	// Row 2: Date header + column names (نماد, نام, تعداد, حجم, ارزش, ...)
// 	// Row 3+: Data rows
// 	// Data columns are offset by 1 (first column is empty/index)
// 	headerFound := false
// 	for _, row := range rows {
// 		if len(row) < 2 {
// 			continue
// 		}

// 		// Detect header row by looking for "نماد" keyword
// 		for _, cell := range row {
// 			if strings.TrimSpace(cell) == "نماد" {
// 				headerFound = true
// 				break
// 			}
// 		}
// 		if !headerFound {
// 			continue
// 		}

// 		// Find the symbol column index dynamically
// 		// Skip the header row itself
// 		symbol := ""
// 		dataStartIdx := -1
// 		for i, cell := range row {
// 			trimmed := strings.TrimSpace(cell)
// 			if trimmed == "نماد" {
// 				// This is still the header row
// 				dataStartIdx = -1
// 				break
// 			}
// 			if trimmed != "" && dataStartIdx == -1 {
// 				dataStartIdx = i
// 				symbol = trimmed
// 				break
// 			}
// 		}

// 		if dataStartIdx == -1 || symbol == "" {
// 			continue
// 		}

// 		// Extract data based on offset from symbol position
// 		// Expected columns after symbol: نام, تعداد, حجم, ارزش, دیروز, اولین, آخرین(مقدار), آخرین(تغییر), آخرین(درصد), پایانی(مقدار), پایانی(تغییر), پایانی(درصد), کمترین, بیشترین
// 		getName := func(idx int) string {
// 			if idx < len(row) {
// 				return strings.TrimSpace(row[idx])
// 			}
// 			return ""
// 		}
// 		getFloat := func(idx int) float64 {
// 			if idx < len(row) {
// 				return parseFloatSafe(row[idx])
// 			}
// 			return 0
// 		}
// 		getInt := func(idx int) int64 {
// 			if idx < len(row) {
// 				return parseIntSafe(row[idx])
// 			}
// 			return 0
// 		}

// 		record := StockRecord{
// 			Symbol:       symbol,
// 			Name:         getName(dataStartIdx + 1),
// 			TradeCount:   getInt(dataStartIdx + 2),
// 			Volume:       getInt(dataStartIdx + 3),
// 			Value:        getInt(dataStartIdx + 4),
// 			Yesterday:    getFloat(dataStartIdx + 5),
// 			First:        getFloat(dataStartIdx + 6),
// 			Last:         getFloat(dataStartIdx + 7),
// 			LastChange:   getFloat(dataStartIdx + 8),
// 			LastPercent:  getFloat(dataStartIdx + 9),
// 			Close:        getFloat(dataStartIdx + 10),
// 			CloseChange:  getFloat(dataStartIdx + 11),
// 			ClosePercent: getFloat(dataStartIdx + 12),
// 			Low:          getFloat(dataStartIdx + 13),
// 			High:         getFloat(dataStartIdx + 14),
// 		}

// 		records = append(records, record)
// 	}

// 	a.logger.Infof("parsed %d stock records", len(records))
// 	return records, nil
// }

// func parseFloatSafe(s string) float64 {
// 	s = strings.TrimSpace(s)
// 	s = strings.ReplaceAll(s, ",", "")
// 	val, _ := strconv.ParseFloat(s, 64)
// 	return val
// }

// func parseIntSafe(s string) int64 {
// 	s = strings.TrimSpace(s)
// 	s = strings.ReplaceAll(s, ",", "")
// 	val, _ := strconv.ParseInt(s, 10, 64)
// 	return val
// }
