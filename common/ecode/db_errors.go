package ecode

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL 错误码常量
const (
	// PgUniqueViolation PostgreSQL 唯一约束冲突错误码
	PgUniqueViolation = "23505"
)

// IsDuplicateKeyError 检查是否为唯一约束冲突错误（重复键）
// 适用于 PostgreSQL 数据库
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == PgUniqueViolation {
		return true
	}
	return false
}
