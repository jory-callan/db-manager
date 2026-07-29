package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"czwlinux.cloud/go-friday-starter/features/audit"
	"czwlinux.cloud/go-friday-starter/features/auth"
	"czwlinux.cloud/go-friday-starter/features/datasource"
	dsrule "czwlinux.cloud/go-friday-starter/features/dsrule"
	"czwlinux.cloud/go-friday-starter/features/escalation"
	"czwlinux.cloud/go-friday-starter/features/project"
	"czwlinux.cloud/go-friday-starter/global"
	"czwlinux.cloud/go-friday-starter/pkg/dbpool"
	"czwlinux.cloud/go-friday-starter/pkg/driver"
	"czwlinux.cloud/go-friday-starter/pkg/pipeline"
	"czwlinux.cloud/go-friday-starter/pkg/pipeline/parser"
)

const (
	defaultQueryTimeout = 30 * time.Second
	maxResultRows       = 10000
)

// newPipeline 构建统一 Pipeline，注册所有可用 Parser
func newPipeline() *pipeline.Pipeline {
	pipe := pipeline.NewPipeline(dsrule.NewRuleEngine())
	pipe.RegisterParser(parser.NewSQLParser())
	pipe.RegisterParser(parser.NewRedisParser())
	pipe.RegisterParser(parser.NewMongoParser())
	pipe.RegisterParser(parser.NewESParser())
	return pipe
}

func Execute(ctx context.Context, userID string, req ExecuteRequest) (any, error) {
	if req.ProjectID == "" || req.Sql == "" {
		return nil, fmt.Errorf("%w: project_id and sql are required", ErrInvalidParam)
	}

	sqlToExecute := req.Sql
	if req.SelectedText != "" {
		sqlToExecute = req.SelectedText
	}

	proj, err := project.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrProjectNotFound, req.ProjectID)
	}

	codes, _ := auth.GetUserPermissionCodes(ctx, userID)
	if !project.HasProjectAccess(ctx, userID, req.ProjectID, codes) {
		return nil, ErrNoProjectAccess
	}

	projDatabases := project.SplitDatabases(proj.Scope)
	dbName := req.Database
	if dbName != "" {
		if !project.IsDatabaseAllowed(projDatabases, dbName) {
			return nil, fmt.Errorf("%w: %s", ErrDatabaseNotAllowed, dbName)
		}
	} else if len(projDatabases) == 1 && !project.HasWildcard(projDatabases) {
		dbName = projDatabases[0]
	}

	ds, err := datasource.GetByID(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDatasourceNotFound, proj.DatasourceID)
	}

	// Pipeline classify
	pipe := newPipeline()
	classifyResult, err := pipe.Classify(ctx, ds.Type, ds.TypeGroup, req.Sql, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

	sum := classifyResult.Summary

	// Reject dangerous
	if sum.HasDangerous {
		auditLog(ctx, userID, proj, ds, "execute", "dangerous", audit.StatusRejected, sqlToExecute, "dangerous operation rejected", 0, classifyResult)
		return nil, ErrDangerous
	}

	// Reject unknown
	if sum.HasUnknown {
		auditLog(ctx, userID, proj, ds, "execute", "unknown", audit.StatusRejected, sqlToExecute, "unrecognizable statement", 0, classifyResult)
		return nil, ErrUnknown
	}

	// Check write — reject if not escalated
	if sum.HasWrite {
		auditLog(ctx, userID, proj, ds, "execute", "write", audit.StatusRejected, sqlToExecute, "write operation rejected", 0, classifyResult)
		return nil, fmt.Errorf("%w: 请使用「提交工单」或「提权执行」按钮", ErrWriteRejected)
	}

	// All read — execute based on type group
	switch ds.TypeGroup {
	case "nosql":
		return executeNoSQLRead(ctx, userID, proj, ds, sqlToExecute, classifyResult)
	case "search":
		return executeSearchRead(ctx, userID, proj, ds, sqlToExecute, classifyResult)
	default:
		return executeSQLRead(ctx, userID, proj, ds, dbName, sqlToExecute, classifyResult)
	}
}

