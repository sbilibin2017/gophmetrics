package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// Updater updates metric values.
type Updater interface {
	Update(ctx context.Context, metric *models.Metrics) (*models.Metrics, error)
}

// Getter retrieves a metric.
type Getter interface {
	Get(ctx context.Context, id *models.MetricID) (*models.Metrics, error)
}

// Lister lists all metrics.
type Lister interface {
	List(ctx context.Context) ([]*models.Metrics, error)
}

// NewMetricUpdatePathHandler saves or updates a metric.
//
// @Summary Save or update a metric
// @Description Updates a metric value or delta via POST request with URL parameters
// @Tags metrics
// @Accept plain
// @Produce plain
// @Param type path string true "Metric type (gauge or counter)"
// @Param name path string true "Metric name"
// @Param value path string true "Metric value (float for gauge, int for counter)"
// @Success 200 "OK"
// @Failure 400 "Bad Request"
// @Failure 404 "Not Found"
// @Failure 500 "Internal Server Error"
// @Router /update/{type}/{name}/{value} [post]
func NewMetricUpdatePathHandler(updater Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		mType := chi.URLParam(r, "type")
		id := chi.URLParam(r, "name")
		valStr := chi.URLParam(r, "value")

		if strings.TrimSpace(id) == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if mType != models.Gauge && mType != models.Counter {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		var metric models.Metrics
		metric.ID = id
		metric.MType = mType

		switch mType {
		case models.Gauge:
			v, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			metric.Value = &v
		case models.Counter:
			d, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			metric.Delta = &d
		}

		if _, err := updater.Update(ctx, &metric); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// NewMetricGetPathHandler retrieves a metric by type and ID.
//
// @Summary Get metric by type and ID
// @Description Retrieves a metric value or delta as plain text (note: no response body declared in swagger)
// @Tags metrics
// @Accept plain
// @Produce plain
// @Param type path string true "Metric type (gauge or counter)"
// @Param id path string true "Metric ID"
// @Success 200 "OK"
// @Failure 400 "Bad Request"
// @Failure 404 "Not Found"
// @Failure 500 "Internal Server Error"
// @Router /value/{type}/{id} [get]
func NewMetricGetPathHandler(getter Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		mType := chi.URLParam(r, "type")
		id := chi.URLParam(r, "id")

		if strings.TrimSpace(id) == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if mType != models.Gauge && mType != models.Counter {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		metric, err := getter.Get(ctx, &models.MetricID{ID: id, MType: mType})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if metric == nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(strconv.FormatFloat(*metric.Value, 'f', -1, 64)))
		case models.Counter:
			if metric.Delta == nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(strconv.FormatInt(*metric.Delta, 10)))
		default:
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
	}
}

// NewMetricListHTMLHandler lists all metrics.
//
// @Summary List all metrics
// @Description Returns a simple HTML page with all metrics listed in a table
// @Tags metrics
// @Accept plain
// @Produce html
// @Success 200 "OK"
// @Failure 500 "Internal Server Error"
// @Router / [get]
func NewMetricListHTMLHandler(lister Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		metrics, err := lister.List(ctx)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var sb strings.Builder

		sb.WriteString("<html><body><h1>Metrics List</h1>")
		sb.WriteString("<table border='1'><tr><th>Name</th><th>Value</th></tr>")

		for _, m := range metrics {
			val := ""
			if m.Value != nil {
				val = strconv.FormatFloat(*m.Value, 'f', -1, 64)
			} else if m.Delta != nil {
				val = strconv.FormatInt(*m.Delta, 10)
			}

			sb.WriteString("<tr><td>")
			sb.WriteString(m.ID)
			sb.WriteString("</td><td>")
			sb.WriteString(val)
			sb.WriteString("</td></tr>")
		}

		sb.WriteString("</table></body></html>")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(sb.String()))
	}
}

// NewMetricUpdateBodyHandler creates a handler that updates a metric using JSON payload.
//
// @Summary Save or update a metric (JSON)
// @Description Updates a metric value or delta via POST request with JSON body.
// @Tags metrics
// @Accept json
// @Produce json
// @Param metric body models.Metrics true "Metric JSON body"
// @Success 200 {object} models.Metrics "Updated metric returned in response"
// @Failure 400 "Bad Request"
// @Failure 500 "Internal Server Error"
// @Router /update/ [post]
func NewMetricUpdateBodyHandler(updater Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		var metric models.Metrics
		dec := json.NewDecoder(r.Body)
		defer r.Body.Close()

		if err := dec.Decode(&metric); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if metric.ID == "" || (metric.MType != models.Gauge && metric.MType != models.Counter) {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		updatedMetric, err := updater.Update(r.Context(), &metric)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(updatedMetric)
	}
}

// NewMetricGetBodyHandler creates a handler that retrieves a metric by JSON payload.
//
// @Summary Get metric value (JSON)
// @Description Retrieves a metric by ID and type using POST JSON body.
// @Tags metrics
// @Accept json
// @Produce json
// @Param metric body models.Metrics true "Metric request body with ID and MType"
// @Success 200 {object} models.Metrics "Metric returned with current value"
// @Failure 400 "Bad Request"
// @Failure 404 "Not Found"
// @Failure 500 "Internal Server Error"
// @Router /value/ [post]
func NewMetricGetBodyHandler(getter Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		var requestMetric models.Metrics
		dec := json.NewDecoder(r.Body)
		defer r.Body.Close()

		if err := dec.Decode(&requestMetric); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if requestMetric.ID == "" || (requestMetric.MType != models.Gauge && requestMetric.MType != models.Counter) {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		metric, err := getter.Get(r.Context(), &models.MetricID{ID: requestMetric.ID, MType: requestMetric.MType})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if metric == nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(metric)
	}
}
