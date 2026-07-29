package exec

import "errors"

// Sentinel errors for the exec package.
// Service layer returns these (wrapped with %w) so the handler can use
// errors.Is to map to HTTP status codes without relying on string matching.
//
// Naming convention:
//   - Err*               — business-meaningful errors
//   - HTTP mapping       — handler is the single place that decides HTTP status
//
// See handler.go for the error → HTTP mapping table.
var (
	// Request-level errors (HTTP 4xx)
	ErrInvalidParam         = errors.New("invalid param")
	ErrProjectNotFound      = errors.New("project not found")
	ErrDatasourceNotFound   = errors.New("datasource not found")
	ErrNoProjectAccess      = errors.New("forbidden: no project access")
	ErrDatabaseNotAllowed   = errors.New("database not in project scope")
	ErrUnsupportedType      = errors.New("unsupported datasource type")
	ErrRedisEscalatedNotSup = errors.New("redis escalated execution not yet supported")

	// Pipeline / classification errors (HTTP 403)
	ErrClassifyFailed     = errors.New("classify failed")
	ErrDangerous          = errors.New("dangerous operation rejected")
	ErrUnknown            = errors.New("unrecognizable statement")
	ErrWriteRejected      = errors.New("write operation rejected: submit ticket or request escalation")
	ErrNoActiveEscalation = errors.New("escalated execution failed: no active escalation")

	// Execution errors (HTTP 400 — user input / DB-side problem, NOT a server error)
	ErrGetDBConnection    = errors.New("get db connection failed")
	ErrSelectDBFailed     = errors.New("select database failed")
	ErrQueryFailed        = errors.New("query failed")
	ErrScanRowsFailed     = errors.New("scan rows failed")
	ErrRedisCommandFailed = errors.New("redis command failed")
	ErrMongoCommandFailed = errors.New("mongo command failed")
	ErrMongoDecodeFailed  = errors.New("mongo decode result failed")
	ErrESRequestFailed    = errors.New("es request failed")
	ErrExecutionFailed    = errors.New("execution failed")
	ErrCheckEscalation    = errors.New("check escalation failed")
)
