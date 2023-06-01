package reporter

import (
	"encoding/json"
	"net/http"

	"github.com/bsn-eng/pon-wtfpl-relay/database"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ReporterServer struct {
	server *http.Server
	url    string
	db     database.DatabaseInterface
	log    *logrus.Entry
}

func NewReporterServer(URL string, DB *database.DatabaseInterface) *ReporterServer {
	return &ReporterServer{
		url: URL,
		db:  *DB,
		log: logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "Reporter API",
			"URL":     URL,
		})}
}

func (reporter *ReporterServer) Routes() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/reporter/blocksubmissions", reporter.handleGetBlockSubmissionsReporter).Methods(http.MethodPost)
	r.HandleFunc("/reporter/payloaddelivered", reporter.handleGetHeaderDeliveredReporter).Methods(http.MethodPost)
	r.HandleFunc("/reporter/proposerblindedblocks", reporter.handleBlindedBeaconBlockReporter).Methods(http.MethodPost)

	return loggingMiddleware(r, *reporter.log)
}

func (reporter *ReporterServer) StartServer() (err error) {
	reporter.log.Info("Reporter Server")
	reporter.server = &http.Server{
		Addr:    reporter.url,
		Handler: reporter.Routes(),
	}

	err = reporter.server.ListenAndServe()
	return err
}

func (reporter *ReporterServer) RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := HTTPError{Code: code, Message: message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		reporter.log.WithField("response", resp).WithError(err).Error("Couldn't write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (reporter *ReporterServer) RespondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		reporter.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (reporter *ReporterServer) handleGetBlockSubmissionsReporter(w http.ResponseWriter, req *http.Request) {

	reporter.log.WithFields(logrus.Fields{
		"method": "Reporter Block Submissions",
	}).Info("Reporter API")

	slotLimit := new(ReporterSlot)
	if err := json.NewDecoder(req.Body).Decode(&slotLimit); err != nil {
		reporter.log.WithError(err).Warn("could not decode payload")
		reporter.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	blockSubmissions, err := reporter.db.GetBuilderBlocksReporter(req.Context(), slotLimit.SlotLower, slotLimit.SlotUpper)
	if err != nil {
		reporter.log.WithError(err).Warn("Failed Block Request Reporter")
		reporter.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	reporter.RespondOK(w, &blockSubmissions)

}

func (reporter *ReporterServer) handleGetHeaderDeliveredReporter(w http.ResponseWriter, req *http.Request) {
	reporter.log.WithFields(logrus.Fields{
		"method": "Reporter Get Header Delivered",
	}).Info("Reporter API")

	slotLimit := new(ReporterSlot)
	if err := json.NewDecoder(req.Body).Decode(&slotLimit); err != nil {
		reporter.log.WithError(err).Warn("could not decode payload")
		reporter.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	GetHeaderDelivered, err := reporter.db.GetValidatorDeliveredHeaderReporter(req.Context(), slotLimit.SlotLower, slotLimit.SlotUpper)
	if err != nil {
		reporter.log.WithError(err).Warn("Failed Get Header Reporter")
		reporter.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	reporter.RespondOK(w, &GetHeaderDelivered)
}

func (reporter *ReporterServer) handleBlindedBeaconBlockReporter(w http.ResponseWriter, req *http.Request) {
	reporter.log.WithFields(logrus.Fields{
		"method": "Reporter Blinded Beacon Block",
	}).Info("Reporter API")

	slotLimit := new(ReporterSlot)
	if err := json.NewDecoder(req.Body).Decode(&slotLimit); err != nil {
		reporter.log.WithError(err).Warn("could not decode payload")
		reporter.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	BlindedBeaconBlock, err := reporter.db.GetValidatorReturnedBlocksReporter(req.Context(), slotLimit.SlotLower, slotLimit.SlotUpper)
	if err != nil {
		reporter.log.WithError(err).Warn("Failed Get Header Reporter")
		reporter.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	reporter.RespondOK(w, &BlindedBeaconBlock)

}
