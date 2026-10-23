package models

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestMarketHasManyTSEMarkets(t *testing.T) {
	marketSchema, err := schema.Parse(&Market{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}

	relation := marketSchema.Relationships.Relations["TSEMarkets"]
	if relation == nil {
		t.Fatal("Market.TSEMarkets relationship is missing")
	}
	if relation.Type != schema.HasMany {
		t.Fatalf("Market.TSEMarkets relationship type = %s, want %s", relation.Type, schema.HasMany)
	}

	var index *schema.Index
	for _, candidate := range marketSchema.ParseIndexes() {
		if candidate.Name == "idx_markets_provider_label" {
			index = candidate
			break
		}
	}
	if index == nil || index.Class != "UNIQUE" {
		t.Fatal("Market must have a unique provider_id/label index")
	}
	indexFields := make([]string, len(index.Fields))
	for i, field := range index.Fields {
		indexFields[i] = field.Field.DBName
	}
	if len(indexFields) != 2 || indexFields[0] != "provider_id" || indexFields[1] != "label" {
		t.Fatalf("Market provider/label index fields = %v, want [provider_id label]", indexFields)
	}

	tseMarketSchema, err := schema.Parse(&TSEMarket{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if !tseMarketSchema.FieldsByDBName["id"].PrimaryKey {
		t.Fatal("TSEMarket.id must be the primary key")
	}
	if tseMarketSchema.FieldsByDBName["market_id"].PrimaryKey {
		t.Fatal("TSEMarket.market_id must not be the primary key")
	}
	if tseMarketSchema.FieldsByDBName["label"] == nil {
		t.Fatal("TSEMarket.label is missing")
	}
}

func TestTSEDailyClosingPriceTracksTSEMarketAndMarket(t *testing.T) {
	dailySchema, err := schema.Parse(&TSEDailyClosingPrice{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"market_id", "tse_market_id", "candle_id"} {
		if dailySchema.FieldsByDBName[field] == nil {
			t.Errorf("TSEDailyClosingPrice.%s is missing", field)
		}
	}
	if !dailySchema.FieldsByDBName["id"].PrimaryKey {
		t.Fatal("TSEDailyClosingPrice.id must be the primary key")
	}
}
