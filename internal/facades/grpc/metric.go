package grpc

import (
	"context"

	"github.com/sbilibin2017/gophmetrics/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/sbilibin2017/gophmetrics/pkg/grpc"
)

// MetricGRPCFacade provides a gRPC client facade for metric operations.
// It wraps the generated MetricWriteServiceClient and exposes
// high-level methods to update metrics via gRPC.
type MetricGRPCFacade struct {
	client pb.MetricWriteServiceClient
}

// NewMetricGRPCFacade creates a new MetricGRPCFacade instance
// given a grpc.ClientConn. It initializes the internal
// MetricWriteServiceClient.
func NewMetricGRPCFacade(
	client pb.MetricWriteServiceClient,
) *MetricGRPCFacade {
	return &MetricGRPCFacade{
		client: client,
	}
}

// Update sends multiple metrics to the gRPC MetricWriteService.Update method.
// For each metric in the slice, it converts the internal models.Metrics representation
// to the protobuf message format, wrapping optional fields appropriately,
// and invokes the Update RPC.
//
// Returns an error if any of the RPC calls fail.
func (f *MetricGRPCFacade) Update(ctx context.Context, metrics []*models.Metrics) error {
	for _, metric := range metrics {
		var delta *wrapperspb.Int64Value
		if metric.Delta != nil {
			delta = wrapperspb.Int64(*metric.Delta)
		}

		var value *wrapperspb.DoubleValue
		if metric.Value != nil {
			value = wrapperspb.Double(*metric.Value)
		}

		req := &pb.UpdateMetricRequest{
			Metric: &pb.Metrics{
				Id:        metric.ID,
				Mtype:     metric.MType,
				Delta:     delta,
				Value:     value,
				CreatedAt: timestamppb.New(metric.CreatedAt),
				UpdatedAt: timestamppb.New(metric.UpdatedAt),
			},
		}

		_, err := f.client.Update(ctx, req)
		if err != nil {
			return err
		}
	}

	return nil
}
