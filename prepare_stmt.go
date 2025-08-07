package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"sync"
	"time"

	"gorm.io/gorm/internal/stmt_store"
)

// PreparedStmtDB
// prepare 模式下的 connPool 实现类.
type PreparedStmtDB struct {
	// 各 stmt 实例. 其中 key 为 sql 模板，stmt 是对封 database/sql 中 *Stmt 的封装
	Stmts stmt_store.Store
	Mux   *sync.RWMutex
	// 内置的 ConnPool 字段通常为 database/sql 中的 *DB
	ConnPool
}

// NewPreparedStmtDB creates and initializes a new instance of PreparedStmtDB.
//
// Parameters:
// - connPool: A connection pool that implements the ConnPool interface, used for managing database connections.
// - maxSize: The maximum number of prepared statements that can be stored in the statement store.
// - ttl: The time-to-live duration for each prepared statement in the store. Statements older than this duration will be automatically removed.
//
// Returns:
// - A pointer to a PreparedStmtDB instance, which manages prepared statements using the provided connection pool and configuration.
func NewPreparedStmtDB(connPool ConnPool, maxSize int, ttl time.Duration) *PreparedStmtDB {
	return &PreparedStmtDB{
		ConnPool: connPool,                     // Assigns the provided connection pool to manage database connections.
		Stmts:    stmt_store.New(maxSize, ttl), // Initializes a new statement store with the specified maximum size and TTL.
		Mux:      &sync.RWMutex{},              // Sets up a read-write mutex for synchronizing access to the statement store.
	}
}

// GetDBConn returns the underlying *sql.DB connection
func (db *PreparedStmtDB) GetDBConn() (*sql.DB, error) {
	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		return sqldb, nil
	}

	if dbConnector, ok := db.ConnPool.(GetDBConnector); ok && dbConnector != nil {
		return dbConnector.GetDBConn()
	}

	return nil, ErrInvalidDB
}

// Close closes all prepared statements in the store
func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	defer db.Mux.Unlock()

	for _, key := range db.Stmts.Keys() {
		db.Stmts.Delete(key)
	}
}

// Reset Deprecated use Close instead
func (db *PreparedStmtDB) Reset() {
	db.Close()
}

// 加读锁，然后以 sql 模板为 key，尝试从 db.Stmts map 中获取 stmt 复用
// 倘若 stmt 不存在，则加写锁 double check
// 调用 conn.PrepareContext(...) 方法，创建新的 stmt，并存放到 map 中供后续复用
func (db *PreparedStmtDB) prepare(ctx context.Context, conn ConnPool, isTransaction bool, query string) (_ *stmt_store.Stmt, err error) {
	// 并发场景下，只允许有一个 goroutine 完成 stmt 的初始化操作
	db.Mux.RLock()
	if db.Stmts != nil {
		// 以 sql 模板为 key，优先复用已有的 stmt
		if stmt, ok := db.Stmts.Get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.RUnlock()
			return stmt, stmt.Error()
		}
	}
	db.Mux.RUnlock()

	// retry
	// 加锁 double check，确认未完成 stmt 初始化则执行初始化操作
	db.Mux.Lock()
	if db.Stmts != nil {
		if stmt, ok := db.Stmts.Get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.Unlock()
			return stmt, stmt.Error()
		}
	}

	return db.Stmts.New(ctx, query, isTransaction, conn, db.Mux)
}

func (db *PreparedStmtDB) BeginTx(ctx context.Context, opt *sql.TxOptions) (ConnPool, error) {
	if beginner, ok := db.ConnPool.(TxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, opt)
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, err
	}

	beginner, ok := db.ConnPool.(ConnPoolBeginner)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	connPool, err := beginner.BeginTx(ctx, opt)
	if err != nil {
		return nil, err
	}
	if tx, ok := connPool.(Tx); ok {
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, nil
	}
	return nil, ErrInvalidTransaction
}

// ExecContext
// 在 prepare 模式下，执行操作通过 PreparedStmtDB.ExecContext(...) 方法实现.
// 首先通过 PreparedStmtDB.prepare(...) 方法尝试复用 stmt，然后调用 stmt.ExecContext(...) 执行查询操作.
// 此处 stm.ExecContext(...) 方法本质上会使用 database/sql 中的 sql.Stmt 完成任务.
func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Stmts.Delete(query)
		}
	}
	return result, err
}

// QueryContext
// 在 prepare 模式下，查询操作通过 PreparedStmtDB.QueryContext(...) 方法实现.
// 首先通过 PreparedStmtDB.prepare(...) 方法尝试复用 stmt，然后调用 stmt.QueryContext(...) 执行查询操作.
// 此处 stm.QueryContext(...) 方法本质上会使用 database/sql 中的 sql.Stmt 完成任务.
func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Stmts.Delete(query)
		}
	}
	return rows, err
}

func (db *PreparedStmtDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		return stmt.QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

func (db *PreparedStmtDB) Ping() error {
	conn, err := db.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}

type PreparedStmtTX struct {
	Tx
	PreparedStmtDB *PreparedStmtDB
}

func (db *PreparedStmtTX) GetDBConn() (*sql.DB, error) {
	return db.PreparedStmtDB.GetDBConn()
}

func (tx *PreparedStmtTX) Commit() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Commit()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) Rollback() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Rollback()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		result, err = tx.Tx.StmtContext(ctx, stmt.Stmt).ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Stmts.Delete(query)
		}
	}
	return result, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		rows, err = tx.Tx.StmtContext(ctx, stmt.Stmt).QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Stmts.Delete(query)
		}
	}
	return rows, err
}

func (tx *PreparedStmtTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		return tx.Tx.StmtContext(ctx, stmt.Stmt).QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

func (tx *PreparedStmtTX) Ping() error {
	conn, err := tx.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}
