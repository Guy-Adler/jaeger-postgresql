package sql

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const cleanSpans = `-- name: CleanSpans :execrows

DELETE FROM spans
WHERE spans.start_time < $1::TIMESTAMP
`

func (q *Queries) CleanSpans(ctx context.Context, pruneBefore pgtype.Timestamp) (int64, error) {
	result, err := q.db.Exec(ctx, cleanSpans, pruneBefore)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const findTraceIDs = `-- name: FindTraceIDs :many

SELECT DISTINCT spans.trace_id as trace_id
FROM spans
    INNER JOIN operations ON (operations.id = spans.operation_id)
    INNER JOIN services ON (services.id = spans.service_id)
WHERE
    (services.name = $1::VARCHAR OR $2::BOOLEAN = FALSE) AND
    (operations.name = $3::VARCHAR OR $4::BOOLEAN = FALSE) AND
    (start_time >= $5::TIMESTAMP OR $6::BOOLEAN = FALSE) AND
    (start_time <= $7::TIMESTAMP OR $8::BOOLEAN = FALSE) AND
    (duration >= $9::INTERVAL OR $10::BOOLEAN = FALSE) AND
    (duration <= $11::INTERVAL OR $12::BOOLEAN = FALSE) AND
		($14::BOOLEAN = FALSE OR (tags @> $13::JSONB) OR (process_tags @> $13::JSONB))
LIMIT $15
`

type TagContent struct {
	Key   string
	Value string
}
type FindTraceIDsParams struct {
	ServiceName                  string
	ServiceNameEnableFilter      bool
	OperationName                string
	OperationNameEnableFilter    bool
	StartTimeMinimum             pgtype.Timestamp
	StartTimeMinimumEnableFilter bool
	StartTimeMaximum             pgtype.Timestamp
	StartTimeMaximumEnableFilter bool
	DurationMinimum              pgtype.Interval
	DurationMinimumEnableFilter  bool
	DurationMaximum              pgtype.Interval
	DurationMaximumEnableFilter  bool
	Tags                         map[string]string
	TagsEnableFilter             bool
	NumTraces                    int32
}

func formatTags(tags map[string]string) []TagContent {
	parsedTags := make([]TagContent, 0, len(tags))
	for key, value := range tags {
		parsedTags = append(parsedTags, TagContent{
			Key:   key,
			Value: value,
		})
	}
	return parsedTags
}

func (q *Queries) FindTraceIDs(ctx context.Context, arg FindTraceIDsParams) ([][]byte, error) {
	tags := formatTags(arg.Tags)

	rows, err := q.db.Query(ctx, findTraceIDs,
		arg.ServiceName,
		arg.ServiceNameEnableFilter,
		arg.OperationName,
		arg.OperationNameEnableFilter,
		arg.StartTimeMinimum,
		arg.StartTimeMinimumEnableFilter,
		arg.StartTimeMaximum,
		arg.StartTimeMaximumEnableFilter,
		arg.DurationMinimum,
		arg.DurationMinimumEnableFilter,
		arg.DurationMaximum,
		arg.DurationMaximumEnableFilter,
		tags,
		arg.TagsEnableFilter,
		arg.NumTraces,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items [][]byte
	for rows.Next() {
		var trace_id []byte
		if err := rows.Scan(&trace_id); err != nil {
			return nil, err
		}
		items = append(items, trace_id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getOperationID = `-- name: GetOperationID :one
SELECT id 
FROM operations 
WHERE 
  name = $1::TEXT AND 
  service_id = $2::BIGINT AND 
  kind = $3::SPANKIND
`

type GetOperationIDParams struct {
	Name      string
	ServiceID int64
	Kind      Spankind
}

func (q *Queries) GetOperationID(ctx context.Context, arg GetOperationIDParams) (int64, error) {
	row := q.db.QueryRow(ctx, getOperationID, arg.Name, arg.ServiceID, arg.Kind)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const getOperations = `-- name: GetOperations :many
SELECT operations.name, operations.kind
FROM operations
  INNER JOIN services ON (operations.service_id = services.id)
WHERE services.name = $1::VARCHAR
ORDER BY operations.name ASC
`

type GetOperationsRow struct {
	Name string
	Kind Spankind
}

func (q *Queries) GetOperations(ctx context.Context, serviceName string) ([]GetOperationsRow, error) {
	rows, err := q.db.Query(ctx, getOperations, serviceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetOperationsRow
	for rows.Next() {
		var i GetOperationsRow
		if err := rows.Scan(&i.Name, &i.Kind); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getServiceID = `-- name: GetServiceID :one
SELECT id
FROM services
WHERE name = $1::TEXT
`

func (q *Queries) GetServiceID(ctx context.Context, name string) (int64, error) {
	row := q.db.QueryRow(ctx, getServiceID, name)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const getServices = `-- name: GetServices :many
SELECT services.name
FROM services
ORDER BY services.name ASC
`

func (q *Queries) GetServices(ctx context.Context) ([]string, error) {
	rows, err := q.db.Query(ctx, getServices)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSpansCount = `-- name: GetSpansCount :one

SELECT COUNT(*) FROM spans
`

func (q *Queries) GetSpansCount(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, getSpansCount)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const getSpansDiskSize = `-- name: GetSpansDiskSize :one

SELECT pg_total_relation_size('spans')
`

func (q *Queries) GetSpansDiskSize(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, getSpansDiskSize)
	var pg_total_relation_size int64
	err := row.Scan(&pg_total_relation_size)
	return pg_total_relation_size, err
}

const getTraceSpans = `-- name: GetTraceSpans :many
SELECT
  spans.span_id as span_id,
  spans.trace_id as trace_id,
  operations.name as operation_name,
  spans.flags as flags,
  spans.start_time as start_time,
  spans.duration as duration,
  spans.tags as tags,
  spans.process_id as process_id,
  spans.warnings as warnings,
  spans.kind as kind,
  services.name as process_name,
  spans.process_tags as process_tags,
  spans.logs as logs,
  spans.refs as refs
FROM spans 
  INNER JOIN operations ON (spans.operation_id = operations.id)
  INNER JOIN services ON (spans.service_id = services.id)
WHERE trace_id = $1::BYTEA
`

type GetTraceSpansRow struct {
	SpanID        []byte
	TraceID       []byte
	OperationName string
	Flags         int64
	StartTime     pgtype.Timestamp
	Duration      pgtype.Interval
	Tags          []byte
	ProcessID     string
	Warnings      []string
	Kind          Spankind
	ProcessName   string
	ProcessTags   []byte
	Logs          []byte
	Refs          []byte
}

func (q *Queries) GetTraceSpans(ctx context.Context, traceID []byte) ([]GetTraceSpansRow, error) {
	rows, err := q.db.Query(ctx, getTraceSpans, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetTraceSpansRow
	for rows.Next() {
		var i GetTraceSpansRow
		if err := rows.Scan(
			&i.SpanID,
			&i.TraceID,
			&i.OperationName,
			&i.Flags,
			&i.StartTime,
			&i.Duration,
			&i.Tags,
			&i.ProcessID,
			&i.Warnings,
			&i.Kind,
			&i.ProcessName,
			&i.ProcessTags,
			&i.Logs,
			&i.Refs,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertSpan = `-- name: InsertSpan :one
INSERT INTO spans (
  span_id,
  trace_id,
  operation_id,
  flags,
  start_time,
  duration,
  tags,
  service_id,
  process_id,
  process_tags,
  warnings,
  kind,
  logs,
  refs
)
VALUES(
  $1::BYTEA,
  $2::BYTEA,
  $3::BIGINT,
  $4::BIGINT,
  $5::TIMESTAMP,
  $6::INTERVAL,
  $7::JSONB,
  $8::BIGINT,
  $9::TEXT,
  $10::JSONB,
  $11::TEXT[],
  $12::SPANKIND,
  $13::JSONB,
  $14::JSONB
)
RETURNING spans.hack_id
`

type InsertSpanParams struct {
	SpanID      []byte
	TraceID     []byte
	OperationID int64
	Flags       int64
	StartTime   pgtype.Timestamp
	Duration    pgtype.Interval
	Tags        []byte
	ServiceID   int64
	ProcessID   string
	ProcessTags []byte
	Warnings    []string
	Kind        Spankind
	Logs        []byte
	Refs        []byte
}

func (q *Queries) InsertSpan(ctx context.Context, arg InsertSpanParams) (int64, error) {
	row := q.db.QueryRow(ctx, insertSpan,
		arg.SpanID,
		arg.TraceID,
		arg.OperationID,
		arg.Flags,
		arg.StartTime,
		arg.Duration,
		arg.Tags,
		arg.ServiceID,
		arg.ProcessID,
		arg.ProcessTags,
		arg.Warnings,
		arg.Kind,
		arg.Logs,
		arg.Refs,
	)
	var hack_id int64
	err := row.Scan(&hack_id)
	return hack_id, err
}

const upsertOperation = `-- name: UpsertOperation :exec
INSERT INTO operations (name, service_id, kind) 
VALUES (
  $1::TEXT, 
  $2::BIGINT, 
  $3::SPANKIND
) ON CONFLICT(name, service_id, kind) DO NOTHING RETURNING id
`

type UpsertOperationParams struct {
	Name      string
	ServiceID int64
	Kind      Spankind
}

func (q *Queries) UpsertOperation(ctx context.Context, arg UpsertOperationParams) error {
	_, err := q.db.Exec(ctx, upsertOperation, arg.Name, arg.ServiceID, arg.Kind)
	return err
}

const upsertService = `-- name: UpsertService :exec


INSERT INTO services (name) 
VALUES ($1::VARCHAR) ON CONFLICT(name) DO NOTHING RETURNING id
`

// -- name: GetDependencies :many
// SELECT
//
//	COUNT(*) AS call_count,
//	source_services.name as parent,
//	child_services.name as child,
//	'' as source
//
// FROM spanrefs
//
//	INNER JOIN spans AS source_spans ON (source_spans.span_id = spanrefs.source_span_id)
//	INNER JOIN spans AS child_spans ON (child_spans.span_id = spanrefs.child_span_id)
//	INNER JOIN services AS source_services ON (source_spans.service_id = source_services.id)
//	INNER JOIN services AS child_services ON (child_spans.service_id = child_services.id)
//
// GROUP BY source_services.name, child_services.name;
// -- name: FindTraceIDs :many
// SELECT DISTINCT spans.trace_id
// FROM spans
//
//	INNER JOIN operations ON (operations.id = spans.operation_id)
//	INNER JOIN services ON (services.id = spans.service_id)
//
// WHERE
//
//	(services.name = sqlc.arg(service_name)::VARCHAR OR sqlc.arg(service_name_enable)::BOOLEAN = FALSE) AND
//	(operations.name = sqlc.arg(operation_name)::VARCHAR OR sqlc.arg(operation_name_enable)::BOOLEAN = FALSE) AND
//	(start_time >= sqlc.arg(start_time_minimum)::TIMESTAMPTZ OR sqlc.arg(start_time_minimum_enable)::BOOLEAN = FALSE) AND
//	(start_time < sqlc.arg(start_time_maximum)::TIMESTAMPTZ OR sqlc.arg(start_time_maximum_enable)::BOOLEAN = FALSE) AND
//	(duration > sqlc.arg(duration_minimum)::INTERVAL OR sqlc.arg(duration_minimum_enable)::BOOLEAN = FALSE) AND
//	(duration < sqlc.arg(duration_maximum)::INTERVAL OR sqlc.arg(duration_maximum_enable)::BOOLEAN = FALSE)
//
// ;
// LIMIT sqlc.arg(limit)::INT;
func (q *Queries) UpsertService(ctx context.Context, name string) error {
	_, err := q.db.Exec(ctx, upsertService, name)
	return err
}
