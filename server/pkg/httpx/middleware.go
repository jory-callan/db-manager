package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"czwlinux.cloud/go-friday-starter/pkg/httpx/response"
	"czwlinux.cloud/go-friday-starter/pkg/idutil"
	"czwlinux.cloud/go-friday-starter/pkg/logger"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func Recover(log *logger.Logger) echo.MiddlewareFunc {
	return echoMiddleware.RecoverWithConfig(echoMiddleware.RecoverConfig{
		StackSize: 2 << 10,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			if log != nil {
				log.Error("http panic", zap.Error(err), zap.ByteString("stack", stack))
			}
			return response.InternalError(c, "unknown error")
		},
	})
}

func Logger(log *logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			id := req.Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = res.Header().Get(echo.HeaderXRequestID)
			}
			if id == "" {
				id = idutil.ShortUUIDv7()
			}
			res.Header().Set(echo.HeaderXRequestID, id)

			status := res.Status
			if err != nil {
				var he *echo.HTTPError
				if errors.As(err, &he) {
					status = he.Code
				}
				if status == 0 {
					status = http.StatusInternalServerError
				}
			}

			fields := []zap.Field{
				zap.String("request_id", id),
				zap.String("remote_ip", c.RealIP()),
				zap.String("host", req.Host),
				zap.String("proto", req.Proto),
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.String("path", req.URL.Path),
				zap.String("route", c.Path()),
				zap.String("user_agent", req.UserAgent()),
				zap.String("referer", req.Referer()),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.Int64("bytes_in", req.ContentLength),
				zap.Int64("bytes_out", res.Size),
			}
			if qs := req.URL.RawQuery; qs != "" {
				fields = append(fields, zap.String("query_string", qs))
			}
			if contentType := req.Header.Get("Content-Type"); contentType != "" {
				fields = append(fields, zap.String("content_type", contentType))
			}
			if err != nil {
				fields = append(fields, zap.Error(err))
			}

			if log != nil {
				switch {
				case status >= 500:
					log.Error("http request", fields...)
				case status >= 400:
					log.Warn("http request", fields...)
				default:
					log.Info("http request", fields...)
				}
			}

			return err
		}
	}
}

func ErrorHandler(log *logger.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed || c.Request().Method == http.MethodHead {
			return
		}

		var he *echo.HTTPError
		if errors.As(err, &he) {
			msg := fmt.Sprint(he.Message)
			_ = response.Error(c, he.Code, msg)
			return
		}

		if errors.Is(err, context.Canceled) {
			_ = response.Error(c, 499, "client closed request")
			return
		}

		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		if log != nil {
			log.Error("http internal error", zap.String("request_id", requestID), zap.Error(err))
		}

		// 直接用 errors.New("xxx") 的文本
		_ = response.InternalError(c, "internal error")
	}
}

func CORS(cfg CORSConfig) echo.MiddlewareFunc {
	if len(cfg.AllowOrigins) == 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	corsCfg := echoMiddleware.CORSConfig{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		MaxAge:           cfg.MaxAge,
		AllowCredentials: cfg.AllowCredentials,
	}
	if slices.Contains(cfg.AllowOrigins, "*") {
		corsCfg.AllowOriginFunc = func(origin string) (bool, error) {
			return true, nil
		}
	}
	return echoMiddleware.CORSWithConfig(corsCfg)
}

func RateLimit(cfg RateLimitConfig) echo.MiddlewareFunc {
	if !cfg.Enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	storeCfg := echoMiddleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(cfg.RequestsPerSecond),
		Burst:     cfg.Burst,
		ExpiresIn: 3 * time.Minute,
	}
	return echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStoreWithConfig(storeCfg))
}
