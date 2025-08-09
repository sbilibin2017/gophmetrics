package grpc

import (
	"context"
	"strings"

	"github.com/sbilibin2017/gophmetrics/internal/models"
	pb "github.com/sbilibin2017/gophmetrics/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

// MetricWriteHandler implements pb.MetricWriteServiceServer interface using Updater.
type MetricWriteHandler struct {
	Updater Updater
	pb.UnimplementedMetricWriteServiceServer
}

// NewMetricWriteHandler creates a new MetricWriteHandler with the given Updater.
func NewMetricWriteHandler(updater Updater) *MetricWriteHandler {
	return &MetricWriteHandler{
		Updater: updater,
	}
}

// Update updates a single metric.
func (s *MetricWriteHandler) Update(ctx context.Context, req *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	m := req.GetMetric()
	if m == nil {
		return nil, status.Errorf(codes.InvalidArgument, "metric is required")
	}

	// Validation: non-empty ID and valid metric type
	if strings.TrimSpace(m.Id) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "metric id is required")
	}
	if m.Mtype != models.Gauge && m.Mtype != models.Counter {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metric type")
	}

	metric := &models.Metrics{
		ID:    m.Id,
		MType: m.Mtype,
	}
	if m.Delta != nil {
		val := m.Delta.GetValue()
		metric.Delta = &val
	}
	if m.Value != nil {
		val := m.Value.GetValue()
		metric.Value = &val
	}
	if m.CreatedAt != nil {
		metric.CreatedAt = m.CreatedAt.AsTime()
	}
	if m.UpdatedAt != nil {
		metric.UpdatedAt = m.UpdatedAt.AsTime()
	}

	updated, err := s.Updater.Update(ctx, metric)
	if err != nil {
		return nil, err
	}

	resp := &pb.UpdateMetricResponse{
		Metric: &pb.Metrics{
			Id:        updated.ID,
			Mtype:     updated.MType,
			CreatedAt: timestamppb.New(updated.CreatedAt),
			UpdatedAt: timestamppb.New(updated.UpdatedAt),
		},
	}

	if updated.Delta != nil {
		resp.Metric.Delta = wrapperspb.Int64(*updated.Delta)
	}
	if updated.Value != nil {
		resp.Metric.Value = wrapperspb.Double(*updated.Value)
	}

	return resp, nil
}

// MetricReadHandler implements pb.MetricReadServiceServer interface using Getter and Lister.
type MetricReadHandler struct {
	Getter Getter
	Lister Lister
	pb.UnimplementedMetricReadServiceServer
}

// NewMetricReadHandler creates a new MetricReadHandler with the given Getter and Lister.
func NewMetricReadHandler(getter Getter, lister Lister) *MetricReadHandler {
	return &MetricReadHandler{
		Getter: getter,
		Lister: lister,
	}
}

// Get returns a metric by id.
func (s *MetricReadHandler) Get(ctx context.Context, req *pb.GetMetricRequest) (*pb.Metrics, error) {
	id := req.GetId()
	if id == nil {
		return nil, status.Errorf(codes.InvalidArgument, "metric id is required")
	}

	// Validation: non-empty ID and valid metric type
	if strings.TrimSpace(id.Id) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "metric id is required")
	}
	if id.Mtype != models.Gauge && id.Mtype != models.Counter {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metric type")
	}

	metricID := &models.MetricID{
		ID:    id.Id,
		MType: id.Mtype,
	}

	metric, err := s.Getter.Get(ctx, metricID)
	if err != nil {
		return nil, err
	}
	if metric == nil {
		return nil, status.Errorf(codes.NotFound, "metric not found")
	}

	resp := &pb.Metrics{
		Id:        metric.ID,
		Mtype:     metric.MType,
		CreatedAt: timestamppb.New(metric.CreatedAt),
		UpdatedAt: timestamppb.New(metric.UpdatedAt),
	}

	if metric.Delta != nil {
		resp.Delta = wrapperspb.Int64(*metric.Delta)
	}
	if metric.Value != nil {
		resp.Value = wrapperspb.Double(*metric.Value)
	}

	return resp, nil
}

// List returns all metrics.
func (s *MetricReadHandler) List(ctx context.Context, _ *emptypb.Empty) (*pb.ListMetricsResponse, error) {
	metrics, err := s.Lister.List(ctx)
	if err != nil {
		return nil, err
	}

	resp := &pb.ListMetricsResponse{}
	for _, m := range metrics {
		pbMetric := &pb.Metrics{
			Id:        m.ID,
			Mtype:     m.MType,
			CreatedAt: timestamppb.New(m.CreatedAt),
			UpdatedAt: timestamppb.New(m.UpdatedAt),
		}
		if m.Delta != nil {
			pbMetric.Delta = wrapperspb.Int64(*m.Delta)
		}
		if m.Value != nil {
			pbMetric.Value = wrapperspb.Double(*m.Value)
		}
		resp.Metrics = append(resp.Metrics, pbMetric)
	}

	return resp, nil
}
