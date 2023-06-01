package reporter

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type ReporterSlot struct {
	SlotLower uint64 `json:"slot_lower"`
	SlotUpper uint64 `json:"slot_upper"`
}

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func loggingMiddleware(next http.Handler, logger logrus.Entry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
