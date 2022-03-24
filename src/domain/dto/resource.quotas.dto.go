package dto

type ResourceQuotas struct {
	Labels map[string]string
	Items  []ResourceQuotasItem
}

type ResourceQuotasItem struct {
	Name             string `json:"name"`
	DisplayValue     string `json:"displayValue"`
	DisplayUsedValue string `json:"displayUsedValue"`
	LimitValue       int64  `json:"limitValue"`
	UsedValue        int64  `json:"usedValue"`
}
