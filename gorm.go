package gorm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// for Config.cacheStore store PreparedStmtDB key
const preparedStmtDBKey = "preparedStmt"

// Config GORM config
type Config struct {
	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	// 会自动开启事务以保证数据一致性。将此项设置为 true 可以跳过这个默认行为，从而提升性能。
	// 适合在你已确保业务逻辑中不会产生数据不一致问题的情况下使用。
	SkipDefaultTransaction bool

	// 如果事务在指定时间内未完成，将自动回滚。
	DefaultTransactionTimeout time.Duration

	// NamingStrategy tables, columns naming strategy
	// NamingStrategy 命名策略，用于控制表名、列名等的生成规则。
	// 可以通过此项自定义命名风格（如是否使用下划线，是否复数等）。
	NamingStrategy schema.Namer

	// FullSaveAssociations full save associations
	// FullSaveAssociations 是否在保存数据时完整保存所有关联（如 has many、belongs to 等）。
	// 默认为 false，只保存主数据。设为 true 会同步所有关联数据（包括更新、删除等）。
	FullSaveAssociations bool

	// Logger
	// Logger 日志接口，允许设置日志记录器（如 GORM 自带的 logger.Default）。
	// 可用于调试 SQL 语句、慢查询分析等。
	Logger logger.Interface

	// NowFunc the function to be used when creating a new timestamp
	// NowFunc 获取当前时间的函数。GORM 在创建时间戳字段时调用该函数。
	// 可自定义时间源（如用于模拟时间、统一时区等）。
	NowFunc func() time.Time

	// DryRun generate sql without execute
	// DryRun 设置为 true 时不会实际执行 SQL，只生成 SQL 语句并返回结果。
	// 通常用于调试或生成 SQL 脚本。
	DryRun bool

	// PrepareStmt executes the given query in cached statement
	// PrepareStmt 启用预编译语句并缓存，可以提高数据库操作性能，尤其是批量操作时。
	// 会使用数据库连接的预编译特性。
	PrepareStmt bool

	// PrepareStmt cache support LRU expired,
	// default maxsize=int64 Max value and ttl=1h
	// PrepareStmtMaxSize 设置缓存中最大预编译语句数量（LRU 缓存），超过则逐出最旧的。
	// 默认是 int64 最大值。
	PrepareStmtMaxSize int

	// PrepareStmtTTL 设置缓存中每个预编译语句的存活时间，默认是 1 小时。
	PrepareStmtTTL time.Duration

	// DisableAutomaticPing
	// DisableAutomaticPing 禁用自动 ping 数据库（GORM 在启动时会尝试 ping 数据库）。
	// 某些数据库或网络条件下可设置为 true 来跳过。
	DisableAutomaticPing bool

	// DisableForeignKeyConstraintWhenMigrating
	// DisableForeignKeyConstraintWhenMigrating 在迁移（AutoMigrate）时禁用外键约束创建。
	// 某些数据库或出于设计需要可以关闭外键。
	DisableForeignKeyConstraintWhenMigrating bool

	// IgnoreRelationshipsWhenMigrating
	// IgnoreRelationshipsWhenMigrating 在迁移时忽略模型间的关联关系（不处理外键、联表等）。
	// 对于不需要数据库层级关联的模型非常有用。
	IgnoreRelationshipsWhenMigrating bool

	// DisableNestedTransaction disable nested transaction
	// DisableNestedTransaction 禁用嵌套事务（Nested Transaction）。
	// 一些数据库不支持嵌套事务，或者你想手动控制事务嵌套逻辑时可以设置为 true。
	DisableNestedTransaction bool

	// AllowGlobalUpdate allow global update
	// 倘若未启用 AllowGlobalUpdate 模式，则会校验使用方是否设置了 where 条件，
	// 未设置会抛出 gorm.ErrMissingWhereClause 错误（对应 checkMissingWhereConditions() 方法）
	AllowGlobalUpdate bool

	// QueryFields executes the SQL query with all fields of the table
	// QueryFields 查询时默认选择所有字段，即使只使用了部分字段。
	// 可用于某些特定场景下避免字段缺失的问题。
	QueryFields bool

	// CreateBatchSize default create batch size
	// CreateBatchSize 设置批量创建记录时的默认每批数量。
	// 数据量大时建议设置为合适的值（如 100、500 等），以避免 SQL 长度超限。
	CreateBatchSize int

	// TranslateError enabling error translation
	// TranslateError 启用数据库错误转换，例如将数据库唯一键冲突错误转换为更易理解的错误类型。
	TranslateError bool

	// PropagateUnscoped propagate Unscoped to every other nested statement
	// PropagateUnscoped 当使用 Unscoped 时，是否将其传递给所有嵌套语句。
	// 默认只对当前语句生效。设置为 true 可以使其全局生效。
	PropagateUnscoped bool

	// ClauseBuilders clause builder
	// ClauseBuilders 子句构造器，用于自定义 SQL 中的子句构建方式。
	// 高级功能，通常用于扩展 GORM 行为或定制 SQL。
	ClauseBuilders map[string]clause.ClauseBuilder

	// ConnPool db conn pool
	// ConnPool 数据库连接池接口，GORM 使用它来管理数据库连接。
	// 可设置为自定义的连接池以满足不同的连接策略。
	ConnPool ConnPool

	// Dialector database dialector
	// Dialector 数据库方言定义，如 mysql、postgres、sqlite 等。
	// GORM 根据 Dialector 确定如何与不同数据库交互。
	Dialector

	// Plugins registered plugins
	// Plugins 已注册的插件集合，可用于扩展 GORM 功能（如审计、软删除增强等）。
	Plugins map[string]Plugin

	// callbacks 回调链，GORM 的核心执行机制之一，处理生命周期中的各类钩子。
	// 通常为内部使用，不建议修改。
	callbacks *callbacks

	// cacheStore 用于内部缓存，如 SQL 构造缓存、表结构缓存等。
	// 类型为 sync.Map，线程安全。
	cacheStore *sync.Map
}