func executeSQLRead(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, dbName, sqlToExecute string, classifyResult *pipeline.ClassifyResult) (any, error) {
	pool, err := dbpool.Get(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetDBConnection, err)
	}

	execCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	sqlConn, err := driver.GetSQLConnector(ds.Type)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, ds.Type)
	}
	if dbName != "" {
		if err := sqlConn.SetDatabase(execCtx, pool, dbName); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrSelectDBFailed, dbName, err)
		}
	}

	start := time.Now()
	var allResults []QueryResult
	totalRows := 0

	for _, inst := range classifyResult.Instructions {
		rows, err := pool.QueryContext(execCtx, inst.Raw)
		if err != nil {
			duration := int(time.Since(start).Milliseconds())
			exec := &SqlExecution{
				ProjectID:      proj.ID,
				Sql:            sqlToExecute,
				Classification: string(inst.OpType),
				Status:         StatusFailed,
				DurationMs:     duration,
			}
			_ = CreateExecution(ctx, exec)
			auditLog(ctx, userID, proj, ds, "execute", "read", audit.StatusFailed, sqlToExecute, err.Error(), duration, classifyResult)
			return nil, fmt.Errorf("%w: %w", ErrQueryFailed, err)
		}

		cols, resultRows, scanErr := driver.ScanRows(rows, maxResultRows-totalRows)
		if scanErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrScanRowsFailed, scanErr)
		}

		allResults = append(allResults, QueryResult{
			Columns: cols,
			Rows:    resultRows,
			Total:   len(resultRows),
		})
		totalRows += len(resultRows)
	}

	duration := int(time.Since(start).Milliseconds())
	exec := &SqlExecution{
		ProjectID:      proj.ID,
		Sql:            sqlToExecute,
		Classification: "read",
		Status:         StatusSuccess,
		RowCount:       totalRows,
		DurationMs:     duration,
	}
	_ = CreateExecution(ctx, exec)
	auditLog(ctx, userID, proj, ds, "execute", "read", audit.StatusSuccess, sqlToExecute, "", duration, classifyResult)

	return map[string]interface{}{
		"mode":      "direct",
		"execution": ToDTO(exec),
		"results":   allResults,
	}, nil
}

func executeRedisRead(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, sqlToExecute string, classifyResult *pipeline.ClassifyResult) (any, error) {
	client, err := dbpool.GetRedis(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetDBConnection, err)
	}

	execCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	start := time.Now()
	var allResults []QueryResult

	for _, inst := range classifyResult.Instructions {
		args := make([]any, 0, len(inst.Args)+1)
		args = append(args, inst.Command)
		for _, a := range inst.Args {
			args = append(args, a)
		}

		cmd := client.Do(execCtx, args...)
		if cmd.Err() != nil {
			duration := int(time.Since(start).Milliseconds())
			auditLog(ctx, userID, proj, ds, "execute", "redis", audit.StatusFailed, sqlToExecute, cmd.Err().Error(), duration, classifyResult)
			return nil, fmt.Errorf("%w: %w", ErrRedisCommandFailed, cmd.Err())
		}

		val := cmd.Val()
		var rows [][]any
		var columns []driver.ColumnInfo

		switch v := val.(type) {
		case []any:
			columns = []driver.ColumnInfo{{Name: "result"}}
			for _, item := range v {
				rows = append(rows, []any{fmt.Sprintf("%v", item)})
			}
		case map[any]any:
			columns = []driver.ColumnInfo{{Name: "field"}, {Name: "value"}}
			for k, vv := range v {
				rows = append(rows, []any{fmt.Sprintf("%v", k), fmt.Sprintf("%v", vv)})
			}
		case string, int64, float64, bool:
			columns = []driver.ColumnInfo{{Name: "result"}}
			rows = append(rows, []any{fmt.Sprintf("%v", v)})
		case nil:
			columns = []driver.ColumnInfo{{Name: "result"}}
			rows = append(rows, []any{"(nil)"})
		default:
			columns = []driver.ColumnInfo{{Name: "result"}}
			rows = append(rows, []any{fmt.Sprintf("%v", v)})
		}

		allResults = append(allResults, QueryResult{
			Columns: columns,
			Rows:    rows,
			Total:   len(rows),
		})
	}

	duration := int(time.Since(start).Milliseconds())
	auditLog(ctx, userID, proj, ds, "execute", "redis", audit.StatusSuccess, sqlToExecute, "", duration, classifyResult)

	return map[string]interface{}{
		"mode":      "direct",
		"execution": nil,
		"results":   allResults,
	}, nil
}

