package tx

type contexKey int

const (
	ctxTxHash contexKey = iota
	ctxTxInOut
	ctxTxInOutId
)