// Apply update config to new config
func (c *Config) Apply(config *Config) error {
	if config != c {
		*config = *c
	}
	return nil
}

// AfterInitialize initialize plugins after db connected
func (c *Config) AfterInitialize(db *DB) error {
	if db != nil {
		for _, plugin := range c.Plugins {
			if err := plugin.Initialize(db); err != nil {
				return err
			}
		}
	}
	return nil
}

// Option gorm option interface
type Option interface {
	Apply(*Config) error
	AfterInitialize(*DB) error
}

// DB GORM DB definition
// gorm 中定义的数据库类
// 所有 orm 的思想
type DB struct {
	// 用户自定义的配置项
	*Config
	// 一次会话执行过程中遇到的错误
	Error error
	// 该请求影响的行数
	RowsAffected int64
	// 一次会话的状态信息，比如请求和响应信息
	Statement *Statement
	// 会话被克隆的次数. 倘若 clone = 1，代表是始祖 DB 实例；倘若 clone > 1，代表是从始祖 DB 克隆出来的会话
	clone int
}

// Session session config when create session with Session() method
type Session struct {
	DryRun                   bool
	PrepareStmt              bool
	NewDB                    bool
	Initialized              bool
	SkipHooks                bool
	SkipDefaultTransaction   bool
	DisableNestedTransaction bool
	AllowGlobalUpdate        bool
	FullSaveAssociations     bool
	PropagateUnscoped        bool
	QueryFields              bool
	Context                  context.Context
	Logger                   logger.Interface
	NowFunc                  func() time.Time
	CreateBatchSize          int
}

