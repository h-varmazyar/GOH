package workers

import (
	"testing"

	"github.com/h-varmazyar/gatec/internal/models"
)

// The instances, wake-ups and stale job checks are tested in the jobrunner package.

func TestJobProgressListsCurrentMarkets(t *testing.T) {
	progress := newJobProgress()
	progress.start(&models.TSEMarket{ID: 1, Label: "فولاد"})
	progress.start(&models.TSEMarket{ID: 2, Label: "خودرو"})
	progress.finish(&models.TSEMarket{ID: 1})
	if got := progress.columns()["current_market"]; got != "خودرو" {
		t.Fatalf("current market = %q", got)
	}
}