// executeNoSQLRead 按 ds.Type 分发 NoSQL 读执行
func executeNoSQLRead(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, sqlToExecute string, classifyResult *pipeline.ClassifyResult) (any, error) {
	switch ds.Type {
	case "redis":
		return executeRedisRead(ctx, userID, proj, ds, sqlToExecute, classifyResult)
	case "mongo":
		return executeMongoRead(ctx, userID, proj, ds, sqlToExecute, classifyResult)
	default:
		return nil, fmt.Errorf("%w (nosql): %s", ErrUnsupportedType, ds.Type)
	}
}

func executeMongoRead(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, sqlToExecute string, classifyResult *pipeline.ClassifyResult) (any, error) {
	client, err := dbpool.GetMongo(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetDBConnection, err)
	}

	execCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	start := time.Now()
	var allResults []QueryResult

	for _, inst := range classifyResult.Instructions {
		// Execute the command as a MongoDB command on the admin DB
		// If a database is specified, use it; otherwise use admin
		dbName := "admin"
		if ds.Database != "" {
			dbName = ds.Database
		}
		if len(inst.Args) > 0 && inst.Args[0] != "" {
			dbName = inst.Args[0]
		}

		// For simple MongoDB commands, execute via RunCommand
		result := client.Database(dbName).RunCommand(execCtx, []byte(inst.Raw))
		if result.Err() != nil {
			duration := int(time.Since(start).Milliseconds())
			auditLog(ctx, userID, proj, ds, "execute", "mongo", audit.StatusFailed, sqlToExecute, result.Err().Error(), duration, classifyResult)
			return nil, fmt.Errorf("%w: %w", ErrMongoCommandFailed, result.Err())
		}

		var doc map[string]any
		if err := result.Decode(&doc); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMongoDecodeFailed, err)
		}

		var columns []driver.ColumnInfo
		var rows [][]any
		for k, v := range doc {
			columns = append(columns, driver.ColumnInfo{Name: k})
			rows = append(rows, []any{fmt.Sprintf("%v", v)})
		}

		allResults = append(allResults, QueryResult{
			Columns: columns,
			Rows:    rows,
			Total:   len(rows),
		})
	}

	duration := int(time.Since(start).Milliseconds())
	auditLog(ctx, userID, proj, ds, "execute", "mongo", audit.StatusSuccess, sqlToExecute, "", duration, classifyResult)

	return map[string]interface{}{
		"mode":      "direct",
		"execution": nil,
		"results":   allResults,
	}, nil
}

// executeSearchRead 按 ds.Type 分发 Search 读执行
func executeSearchRead(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, sqlToExecute string, classifyResult *pipeline.ClassifyResult) (any, error) {
	switch ds.Type {
	case "es":
		return executeESRead(ctx, userID, proj, ds, sqlToExecute, classifyResult)
	default:
		return nil, fmt.Errorf("%w (search): %s", ErrUnsupportedType, ds.Type)
	}
}

