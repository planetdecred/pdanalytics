package web

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

// CacheControl creates a new middleware to set the HTTP response header with
// "Cache-Control: max-age=maxAge" where maxAge is in seconds.
func CacheControl(maxAge int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(maxAge, 10))
			next.ServeHTTP(w, r)
		})
	}
}

// chartDataTypeCtx returns a http.HandlerFunc that embeds the value at the url
// part {chartAxisType} into the request context.
func ChartDataTypeCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), CtxChartDataType,
			chi.URLParam(r, "chartDataType"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getChartDataTypeCtx retrieves the ctxChartAxisType data from the request context.
// If not set, the return value is an empty string.
func GetChartDataTypeCtx(r *http.Request) string {
	chartAxisType, ok := r.Context().Value(CtxChartDataType).(string)
	if !ok {
		log.Trace("chart axis type not set")
		return ""
	}
	return chartAxisType
}
