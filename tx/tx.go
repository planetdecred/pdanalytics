package tx

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/web"
)

type Tx struct {
	server *web.Server
}

// Activate activates the Transaction module.
func Activate(webServer *web.Server) error {
	tx := &Tx{
		server: webServer,
	}

	// Register routes.
	tx.server.AddRoute("/tx/{txid}", web.GET, tx.TxPage, TransactionHashCtx)
	tx.server.AddRoute("/tx/{txid}/{inout}/{inoutid}", web.GET, tx.TxPage, TransactionHashCtx, TransactionIoIndexCtx)
	return nil
}

// TransactionHashCtx embeds "txid" into the request context
func TransactionHashCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txid := chi.URLParam(r, "txid")
		ctx := context.WithValue(r.Context(), ctxTxHash, txid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TransactionIoIndexCtx embeds "inout" and "inoutid" into the request context
func TransactionIoIndexCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inout := chi.URLParam(r, "inout")
		inoutid := chi.URLParam(r, "inoutid")
		ctx := context.WithValue(r.Context(), ctxTxInOut, inout)
		ctx = context.WithValue(ctx, ctxTxInOutId, inoutid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
