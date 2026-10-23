package runtime

import (
	"fmt"
	"strings"

	"github.com/h-varmazyar/gatec/pkg/strategy/ast"
	"github.com/shopspring/decimal"
)

type EvalContext struct {
	Values map[string]decimal.Decimal
	Funcs  map[string]func(args []decimal.Decimal) (decimal.Decimal, error)
}

func EvalBool(expr ast.Expression, ctx EvalContext) (bool, error) {
	v, err := eval(expr, ctx)
	if err != nil {
		return false, err
	}
	return !v.IsZero(), nil
}

func eval(expr ast.Expression, ctx EvalContext) (decimal.Decimal, error) {
	switch e := expr.(type) {
	case *ast.Literal:
		return e.Value, nil
	case *ast.BoolLiteral:
		if e.Value {
			return decimal.NewFromInt(1), nil
		}
		return decimal.Zero, nil
	case *ast.Identifier:
		k := strings.Join(e.Path, ".")
		v, ok := ctx.Values[k]
		if !ok {
			return decimal.Zero, fmt.Errorf("unknown identifier: %s", k)
		}
		return v, nil
	case *ast.UnaryExpression:
		v, err := eval(e.Expr, ctx)
		if err != nil {
			return decimal.Zero, err
		}
		if e.Operator == "!" {
			if v.IsZero() {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		}
		if e.Operator == "-" {
			return v.Neg(), nil
		}
		return decimal.Zero, fmt.Errorf("unsupported unary")
	case *ast.FunctionCall:
		fn, ok := ctx.Funcs[e.Name]
		if !ok {
			return decimal.Zero, fmt.Errorf("unknown function: %s", e.Name)
		}
		args := make([]decimal.Decimal, 0, len(e.Args))
		for _, a := range e.Args {
			av, err := eval(a, ctx)
			if err != nil {
				return decimal.Zero, err
			}
			args = append(args, av)
		}
		return fn(args)
	case *ast.BinaryExpression:
		l, err := eval(e.Left, ctx)
		if err != nil {
			return decimal.Zero, err
		}
		r, err := eval(e.Right, ctx)
		if err != nil {
			return decimal.Zero, err
		}
		switch e.Operator {
		case "+":
			return l.Add(r), nil
		case "-":
			return l.Sub(r), nil
		case "*":
			return l.Mul(r), nil
		case "/":
			return l.Div(r), nil
		case ">":
			if l.GreaterThan(r) {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case ">=":
			if l.GreaterThanOrEqual(r) {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case "<":
			if l.LessThan(r) {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case "<=":
			if l.LessThanOrEqual(r) {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case "==":
			if l.Equal(r) {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case "!=":
			if !l.Equal(r) {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case "&&":
			if !l.IsZero() && !r.IsZero() {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		case "||":
			if !l.IsZero() || !r.IsZero() {
				return decimal.NewFromInt(1), nil
			}
			return decimal.Zero, nil
		}
	}
	return decimal.Zero, fmt.Errorf("unsupported expression")
}
