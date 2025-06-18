package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

var (
	dbQueryDuration metric.Float64Histogram
	dbQueryCount    metric.Int64Counter
)

type UserSymbols struct {
	gorm.Model
	Symbol   string `gorm:"size:100;not null;index"`
	UserId   string `gorm:"size:200;not null;index"`
	Type     string `gorm:"size:50;not null;check:type IN ('STOCK', 'CRYPTO')"`
	CryptoId string `gorm:"size:200;default:null"`
}

func (UserSymbols) TableName() string {
	return "user_symbols"
}

func initDB() error {

	connectionStr := getDBConnectionString()

	config := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	var err error
	DB, err = gorm.Open(postgres.Open(connectionStr), config)
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("error getting underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	meter := otel.Meter("stock-tracker-db-metrics")

	dbQueryDuration, err = meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Measures the duration of database queries."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create db.query.duration instrument: %w", err)
	}

	dbQueryCount, err = meter.Int64Counter(
		"db.query.count",
		metric.WithDescription("Measures the number of database queries."),
		metric.WithUnit("{query}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create db.query.count instrument: %w", err)
	}

	if err := DB.Use(otelgorm.NewPlugin(
		otelgorm.WithDBName("stock-tracker-db"),
		otelgorm.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.name", "mydb1"),
			attribute.String("service.name", "stock-tracker"),
			attribute.String("service.version", "0.1.0"),
		),
	)); err != nil {
		return fmt.Errorf("error enabling OpenTelemetry for GORM: %w", err)
	}

	DB.Callback().Query().Before("gorm:query").Register("query_start_time", func(db *gorm.DB) {

		if db.Statement.Context == nil {
			db.Statement.Context = context.Background()
		}
		db.Statement.Context = context.WithValue(db.Statement.Context, "gorm_start_time", time.Now())
	})
	DB.Callback().Create().Before("gorm:create").Register("create_start_time", func(db *gorm.DB) {
		if db.Statement.Context == nil {
			db.Statement.Context = context.Background()
		}
		db.Statement.Context = context.WithValue(db.Statement.Context, "gorm_start_time", time.Now())
	})
	DB.Callback().Update().Before("gorm:update").Register("update_start_time", func(db *gorm.DB) {
		if db.Statement.Context == nil {
			db.Statement.Context = context.Background()
		}
		db.Statement.Context = context.WithValue(db.Statement.Context, "gorm_start_time", time.Now())
	})
	DB.Callback().Delete().Before("gorm:delete").Register("delete_start_time", func(db *gorm.DB) {
		if db.Statement.Context == nil {
			db.Statement.Context = context.Background()
		}
		db.Statement.Context = context.WithValue(db.Statement.Context, "gorm_start_time", time.Now())
	})
	DB.Callback().Raw().Before("gorm:raw").Register("raw_start_time", func(db *gorm.DB) {
		if db.Statement.Context == nil {
			db.Statement.Context = context.Background()
		}
		db.Statement.Context = context.WithValue(db.Statement.Context, "gorm_start_time", time.Now())
	})

	recordGormMetricsAndSpanStatus := func(db *gorm.DB) {
		ctx := db.Statement.Context
		if ctx == nil {
			ctx = context.Background()
		}

		op := "unknown"

		sqlStr := db.Statement.SQL.String()
		if sqlStr != "" && len(sqlStr) >= 6 {
			op = sqlStr[:6]
		}
		tableName := db.Statement.Table

		isError := db.Error != nil && db.Error != gorm.ErrRecordNotFound

		attrs := []attribute.KeyValue{
			attribute.String("db.operation", op),
			attribute.String("db.table", tableName),
			attribute.Bool("db.error", isError),
		}

		if startTime, ok := ctx.Value("gorm_start_time").(time.Time); ok {
			duration := time.Since(startTime).Seconds()
			if dbQueryDuration != nil {
				dbQueryDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
			}
		}

		if dbQueryCount != nil {
			dbQueryCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		}

		if isError {
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.SetStatus(codes.Error, db.Error.Error())
				span.RecordError(db.Error)
			}
		} else {
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.SetStatus(codes.Ok, "OK")
			}
		}
	}

	DB.Callback().Query().After("gorm:after_query").Register("query_metrics", recordGormMetricsAndSpanStatus)
	DB.Callback().Create().After("gorm:after_create").Register("create_metrics", recordGormMetricsAndSpanStatus)
	DB.Callback().Update().After("gorm:after_update").Register("update_metrics", recordGormMetricsAndSpanStatus)
	DB.Callback().Delete().After("gorm:after_delete").Register("delete_metrics", recordGormMetricsAndSpanStatus)
	DB.Callback().Raw().After("gorm:after_raw").Register("raw_metrics", recordGormMetricsAndSpanStatus)

	if err := DB.AutoMigrate(&UserSymbols{}); err != nil {
		return fmt.Errorf("error migrating database: %w", err)
	}

	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_user_symbols_user_type ON user_symbols(user_id, type)").Error; err != nil {
		log.Printf("Warning: Could not create composite index: %v", err)
	}

	fmt.Println("Database connected and instrumented successfully")
	return nil
}

func getDBConnectionString() string {
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("DB_USER", "abdullah")
	password := getEnv("DB_PASSWORD", "edhi")
	dbname := getEnv("DB_NAME", "mydb1")
	port := getEnv("DB_PORT", "5432")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func CloseDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return fmt.Errorf("error getting underlying sql.DB: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}
