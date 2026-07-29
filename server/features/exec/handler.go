package exec

import (
	"errors"

	"czwlinux.cloud/go-friday-starter/global"
	"czwlinux.cloud/go-friday-starter/pkg/httpx"
	"czwlinux.cloud/go-friday-starter/pkg/httpx/response"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// mapErrToHTTP maps service-layer sentinel errors to HTTP status + message.
// Single source of truth: any new sentinel error in errors.go must be classified here.
//
// Categories:
//   - 400: client / input errors
//   - 403: business rejection (write rejected, dangerous, etc.)
//   - 404: resource not found
//   - 500: any unclassified error
func mapErrToHTTP(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	switch {
	// 400 — invalid request
	case errors.Is(err, ErrInvalidParam),
		errors.Is(err, ErrDatabaseNotAllowed),
		errors.Is(err, ErrUnsupportedType),
		errors.Is(err, ErrQueryFailed),
		errors.Is(err, ErrScanRowsFailed),
		errors.Is(err, ErrRedisCommandFailed),
		errors.Is(err, ErrMongoCommandFailed),
		errors.Is(err, ErrMongoDecodeFailed),
		errors.Is(err, ErrESRequestFailed),
		errors.Is(err, ErrSelectDBFailed),
		errors.Is(err, ErrGetDBConnection),
		errors.Is(err, ErrExecutionFailed),
		errors.Is(err, ErrClassifyFailed),
		errors.Is(err, ErrCheckEscalation),
		errors.Is(err, ErrRedisEscalatedNotSup):
		return response.BadRequest(c, err.Error())

	// 404 — resource not found
	case errors.Is(err, ErrProjectNotFound),
		errors.Is(err, ErrDatasourceNotFound):
		return response.NotFound(c, err.Error())

	// 403 — business rejection
	case errors.Is(err, ErrNoProjectAccess),
		errors.Is(err, ErrDangerous),
		errors.Is(err, ErrUnknown),
		errors.Is(err, ErrWriteRejected),
		errors.Is(err, ErrNoActiveEscalation):
		return response.Forbidden(c, err.Error())
	}

	// Unclassified — log with context, return 500
	requestID := ""
	if c.Response() != nil {
		requestID = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	global.Log.Warn("exec: unclassified error",
		zap.String("request_id", requestID),
		zap.Error(err))
	return response.InternalError(c, "internal error")
}

func (h *Handler) Execute(c echo.Context) error {
	var req ExecuteRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid param")
	}

	userID := httpx.CurrentUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "unauthorized")
	}

	result, err := Execute(c.Request().Context(), userID, req)
	if err != nil {
		return mapErrToHTTP(c, err)
	}

	return response.Ok(c, result)
}

func (h *Handler) ExecuteEscalated(c echo.Context) error {
	var req ExecuteRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid param")
	}

	userID := httpx.CurrentUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "unauthorized")
	}

	result, err := ExecuteEscalated(c.Request().Context(), userID, req)
	if err != nil {
		return mapErrToHTTP(c, err)
	}

	return response.Ok(c, result)
}
