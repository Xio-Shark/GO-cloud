package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go-cloud/internal/metrics"
	"go-cloud/internal/notifier"
	"go-cloud/internal/queue"
	"go-cloud/internal/repository"
	"go-cloud/pkg/traceutil"
)

type NotifierService interface {
	ConsumeLoop(ctx context.Context) error
	HandleOneMessage(ctx context.Context, raw []byte) error
}

type notifierService struct {
	queueRepo   repository.QueueRepository
	notifier    notifier.Notifier
	pollTimeout time.Duration
}

func NewNotifierService(queueRepo repository.QueueRepository, sender notifier.Notifier, pollTimeout time.Duration) NotifierService {
	if pollTimeout <= 0 {
		pollTimeout = 5 * time.Second
	}
	return &notifierService{
		queueRepo:   queueRepo,
		notifier:    sender,
		pollTimeout: pollTimeout,
	}
}

func (s *notifierService) ConsumeLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		raw, err := s.queueRepo.DequeueNotification(ctx, s.pollTimeout)
		if err != nil {
			return err
		}
		if len(raw) == 0 {
			continue
		}
		messageCtx := traceutil.WithTraceID(ctx, traceutil.NewTraceID())
		if err := s.HandleOneMessage(messageCtx, raw); err != nil {
			slog.Default().ErrorContext(messageCtx, "notifier handle message failed", "trace_id", traceutil.FromContext(messageCtx), "error", err)
		}
	}
}

func (s *notifierService) HandleOneMessage(ctx context.Context, raw []byte) error {
	start := time.Now()
	message := queue.NotificationMessage{}
	if err := json.Unmarshal(raw, &message); err != nil {
		metrics.NotifierFailedTotal.Inc()
		return err
	}
	if message.TraceID != "" {
		ctx = traceutil.WithTraceID(ctx, message.TraceID)
	}
	if err := s.notifier.Send(ctx, message); err != nil {
		metrics.NotifierFailedTotal.Inc()
		return err
	}
	metrics.NotifierSentTotal.Inc()
	metrics.NotifierRequestDurationMs.Observe(float64(time.Since(start).Milliseconds()))
	return nil
}
