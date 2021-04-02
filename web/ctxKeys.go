package web

type contextKey int

const (
	CtxXcToken contextKey = iota
	CtxChartDataType
	CtxTimestamp
	CtxProposalRefID
	CtxProposalToken
	CtxAgendaId
)
