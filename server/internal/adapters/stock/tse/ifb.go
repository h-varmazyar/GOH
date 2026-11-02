package tse

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	ifbListingURL = "https://www.ifb.ir/omidnameh.aspx"
	ifbMarketURL  = "https://www.ifb.ir/markets.aspx?id=%d"

	ifbFirstMarketID  = 1
	ifbSecondMarketID = 2

	ifbAllowedState = "مجاز"
)

// ifbMarketRow is a row of the IFB market page (markets.aspx).
type ifbMarketRow struct {
	SymID    string
	Symbol   string
	Name     string
	Industry string
	State    string
}

// ifbListingRow is a row of the IFB listing page (omidnameh.aspx).
type ifbListingRow struct {
	SymID          string
	Symbol         string
	Name           string
	InsertDate     *time.Time
	ReleaseDate    *time.Time
	AcceptanceDate *time.Time
	Cancelled      bool
}

func (a *Adapter) fetchIFBMarket(ctx context.Context, marketID int) ([]ifbMarketRow, error) {
	body, err := a.get(ctx, fmt.Sprintf(ifbMarketURL, marketID), "text/html")
	if err != nil {
		return nil, err
	}
	return parseIFBMarket(body)
}

func (a *Adapter) fetchIFBListings(ctx context.Context) (map[string]ifbListingRow, error) {
	body, err := a.get(ctx, ifbListingURL, "text/html")
	if err != nil {
		return nil, err
	}
	return parseIFBListings(body)
}

// parseIFBMarket extracts the symbols table of markets.aspx. Columns are:
// row number, symbol, company name, capital, industry and trading state.
func parseIFBMarket(body []byte) ([]ifbMarketRow, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse IFB market page: %w", err)
	}
	result := make([]ifbMarketRow, 0)
	for _, table := range findElements(doc, "table") {
		if !hasClass(table, "tblGrp") {
			continue
		}
		for _, row := range findElements(table, "tr") {
			cells := childElements(row, "td")
			if len(cells) < 6 {
				continue
			}
			symID := ""
			for _, link := range findElements(cells[1], "a") {
				if symID = instrumentPageID(attr(link, "href")); symID != "" {
					break
				}
			}
			if symID == "" {
				continue
			}
			result = append(result, ifbMarketRow{
				SymID:    symID,
				Symbol:   nodeText(cells[1]),
				Name:     nodeText(cells[2]),
				Industry: nodeText(cells[4]),
				State:    nodeText(cells[5]),
			})
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("IFB market page has no symbol rows")
	}
	return result, nil
}

// parseIFBListings extracts the rows of omidnameh.aspx keyed by IFB symbol ID.
// Columns are: symbol, company name, insert date, release date, acceptance date,
// prospectus, valuation and listing cancellation. A symbol may be listed more than
// once; rows are ordered newest first and an active row wins over a cancelled one.
func parseIFBListings(body []byte) (map[string]ifbListingRow, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse IFB listing page: %w", err)
	}
	result := make(map[string]ifbListingRow)
	for _, row := range findElements(doc, "tr") {
		symID := strings.TrimSpace(attr(row, "symid"))
		if symID == "" {
			continue
		}
		cells := childElements(row, "td")
		if len(cells) < 5 {
			continue
		}
		cancelled := false
		for _, icon := range findElements(cells[len(cells)-1], "i") {
			if hasClass(icon, "icon-check") {
				cancelled = true
			}
		}
		listing := ifbListingRow{
			SymID:          symID,
			Symbol:         nodeText(cells[0]),
			Name:           nodeText(cells[1]),
			InsertDate:     optionalJalaliDate(nodeText(cells[2])),
			ReleaseDate:    optionalJalaliDate(nodeText(cells[3])),
			AcceptanceDate: optionalJalaliDate(nodeText(cells[4])),
			Cancelled:      cancelled,
		}
		if existing, ok := result[symID]; ok && (cancelled || !existing.Cancelled) {
			continue
		}
		result[symID] = listing
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("IFB listing page has no symbol rows")
	}
	return result, nil
}

func optionalJalaliDate(value string) *time.Time {
	if value == "" {
		return nil
	}
	date, err := parseJalaliDate(value)
	if err != nil {
		return nil
	}
	return &date
}

func instrumentPageID(href string) string {
	parsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil || !strings.EqualFold(strings.TrimPrefix(parsed.Path, "/"), "Instruments.aspx") {
		return ""
	}
	return parsed.Query().Get("id")
}

func findElements(root *html.Node, tag string) []*html.Node {
	var result []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode && child.Data == tag {
				result = append(result, child)
			}
			walk(child)
		}
	}
	walk(root)
	return result
}

func childElements(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			result = append(result, child)
		}
	}
	return result
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attr(n, "class")), class)
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return cleanText(b.String())
}
