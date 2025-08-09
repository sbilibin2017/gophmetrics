package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sbilibin2017/gophmetrics/internal/models"
	pb "github.com/sbilibin2017/gophmetrics/pkg/grpc"
	"google.golang.org/grpc"
)

// mockMetricWriteClient mocks pb.MetricWriteServiceClient for tests.
type mockMetricWriteClient struct {
	ReceivedRequests []*pb.UpdateMetricRequest
	UpdateErr        error
}

// Update implements MetricWriteServiceClient.Update
func (m *mockMetricWriteClient) Update(ctx context.Context, in *pb.UpdateMetricRequest, opts ...grpc.CallOption) (*pb.UpdateMetricResponse, error) {
	m.ReceivedRequests = append(m.ReceivedRequests, in)
	return &pb.UpdateMetricResponse{
		Metric: in.Metric,
	}, m.UpdateErr
}

// required to implement interface, even if empty
func (m *mockMetricWriteClient) mustEmbedUnimplementedMetricWriteServiceClient() {}

// TestMetricGRPCFacade_Update_Success tests successful Update calls.
func TestMetricGRPCFacade_Update_Success(t *testing.T) {
	mockClient := &mockMetricWriteClient{}

	// Use the new constructor that accepts pb.MetricWriteServiceClient interface
	facade := NewMetricGRPCFacade(mockClient)

	now := time.Now()
	deltaVal := int64(42)
	valueVal := 3.14

	metrics := []*models.Metrics{
		{
			ID:        "metric1",
			MType:     "counter",
			Delta:     &deltaVal,
			Value:     nil,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "metric2",
			MType:     "gauge",
			Delta:     nil,
			Value:     &valueVal,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := facade.Update(context.Background(), metrics)
	require.NoError(t, err)
	assert.Len(t, mockClient.ReceivedRequests, 2)

	req1 := mockClient.ReceivedRequests[0]
	assert.Equal(t, "metric1", req1.Metric.Id)
	assert.Equal(t, "counter", req1.Metric.Mtype)
	assert.NotNil(t, req1.Metric.Delta)
	assert.Nil(t, req1.Metric.Value)
	assert.Equal(t, deltaVal, req1.Metric.Delta.Value)

	req2 := mockClient.ReceivedRequests[1]
	assert.Equal(t, "metric2", req2.Metric.Id)
	assert.Equal(t, "gauge", req2.Metric.Mtype)
	assert.Nil(t, req2.Metric.Delta)
	assert.NotNil(t, req2.Metric.Value)
	assert.InEpsilon(t, valueVal, req2.Metric.Value.Value, 0.0001)
}

// TestMetricGRPCFacade_Update_Error tests error returned from client.
func TestMetricGRPCFacade_Update_Error(t *testing.T) {
	mockClient := &mockMetricWriteClient{
		UpdateErr: errors.New("rpc error"),
	}

	facade := NewMetricGRPCFacade(mockClient)

	metrics := []*models.Metrics{
		{
			ID:    "metric1",
			MType: "counter",
			Delta: nil,
			Value: nil,
		},
	}

	err := facade.Update(context.Background(), metrics)
	require.Error(t, err)
	assert.EqualError(t, err, "rpc error")
}
