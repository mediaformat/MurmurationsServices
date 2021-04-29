package logger

import (
	"fmt"
	"time"

	"github.com/MurmurationsNetwork/MurmurationsServices/common/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var defaultSkipPaths = []string{"/ping", "/nodes/_search"}

func NewLogger() gin.HandlerFunc {
	return LoggerWithConfig(gin.LoggerConfig{})
}

func LoggerWithConfig(conf gin.LoggerConfig) gin.HandlerFunc {
	notlogged := conf.SkipPaths
	notlogged = append(notlogged, defaultSkipPaths...)

	var skip map[string]struct{}

	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		fmt.Println("=========================")
		fmt.Printf("c.ClientIP(): %v \n", c.ClientIP())
		fmt.Printf("c.Request.Header: %+v \n", c.Request.Header)
		fmt.Println("=========================")

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if _, ok := skip[path]; !ok {
			param := gin.LogFormatterParams{
				Request: c.Request,
				Keys:    c.Keys,
			}

			param.TimeStamp = time.Now()
			param.Latency = param.TimeStamp.Sub(start)

			param.ClientIP = c.ClientIP()
			param.Method = c.Request.Method
			param.StatusCode = c.Writer.Status()

			param.BodySize = c.Writer.Size()

			if raw != "" {
				path = path + "?" + raw
			}

			param.Path = path

			// Get the user geographic information.
			geoInfo := getGeoInfo(param.ClientIP)

			logger.Info(
				"Log Entry",
				zap.Int("status", param.StatusCode),
				zap.String("latency", fmt.Sprintf("%v", param.Latency)),
				zap.String("method", param.Method),
				zap.String("path", param.Path),
				zap.String("ip", param.ClientIP),
				zap.String("city", geoInfo.City),
				zap.String("country", geoInfo.Country),
				zap.Float64("lat", geoInfo.Lat),
				zap.Float64("lon", geoInfo.Lon),
			)
		}
	}
}
