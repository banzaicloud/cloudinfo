package logger

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type ctxMarker struct{}

var (
	ctxKey = &ctxMarker{}

	// Logger is root logger for events
	Logger *logrus.Logger
)

const (
	providerKey           = "provider"
	regionKey             = "region"
	serviceKey            = "service"
	correlationIdKey      = "correlation-id"
	scrapeIdCompleteKey   = "scrapeIdComplete"
	scrapeIdShortLivedKey = "scrapeIdShortLived"
)

var loggerKey = []string{providerKey, regionKey, serviceKey, correlationIdKey, scrapeIdCompleteKey, scrapeIdShortLivedKey}

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
func Extract(ctx context.Context) GlobalLogger {
	fds, ok := ctx.Value(ctxKey).(map[string]interface{})
	if !ok || fds == nil {
		return logrus.NewEntry(Logger)
	}

	fields := logrus.Fields{}
	for k, v := range fds {
		for _, key := range loggerKey {
			if k == key {
				fields[k] = v
			}
		}
	}
	return Logger.WithFields(fields)
}

// ToContext sets a logrus logger on the context, which can then obtained by Extract
func ToContext(ctx context.Context, fields ...map[string]interface{}) context.Context {
	fds := make(map[string]interface{})

	for _, field := range fields {
		for k, v := range field {
			fds[k] = v
		}
	}
	return context.WithValue(ctx, ctxKey, fds)
}

// GetCorrelationId get correlation id from gin context
func GetCorrelationId(c *gin.Context) string {
	id := c.GetString(ContextKey)
	return id
}

// Provider get provider value
func Provider(provider string) map[string]interface{} {
	providers := make(map[string]interface{})
	providers[providerKey] = provider
	return providers
}

// Region get region value
func Region(region string) map[string]interface{} {
	regions := make(map[string]interface{})
	regions[regionKey] = region
	return regions
}

// Service get service value
func Service(service string) map[string]interface{} {
	services := make(map[string]interface{})
	services[serviceKey] = service
	return services
}

// ScrapeIdShortLived get scrape id short lived value
func ScrapeIdShortLived(scrapeId uint64) map[string]interface{} {
	scrapeIdShortLived := make(map[string]interface{})
	scrapeIdShortLived[scrapeIdShortLivedKey] = scrapeId
	return scrapeIdShortLived
}

// ScrapeIdComplete get scrape id complete value
func ScrapeIdComplete(scrapeId uint64) map[string]interface{} {
	scrapeIdComplete := make(map[string]interface{})
	scrapeIdComplete[scrapeIdCompleteKey] = scrapeId
	return scrapeIdComplete
}

// CorrelationId get correlation id value
func CorrelationId(correlationId string) map[string]interface{} {
	correlationIds := make(map[string]interface{})
	correlationIds[correlationIdKey] = correlationId
	return correlationIds
}

// LogrusEntry ...
type LogrusEntry struct {
	*logrus.Entry
}

// GlobalLogger ...
type GlobalLogger interface {
	WithError(err error) *logrus.Entry
	WithField(key string, value interface{}) *logrus.Entry
	WithFields(fields logrus.Fields) *logrus.Entry
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Fatal(args ...interface{})
}
