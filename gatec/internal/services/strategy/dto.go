package strategy

type StrategyDTO struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Provider    string `json:"provider"`
}

type StrategyListDTO struct {
	Strategies []*StrategyDTO `json:"strategies"`
}
