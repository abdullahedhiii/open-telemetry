package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type UserSymbols struct {
	gorm.Model
	Symbol   string `gorm:"size:100; NOT NULL; UNIQUE;"`
	UserId   string `gorm:"size:200; NOT NULL;"`
	Type     string `gorm:"size:200; NOT NULL; VALUE IN (STOCK, CRYPTO)"`
	CryptoId string `gorm:"size:200; DEFAULT NULL;"` //only if crypto
}

// Custom GORM logger that integrates with OpenTelemetry
type otelLogger struct {
	logger.Interface
}

func (l *otelLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// First call the underlying logger's Trace method
	if l.Interface != nil {
		l.Interface.Trace(ctx, begin, fc, err)
	}

	// Get the SQL and rows affected
	sql, rows := fc()
	duration := time.Since(begin)

	// If context is nil, create a background context
	if ctx == nil {
		ctx = context.Background()
	}

	// Only record metrics if they are initialized
	if dbLatencyHistogram != nil {
		dbLatencyHistogram.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("query", sql),
				attribute.Int64("rows_affected", rows),
			))
	}

	if dbOperationCounter != nil {
		dbOperationCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("query", sql),
				attribute.Bool("success", err == nil),
			))
	}

	if err != nil && recordError != nil {
		recordError(ctx, "database_operation", err)
	}
}

func initDB() {
	connectionStr := "host=localhost user=abdullah password=edhi dbname=mydb1 port=5432"
	var err error

	// Create custom logger
	gormLogger := &otelLogger{
		Interface: logger.New(
			log.New(log.Writer(), "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: false,
				Colorful:                  true,
			},
		),
	}

	DB, err = gorm.Open(postgres.Open(connectionStr), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		log.Fatal("error connecting to database")
	}

	DB.AutoMigrate(&UserSymbols{})
	fmt.Println("Database connected successfully")
}