func executeESRead(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, sqlToExecute string, classifyResult *pipeline.ClassifyResult) (any, error) {
	client, err := dbpool.GetES(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetDBConnection, err)
	}

	start := time.Now()
	var allResults []QueryResult

	for _, inst := range classifyResult.Instructions {
		// Parse the ES request: METHOD /path\nbody
		lines := strings.SplitN(inst.Raw, "\n", 2)
		firstLine := lines[0]
		body := ""
		if len(lines) > 1 {
			body = lines[1]
		}

		parts := strings.Fields(firstLine)
		if len(parts) < 2 {
			continue
		}
		method := strings.ToUpper(parts[0])
		path := parts[1]

		// Build the HTTP request manually for maximum flexibility
		var bodyReader io.Reader
		if body != "" {
			bodyReader = strings.NewReader(body)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, path, bodyReader)
		if reqErr != nil {
			duration := int(time.Since(start).Milliseconds())
			auditLog(ctx, userID, proj, ds, "execute", "es", audit.StatusFailed, sqlToExecute, reqErr.Error(), duration, classifyResult)
			return nil, fmt.Errorf("%w: %w", ErrESRequestFailed, reqErr)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}

		// Use the ES client's transport to execute
		resp, reqErr := client.Transport.Perform(req)
		if reqErr != nil {
			duration := int(time.Since(start).Milliseconds())
			auditLog(ctx, userID, proj, ds, "execute", "es", audit.StatusFailed, sqlToExecute, reqErr.Error(), duration, classifyResult)
			return nil, fmt.Errorf("%w: %w", ErrESRequestFailed, reqErr)
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
			duration := int(time.Since(start).Milliseconds())
			auditLog(ctx, userID, proj, ds, "execute", "es", audit.StatusFailed, sqlToExecute, errMsg, duration, classifyResult)
			return nil, fmt.Errorf("%w: %s", ErrESRequestFailed, errMsg)
		}

		columns := []driver.ColumnInfo{{Name: "result"}}
		rows := [][]any{{string(bodyBytes)}}

		allResults = append(allResults, QueryResult{
			Columns: columns,
			Rows:    rows,
			Total:   1,
		})
	}

	duration := int(time.Since(start).Milliseconds())
	auditLog(ctx, userID, proj, ds, "execute", "es", audit.StatusSuccess, sqlToExecute, "", duration, classifyResult)

	return map[string]interface{}{
		"mode":    "direct",
		"results": allResults,
	}, nil
}

