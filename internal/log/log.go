package log

import (
	"context"

	"github.com/containerd/containerd/log"
	"github.com/sirupsen/logrus"
)

func WithLogContext(ctx context.Context) context.Context {
	logger := logrus.New()
	logger.SetLevel(logrus.GetLevel())
	return log.WithLogger(ctx, logrus.NewEntry(logger))
}
