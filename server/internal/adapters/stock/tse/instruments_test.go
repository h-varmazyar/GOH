package tse

import (
	"testing"
	"time"
)

func TestParseJalaliDate(t *testing.T) {
	cases := map[string]string{
		"1405/08/03": "2026-10-25",
		"1404/01/01": "2025-03-21",
		"1403/12/30": "2025-03-20",
		"1399/01/01": "2020-03-20",
		"1390/08/25": "2011-11-16",
		"۱۴۰۲/۱۲/۲۹": "2024-03-19",
	}
	for input, want := range cases {
		got, err := parseJalaliDate(input)
		if err != nil {
			t.Fatalf("parseJalaliDate(%q): %v", input, err)
		}
		if got.Format(time.DateOnly) != want {
			t.Errorf("parseJalaliDate(%q) = %s, want %s", input, got.Format(time.DateOnly), want)
		}
	}
	for _, input := range []string{"", "1405/13/01", "1405/07/31", "abc"} {
		if _, err := parseJalaliDate(input); err == nil {
			t.Errorf("parseJalaliDate(%q) expected error", input)
		}
	}
}

func TestMatchKeyAndSimilarity(t *testing.T) {
	if matchKey("كاريز") != matchKey("کاریز") {
		t.Error("Arabic and Persian spellings must match")
	}
	if matchKey("آ س پ") != matchKey("آ.س.پ") {
		t.Error("separators must be ignored")
	}
	if matchKey("شکبیر۱") != matchKey("شکبیر1") {
		t.Error("Persian digits must match ASCII digits")
	}
	if cleanText("ﺻﻨﺪﻭﻕ  ﺱ") != "صندوق س" {
		t.Errorf("cleanText presentation forms = %q", cleanText("ﺻﻨﺪﻭﻕ  ﺱ"))
	}
	if s := nameSimilarity("سرمایه‌گذاری پارس آریان", "سرمايه گذاري پارس آريان"); s != 1 {
		t.Errorf("same name similarity = %v", s)
	}
	if s := nameSimilarity("فولاد مبارکه اصفهان", "بیمه زندگی کاریزما"); s >= minNameSimilarity {
		t.Errorf("different names similarity = %v", s)
	}
}

func TestParseIFBMarket(t *testing.T) {
	page := `<html><body>
<a href="/Instruments.aspx?id=1">menu link</a>
<table class="tblGrp">
<tr><th>ردیف</th><th>نماد</th><th>نام شرکت</th><th>سرمایه</th><th>صنعت</th><th>وضعیت</th></tr>
<tr><td>1</td><td><a target="_blank" href="/Instruments.aspx?id=17798"> آ س پ</a></td><td> آ.س.پ </td><td>1,000</td><td> انبوه سازي </td><td> مجاز </td></tr>
<tr><td>2</td><td><a href="/Instruments.aspx?id=22343">اپرداز</a></td><td>آتیه داده پرداز</td><td>2,250</td><td>اطلاعات</td><td>ممنوع-متوقف</td></tr>
</table></body></html>`
	rows, err := parseIFBMarket([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].SymID != "17798" || rows[0].Symbol != "آ س پ" || rows[0].Industry != "انبوه سازی" || rows[0].State != ifbAllowedState {
		t.Errorf("unexpected first row %+v", rows[0])
	}
	if rows[1].State == ifbAllowedState {
		t.Errorf("second row must not be allowed: %+v", rows[1])
	}
}

func TestParseIFBListings(t *testing.T) {
	page := `<table class="tblGrp OmidnameTable">
<tr><th>نماد</th><th>نام شرکت</th><th>تاریخ درج</th><th>تاریخ عرضه</th><th>تاریخ پذیرش</th><th></th><th></th><th>لغو پذیرش</th></tr>
<tr symid="21743" year="1400"><td>حفجر</td><td><a href="/Instruments.aspx?id=21743">کشتیرانی والفجر</a></td><td>1400/03/19</td><td></td><td>1399/07/21</td><td><i class="icon-file-download"></i></td><td></td><td><i class='icon-check btn'></i></td></tr>
<tr symid="30374" year="1405"><td>آواپارس</td><td><a href="/Instruments.aspx?id=30374">بیمه اتکایی آوای پارس</a></td><td>1405/06/28</td><td>1405/07/18</td><td>1405/06/04</td><td><i class="icon-file-download"></i></td><td></td><td><i class='icon-close btn'></i></td></tr>
<tr symid="21743" year="1394"><td>حفجر</td><td>کشتيراني والفجر</td><td>1394/10/02</td><td></td><td>1393/08/25</td><td></td><td></td><td><i class='icon-close btn'></i></td></tr>
</table>`
	rows, err := parseIFBListings([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	listing := rows["30374"]
	if listing.Cancelled || listing.InsertDate == nil || listing.InsertDate.Format(time.DateOnly) != "2026-09-19" ||
		listing.ReleaseDate == nil || listing.ReleaseDate.Format(time.DateOnly) != "2026-10-10" ||
		listing.AcceptanceDate == nil || listing.AcceptanceDate.Format(time.DateOnly) != "2026-08-26" {
		t.Errorf("unexpected listing %+v", listing)
	}
	relisted := rows["21743"]
	if relisted.Cancelled || relisted.ReleaseDate != nil || relisted.InsertDate.Format(time.DateOnly) != "2015-12-23" {
		t.Errorf("the active row must win over the cancelled one: %+v", relisted)
	}
}