func ExecuteEscalated(ctx context.Context, userID string, req ExecuteRequest) (any, error) {
	if req.ProjectID == "" || req.Sql == "" {
		return nil, fmt.Errorf("%w: project_id and sql are required", ErrInvalidParam)
	}

	sqlToExecute := req.Sql
	if req.SelectedText != "" {
		sqlToExecute = req.SelectedText
	}

	proj, err := project.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrProjectNotFound, req.ProjectID)
	}

	codes, _ := auth.GetUserPermissionCodes(ctx, userID)
	if !project.HasProjectAccess(ctx, userID, req.ProjectID, codes) {
		return nil, ErrNoProjectAccess
	}

	projDatabases := project.SplitDatabases(proj.Scope)
	dbName := req.Database
	if dbName != "" {
		if !project.IsDatabaseAllowed(projDatabases, dbName) {
			return nil, fmt.Errorf("%w: %s", ErrDatabaseNotAllowed, dbName)
		}
	} else if len(projDatabases) == 1 && !project.HasWildcard(projDatabases) {
		dbName = projDatabases[0]
	}

	ds, err := datasource.GetByID(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDatasourceNotFound, proj.DatasourceID)
	}

	if ds.Type == "redis" {
		return nil, ErrRedisEscalatedNotSup
	}

	pipe := newPipeline()
	classifyResult, err := pipe.Classify(ctx, ds.Type, ds.TypeGroup, req.Sql, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

	sum := classifyResult.Summary

	if sum.HasDangerous {
		auditLog(ctx, userID, proj, ds, "escalated_execute", "dangerous", audit.StatusRejected, sqlToExecute, "dangerous operation rejected", 0, classifyResult)
		return nil, ErrDangerous
	}

	if sum.HasUnknown {
		auditLog(ctx, userID, proj, ds, "escalated_execute", "unknown", audit.StatusRejected, sqlToExecute, "unrecognizable statement", 0, classifyResult)
		return nil, ErrUnknown
	}

	// 超管（perm code: "*"）无需提权审批，直接执行
	if !auth.HasPermission(codes, "*") {
		activeResp, err := escalation.CheckActiveEscalation(ctx, userID, req.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCheckEscalation, err)
		}
		if !activeResp.Active {
			auditLog(ctx, userID, proj, ds, "escalated_execute", "write", audit.StatusRejected, sqlToExecute, "no active escalation", 0, classifyResult)
			return nil, fmt.Errorf("%w: 当前项目无有效提权，请先申请提权", ErrNoActiveEscalation)
		}
	}

	pool, err := dbpool.Get(ctx, proj.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetDBConnection, err)
	}

	execCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	sqlConn, err := driver.GetSQLConnector(ds.Type)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, ds.Type)
	}
	if dbName != "" {
		if err := sqlConn.SetDatabase(execCtx, pool, dbName); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrSelectDBFailed, dbName, err)
		}
	}

	start := time.Now()
	var affectedRows int64
	var execErrStr string
	for _, inst := range classifyResult.Instructions {
		stmt := inst.Raw
		result, execErr := pool.ExecContext(execCtx, stmt)
		if execErr != nil {
			if execErrStr == "" {
				execErrStr = execErr.Error()
			} else {
				execErrStr += "; " + execErr.Error()
			}
		} else if result != nil {
			rows, _ := result.RowsAffected()
			affectedRows += rows
		}
	}

	duration := int(time.Since(start).Milliseconds())
	status := StatusSuccess
	if execErrStr != "" {
		status = StatusFailed
	}

	exec := &SqlExecution{
		ProjectID:      req.ProjectID,
		Sql:            sqlToExecute,
		Classification: "write",
		Status:         status,
		AffectedRows:   int(affectedRows),
		DurationMs:     duration,
	}
	_ = CreateExecution(ctx, exec)

	auditStatus := audit.StatusSuccess
	errMsg := ""
	if execErrStr != "" {
		auditStatus = audit.StatusFailed
		errMsg = execErrStr
	}
	auditLog(ctx, userID, proj, ds, "escalated_execute", "write", auditStatus, sqlToExecute, errMsg, duration, classifyResult)

	if execErrStr != "" {
		return nil, fmt.Errorf("%w: %s", ErrExecutionFailed, execErrStr)
	}

	return map[string]interface{}{
		"mode":      "direct",
		"execution": ToDTO(exec),
		"results":   []QueryResult{},
	}, nil
}

func CreateExecution(ctx context.Context, exec *SqlExecution) error {
	return global.DB.WithContext(ctx).Create(exec).Error
}
func auditLog(ctx context.Context, userID string, proj *project.DataProject, ds *datasource.Datasource, action, classification, status, rawText, errMsg string, durationMs int, classifyResult *pipeline.ClassifyResult) {
	instJSON := ""
	if classifyResult != nil && len(classifyResult.Instructions) > 0 {
		if b, err := json.Marshal(classifyResult.Instructions); err == nil {
			instJSON = string(b)
		}
	}
	audit.CreateAuditLog(ctx, audit.CreateAuditLogRequest{
		ActorID:         userID,
		ProjectID:       proj.ID,
		DatasourceID:    proj.DatasourceID,
		Action:          action,
		RawText:         rawText,
		InstructionJSON: instJSON,
		Classification:  classification,
		Status:          status,
		ErrorMessage:    errMsg,
		DurationMs:      durationMs,
	})
}
