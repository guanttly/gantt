package model

type ReRankParam struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}
type ReRankResult struct {
	Documents []string `json:"documents"`
}
