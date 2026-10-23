package registry

type FunctionSpec struct {
	Name     string
	MinArgs  int
	MaxArgs  int
	Variadic bool
}

var Functions = map[string]FunctionSpec{
	"cross_over":  {Name: "cross_over", MinArgs: 2, MaxArgs: 2},
	"cross_under": {Name: "cross_under", MinArgs: 2, MaxArgs: 2},
	"sma":         {Name: "sma", MinArgs: 2, MaxArgs: 2},
	"ema":         {Name: "ema", MinArgs: 2, MaxArgs: 2},
	"max":         {Name: "max", MinArgs: 2, MaxArgs: 64, Variadic: true},
	"min":         {Name: "min", MinArgs: 2, MaxArgs: 64, Variadic: true},
}
