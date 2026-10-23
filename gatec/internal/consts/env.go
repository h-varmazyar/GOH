package consts

import "strings"

type AppEnv string

const (
	ProductionEnv AppEnv = "production"
	StageEnv      AppEnv = "stage"
	DevelopEnv    AppEnv = "develop"
	LocalEnv      AppEnv = "local"
)

func Environment(env string) AppEnv {
	return AppEnv(strings.ToLower(env))
}