// Open initialize db session based on dialector
// 完成 gorm.Config 配置的创建和注入
// 完成连接器 dialector 的注入，本篇使用的是 mysql 版本
// 完成 callbacks 中 crud 等几类 processor 的创建 ( 通过 initializeCallbacks(...) 方法 )
// 完成 connPool 的创建以及各类 processor fns 函数的注册（ 通过 dialector.Initialize(...) 方法 ）
// 倘若启用了 prepare 模式，需要使用 preparedStmtDB 进行 connPool 的平替
// 构造 statement 实例
// 根据策略，决定是否通过 ping 请求测试连接
// 返回创建好的 db 实例
func Open(dialector Dialector, opts ...Option) (db *DB, err error) {
	config := &Config{}

	sort.Slice(opts, func(i, j int) bool {
		_, isConfig := opts[i].(*Config)
		_, isConfig2 := opts[j].(*Config)
		return isConfig && !isConfig2
	})

	var skipAfterInitialize bool
	for _, opt := range opts {
		if opt != nil {
			if applyErr := opt.Apply(config); applyErr != nil {
				return nil, applyErr
			}
			defer func(opt Option) {
				if skipAfterInitialize {
					return
				}
				if errr := opt.AfterInitialize(db); errr != nil {
					err = errr
				}
			}(opt)
		}
	}

	if d, ok := dialector.(interface{ Apply(*Config) error }); ok {
		if err = d.Apply(config); err != nil {
			return
		}
	}

	// 表、列命名策略
	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{IdentifierMaxLength: 64} // Default Identifier length is 64
	}

	if config.Logger == nil {
		config.Logger = logger.Default
	}

	if config.NowFunc == nil {
		config.NowFunc = func() time.Time { return time.Now().Local() }
	}

	// 连接器
	if dialector != nil {
		config.Dialector = dialector
	}

	if config.Plugins == nil {
		config.Plugins = map[string]Plugin{}
	}

	if config.cacheStore == nil {
		config.cacheStore = &sync.Map{}
	}

	db = &DB{Config: config, clone: 1}

	// 初始化 callback 当中的各个 processor
	db.callbacks = initializeCallbacks(db)

	if config.ClauseBuilders == nil {
		config.ClauseBuilders = map[string]clause.ClauseBuilder{}
	}

	if config.Dialector != nil {
		// 在其中会对 crud 各个方法的 callback 方法进行注册
		// 会对 db.connPool 进行初始化，通常情况下是 database/sql 库下 *sql.DB 的类型
		err = config.Dialector.Initialize(db)
		if err != nil {
			if db, _ := db.DB(); db != nil {
				_ = db.Close()
			}

			// DB is not initialized, so we skip AfterInitialize
			skipAfterInitialize = true
			return
		}

		if config.TranslateError {
			if _, ok := db.Dialector.(ErrorTranslator); !ok {
				config.Logger.Warn(context.Background(), "The TranslateError option is enabled, but the Dialector %s does not implement ErrorTranslator.", db.Dialector.Name())
			}
		}
	}

	// 是否启用 prepare 模式
	if config.PrepareStmt {
		preparedStmt := NewPreparedStmtDB(db.ConnPool, config.PrepareStmtMaxSize, config.PrepareStmtTTL)
		db.cacheStore.Store(preparedStmtDBKey, preparedStmt)
		db.ConnPool = preparedStmt
	}

	// 构造一个 statement 用于存储处理链路中的一些状态信息
	db.Statement = &Statement{
		DB:       db,
		ConnPool: db.ConnPool,
		Context:  context.Background(),
		Clauses:  map[string]clause.Clause{},
	}

	// 倘若未禁用 AutomaticPing
	if err == nil && !config.DisableAutomaticPing {
		if pinger, ok := db.ConnPool.(interface{ Ping() error }); ok {
			err = pinger.Ping()
		}
	}

	if err != nil {
		config.Logger.Error(context.Background(), "failed to initialize database, got error %v", err)
	}

	return
}

