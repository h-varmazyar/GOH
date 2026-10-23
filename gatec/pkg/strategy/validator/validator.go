package validator

import (
	"fmt"

	"github.com/h-varmazyar/gatec/pkg/strategy/loader"
	// "github.com/h-varmazyar/gatec/pkg/strategy/parser"
)

func ValidateStrategyFile(s *loader.StrategyFile) error {
	if s == nil {
		return fmt.Errorf("strategy file is nil")
	}
	if err := s.Validate(); err != nil {
		return err
	}
	// for _, expr := range collectRules(s) {
	// 	if expr == "" {
	// 		continue
	// 	}
	// 	p, err := parser.New(expr)
	// 	if err != nil {
	// 		return fmt.Errorf("parser init failed for %q: %w", expr, err)
	// 	}
	// 	if _, err := p.ParseExpression(); err != nil {
	// 		return fmt.Errorf("invalid expression %q: %w", expr, err)
	// 	}
	// }
	return nil
}

// func collectRules(s *loader.StrategyFile) []string {
// out := make([]string, 0)
// out = append(out, s.Filters.Rules...)
// out = append(out, s.Entry.Long.All...)
// out = append(out, s.Entry.Long.Any...)
// out = append(out, s.Entry.Short.All...)
// out = append(out, s.Entry.Short.Any...)
// out = append(out, s.Exit.Long.All...)
// out = append(out, s.Exit.Long.Any...)
// out = append(out, s.Exit.Short.All...)
// out = append(out, s.Exit.Short.Any...)
// return out
// }
