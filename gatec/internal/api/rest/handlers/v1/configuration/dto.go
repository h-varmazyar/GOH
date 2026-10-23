package configuration

type UpdateSectionReq struct {
	Values map[string]string `json:"values" binding:"required"`
}

type SetWorkerEnabledReq struct {
	Enabled *bool `json:"enabled" binding:"required"`
}
