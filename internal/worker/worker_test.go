package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMetricWorker_Start_RestoreAndShutdown_SaveAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Создаём моки
	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	// Пример метрик
	mockMetrics := []*models.Metrics{
		{ID: "Alloc", MType: "gauge", Value: ptrFloat64(123.4)},
		{ID: "PollCount", MType: "counter", Delta: ptrInt64(42)},
	}

	// Ожидания при restore = true
	fileReader.EXPECT().List(gomock.Any()).Return(mockMetrics, nil)
	currentWriter.EXPECT().Save(gomock.Any(), mockMetrics[0]).Return(nil)
	currentWriter.EXPECT().Save(gomock.Any(), mockMetrics[1]).Return(nil)

	// Ожидания на shutdown (List + Save)
	currentReader.EXPECT().List(gomock.Any()).Return(mockMetrics, nil)
	fileWriter.EXPECT().Save(gomock.Any(), mockMetrics[0]).Return(nil)
	fileWriter.EXPECT().Save(gomock.Any(), mockMetrics[1]).Return(nil)

	// Инициализируем воркер
	mw := NewMetricWorker(
		true, // restore
		nil,  // no ticker — только при завершении
		currentReader,
		currentWriter,
		fileReader,
		fileWriter,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)

	// Стартуем worker в горутине
	go func() {
		done <- mw.Start(ctx)
	}()

	// Завершаем контекст через 100мс
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Проверяем, что ошибок не было
	err := <-done
	assert.NoError(t, err)
}

func TestMetricWorker_Start_PeriodicStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	mockMetrics := []*models.Metrics{
		{ID: "Heap", MType: "gauge", Value: ptrFloat64(99.9)},
	}

	// AnyTimes — потому что тикер может сработать 2-3 раза
	currentReader.EXPECT().List(gomock.Any()).Return(mockMetrics, nil).AnyTimes()
	fileWriter.EXPECT().Save(gomock.Any(), mockMetrics[0]).Return(nil).AnyTimes()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	mw := NewMetricWorker(
		false,
		ticker,
		currentReader,
		currentWriter,
		fileReader,
		fileWriter,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := mw.Start(ctx)
	assert.NoError(t, err)
}

func TestMetricWorker_Start_RestoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	fileReader.EXPECT().List(gomock.Any()).Return(nil, errors.New("file read error"))

	mw := NewMetricWorker(
		true, // restore
		nil,
		currentReader,
		currentWriter,
		fileReader,
		fileWriter,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := mw.Start(ctx)
	assert.EqualError(t, err, "file read error")
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}
