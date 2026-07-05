package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// InstrumentMiddleware records HTTP request metrics for every request.
// Should be registered on the root router.
func InstrumentMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		HTTPRequestsInFlight.Inc()
		start := time.Now()

		c.Next()

		elapsed := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath() // e.g. "/api/v1/data/:id"

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(elapsed.Seconds())
		HTTPRequestsInFlight.Dec()
	}
}
