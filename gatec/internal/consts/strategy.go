package consts

type StrategyStatus string

const (
	StrategyStatusEnable  StrategyStatus = "ENABLE"
	StrategyStatusDISABLE StrategyStatus = "DISABLE"
)

var strategyStatusPersianLabel = map[StrategyStatus]string{
	StrategyStatusEnable:  "فعال",
	StrategyStatusDISABLE: "غیرفعال",
}

func (ss StrategyStatus) PersianLabel() string {
	if label, exists := strategyStatusPersianLabel[ss]; exists {
		return label
	}
	return ""
}
