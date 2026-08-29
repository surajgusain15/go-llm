package database

type QueryResult struct {
	Columns        []string `json:"Columns"`
	Rows           [][]any  `json:"Rows"`
	Count          int      `json:"Count"`
	Truncated      bool     `json:"Truncated"`
	TruncateReason string   `json:"TruncateReason,omitempty"`
}
