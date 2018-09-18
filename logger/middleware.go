package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"time"
)

// Middleware returns a gin compatible handler.
func Middleware(logger GlobalLogger, notlogged ...string) gin.HandlerFunc {
	var skip map[string]struct{}

	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// start timer
		start := time.Now()

		// prevent middlewares from faking the request path
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			end := time.Now()
			latency := end.Sub(start)

			if raw != "" {
				path = path + "?" + raw
			}

			fields := logrus.Fields{
				"status":  c.Writer.Status(),
				"method":  c.Request.Method,
				"path":    path,
				"latency": latency,
			}

			if cid := c.GetString(ContextKey); cid != "" {
				fields["correlation-id"] = cid
			}

			entry := logger.WithFields(fields)

			if len(c.Errors) > 0 {
				// Append error field if this is an erroneous request.
				entry.Error(c.Errors.String())
			} else {
				entry.Info()
			}
		}
	}
}

// ContextKey is the key the retrieved (or generated) correlation ID is stored under in the gin Context.
const ContextKey = "correlationid"

// Default correlation ID header
const defaultHeader = "Correlation-ID"

// MiddlewareCorrelationId returns a gin compatible handler.
func MiddlewareCorrelationId() gin.HandlerFunc {
	m := new(middleware)

	if m.header == "" {
		m.header = defaultHeader
	}

	return m.Handle
}

type middleware struct {
	header string
}

func (m *middleware) Handle(ctx *gin.Context) {
	if header := ctx.GetHeader(m.header); header != "" {
		ctx.Set(ContextKey, header)
	} else {
		ctx.Set(ContextKey, uuid.NewV4().String())
	}

	ctx.Next()
}
