package logger

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type ctxLoggerMarker struct{}

var (
	ctxLoggerKey = &ctxLoggerMarker{}

	// Logger is root logger for events
	Logger *logrus.Logger
)

// NewLogger sets level and format for Logger
func NewLogger() *logrus.Logger {
	logger := newLogger(Config{
		Level:  viper.GetString("log-level"),
		Format: viper.GetString("log-format"),
	})

	return logger
}

// Config holds information necessary for customizing the logger.
type Config struct {
	Level  string
	Format string
}

func newLogger(config Config) *logrus.Logger {
	logger := logrus.New()

	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}

	logger.Level = level

	switch config.Format {
	case "json":
		logger.Formatter = new(logrus.JSONFormatter)

	default:
		textFormatter := new(logrus.TextFormatter)
		textFormatter.FullTimestamp = true

		logger.Formatter = textFormatter
	}

	return logger
}

// Extract takes the logrus.Entry from the context
func Extract(ctx context.Context) *logrus.Entry {
	l, ok := ctx.Value(ctxLoggerKey).(*logrus.Entry)
	if !ok || l == nil {
		return logrus.NewEntry(Logger)
	}

	fields := logrus.Fields{}
	for k, v := range l.Data {
		fields[k] = v
	}
	return Logger.WithFields(fields)
}

// ToContext sets a logrus logger on the context, which can then obtained by Extract
func ToContext(ctx context.Context, entry *logrus.Entry) context.Context {
	return context.WithValue(ctx, ctxLoggerKey, entry)
}

// GetCorrelationId get correlation id from gin context
func GetCorrelationId(c *gin.Context) string {
	id := c.GetString(ContextKey)
	return id
}
