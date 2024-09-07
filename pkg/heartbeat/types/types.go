package types

type QueryPlanTablesPayload struct {
	Tables []Table `json:"tables"`
}

type QueryPlanTablesResponse struct {
	Token string `json:"token"`
}

type QueryPlanQuery struct {
	ExecutedAt          int64  `json:"executed_at"`
	Duration            int64  `json:"duration"`
	Query               string `json:"query"`
	IsPreparedStatement bool   `json:"is_prepared_statement"`

	// Callstack?
}

type QueryPlanQueriesPayload struct {
	Queries []QueryPlanQuery `json:"queries"`
	// Transactions []QueryPlanTransaction `json:"transactions"`
}