// Session create new db session
func (db *DB) Session(config *Session) *DB {
	var (
		txConfig = *db.Config
		tx       = &DB{
			Config:    &txConfig,
			Statement: db.Statement,
			Error:     db.Error,
			clone:     1,
		}
	)
	if config.CreateBatchSize > 0 {
		tx.Config.CreateBatchSize = config.CreateBatchSize
	}

	if config.SkipDefaultTransaction {
		tx.Config.SkipDefaultTransaction = true
	}

	if config.AllowGlobalUpdate {
		txConfig.AllowGlobalUpdate = true
	}

	if config.FullSaveAssociations {
		txConfig.FullSaveAssociations = true
	}

	if config.PropagateUnscoped {
		txConfig.PropagateUnscoped = true
	}

	if config.Context != nil || config.PrepareStmt || config.SkipHooks {
		tx.Statement = tx.Statement.clone()
		tx.Statement.DB = tx
	}

	if config.Context != nil {
		tx.Statement.Context = config.Context
	}

	if config.PrepareStmt {
		var preparedStmt *PreparedStmtDB

		if v, ok := db.cacheStore.Load(preparedStmtDBKey); ok {
			preparedStmt = v.(*PreparedStmtDB)
		} else {
			preparedStmt = NewPreparedStmtDB(db.ConnPool, db.PrepareStmtMaxSize, db.PrepareStmtTTL)
			db.cacheStore.Store(preparedStmtDBKey, preparedStmt)
		}

		switch t := tx.Statement.ConnPool.(type) {
		case Tx:
			tx.Statement.ConnPool = &PreparedStmtTX{
				Tx:             t,
				PreparedStmtDB: preparedStmt,
			}
		default:
			tx.Statement.ConnPool = &PreparedStmtDB{
				ConnPool: db.Config.ConnPool,
				Mux:      preparedStmt.Mux,
				Stmts:    preparedStmt.Stmts,
			}
		}
		txConfig.ConnPool = tx.Statement.ConnPool
		txConfig.PrepareStmt = true
	}

	if config.SkipHooks {
		tx.Statement.SkipHooks = true
	}

	if config.DisableNestedTransaction {
		txConfig.DisableNestedTransaction = true
	}

	if !config.NewDB {
		tx.clone = 2
	}

	if config.DryRun {
		tx.Config.DryRun = true
	}

	if config.QueryFields {
		tx.Config.QueryFields = true
	}

	if config.Logger != nil {
		tx.Config.Logger = config.Logger
	}

	if config.NowFunc != nil {
		tx.Config.NowFunc = config.NowFunc
	}

	if config.Initialized {
		tx = tx.getInstance()
	}

	return tx
}

// WithContext change current instance db's context to ctx
func (db *DB) WithContext(ctx context.Context) *DB {
	return db.Session(&Session{Context: ctx})
}

// Debug start debug mode
func (db *DB) Debug() (tx *DB) {
	tx = db.getInstance()
	return tx.Session(&Session{
		Logger: db.Logger.LogMode(logger.Info),
	})
}

// Set store value with key into current db instance's context
func (db *DB) Set(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(key, value)
	return tx
}

// Get get value with key from current db instance's context
func (db *DB) Get(key string) (interface{}, bool) {
	return db.Statement.Settings.Load(key)
}

// InstanceSet store value with key into current db instance's context
func (db *DB) InstanceSet(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(fmt.Sprintf("%p", tx.Statement)+key, value)
	return tx
}

// InstanceGet get value with key from current db instance's context
func (db *DB) InstanceGet(key string) (interface{}, bool) {
	return db.Statement.Settings.Load(fmt.Sprintf("%p", db.Statement) + key)
}

// Callback returns callback manager
func (db *DB) Callback() *callbacks {
	return db.callbacks
}

// AddError add error to db
// DB 类的 AddError 方法，用于在会话执行过程中抛出错误.
// 一次会话在执行过程中可能会遇到多个错误，因此会通过 error wrapping 的方式，实现错误的拼接.
func (db *DB) AddError(err error) error {
	if err != nil {
		if db.Config.TranslateError {
			if errTranslator, ok := db.Dialector.(ErrorTranslator); ok {
				err = errTranslator.Translate(err)
			}
		}

		if db.Error == nil {
			db.Error = err
		} else {
			db.Error = fmt.Errorf("%v; %w", db.Error, err)
		}
	}
	return db.Error
}

