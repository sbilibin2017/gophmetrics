package http

import (
	"context"
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
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if mType != models.Gauge && mType != models.Counter {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metric models.Metrics
		metric.ID = id
		metric.MType = mType

		switch mType {
		case models.Gauge:
			v, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric.Value = &v
		case models.Counter:
			d, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric.Delta = &d
		}

		if _, err := updater.Update(ctx, &metric); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if mType != models.Gauge && mType != models.Counter {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		metric, err := getter.Get(ctx, &models.MetricID{ID: id, MType: mType})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(strconv.FormatFloat(*metric.Value, 'f', -1, 64)))
		case models.Counter:
			if metric.Delta == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(strconv.FormatInt(*metric.Delta, 10)))
		default:
			w.WriteHeader(http.StatusBadRequest)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body><h1>Metrics List</h1><table border='1'><tr><th>ID</th><th>Type</th><th>Value</th><th>Delta</th></tr>"))

		for _, m := range metrics {
			val := ""
			if m.Value != nil {
				val = strconv.FormatFloat(*m.Value, 'f', -1, 64)
			}
			delta := ""
			if m.Delta != nil {
				delta = strconv.FormatInt(*m.Delta, 10)
			}
			row := "<tr><td>" + m.ID + "</td><td>" + m.MType + "</td><td>" + val + "</td><td>" + delta + "</td></tr>"
			w.Write([]byte(row))
		}

		w.Write([]byte("</table></body></html>"))
	}
}
