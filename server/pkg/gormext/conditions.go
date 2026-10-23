package gormext

import "fmt"

var Where = func(columnName string, value any) (string, any) {
	return fmt.Sprintf("%v = ?", columnName), value
}