// DB returns `*sql.DB`
func (db *DB) DB() (*sql.DB, error) {
	connPool := db.ConnPool
	if db.Statement != nil && db.Statement.ConnPool != nil {
		connPool = db.Statement.ConnPool
	}
	if tx, ok := connPool.(*sql.Tx); ok && tx != nil {
		return (*sql.DB)(reflect.ValueOf(tx).Elem().FieldByName("db").UnsafePointer()), nil
	}

	if dbConnector, ok := connPool.(GetDBConnector); ok && dbConnector != nil {
		if sqldb, err := dbConnector.GetDBConn(); sqldb != nil || err != nil {
			return sqldb, err
		}
	}

	if sqldb, ok := connPool.(*sql.DB); ok && sqldb != nil {
		return sqldb, nil
	}

	return nil, ErrInvalidDB
}

func (db *DB) getInstance() *DB {
	if db.clone > 0 {
		tx := &DB{Config: db.Config, Error: db.Error}

		// 倘若是首次对 db 进行 clone，则需要构造出一个新的 statement 实例
		if db.clone == 1 {
			// clone with new statement
			tx.Statement = &Statement{
				DB:        tx,
				ConnPool:  db.Statement.ConnPool,
				Context:   db.Statement.Context,
				Clauses:   map[string]clause.Clause{},
				Vars:      make([]interface{}, 0, 8),
				SkipHooks: db.Statement.SkipHooks,
			}
			if db.Config.PropagateUnscoped {
				tx.Statement.Unscoped = db.Statement.Unscoped
			}
		} else {
			// with clone statement
			// 倘若已经 db clone 过了，则还需要 clone 原先的 statement
			tx.Statement = db.Statement.clone()
			tx.Statement.DB = tx
		}

		return tx
	}

	return db
}

// Expr returns clause.Expr, which can be used to pass SQL expression as params
func Expr(expr string, args ...interface{}) clause.Expr {
	return clause.Expr{SQL: expr, Vars: args}
}

// SetupJoinTable setup join table schema
func (db *DB) SetupJoinTable(model interface{}, field string, joinTable interface{}) error {
	var (
		tx                      = db.getInstance()
		stmt                    = tx.Statement
		modelSchema, joinSchema *schema.Schema
	)

	err := stmt.Parse(model)
	if err != nil {
		return err
	}
	modelSchema = stmt.Schema

	err = stmt.Parse(joinTable)
	if err != nil {
		return err
	}
	joinSchema = stmt.Schema

	relation, ok := modelSchema.Relationships.Relations[field]
	isRelation := ok && relation.JoinTable != nil
	if !isRelation {
		return fmt.Errorf("failed to find relation: %s", field)
	}

	for _, ref := range relation.References {
		f := joinSchema.LookUpField(ref.ForeignKey.DBName)
		if f == nil {
			return fmt.Errorf("missing field %s for join table", ref.ForeignKey.DBName)
		}

		f.DataType = ref.ForeignKey.DataType
		f.GORMDataType = ref.ForeignKey.GORMDataType
		if f.Size == 0 {
			f.Size = ref.ForeignKey.Size
		}
		ref.ForeignKey = f
	}

	for name, rel := range relation.JoinTable.Relationships.Relations {
		if _, ok := joinSchema.Relationships.Relations[name]; !ok {
			rel.Schema = joinSchema
			joinSchema.Relationships.Relations[name] = rel
		}
	}
	relation.JoinTable = joinSchema

	return nil
}

// Use use plugin
func (db *DB) Use(plugin Plugin) error {
	name := plugin.Name()
	if _, ok := db.Plugins[name]; ok {
		return ErrRegistered
	}
	if err := plugin.Initialize(db); err != nil {
		return err
	}
	db.Plugins[name] = plugin
	return nil
}

// ToSQL for generate SQL string.
//
//	db.ToSQL(func(tx *gorm.DB) *gorm.DB {
//			return tx.Model(&User{}).Where(&User{Name: "foo", Age: 20})
//				.Limit(10).Offset(5)
//				.Order("name ASC")
//				.First(&User{})
//	})
func (db *DB) ToSQL(queryFn func(tx *DB) *DB) string {
	tx := queryFn(db.Session(&Session{DryRun: true, SkipDefaultTransaction: true}).getInstance())
	stmt := tx.Statement

	return db.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
}
