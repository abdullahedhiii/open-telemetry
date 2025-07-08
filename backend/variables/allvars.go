package variables

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	gorm.Model
	ID       int    `gorm:"s;autoIncrement"`
	Username string `gorm:"size:100;unique"`
	Email    string `gorm:"size:200;unique"`
	Password []byte `json:"-"`
}

type UserSymbols struct {
	gorm.Model
	ID       int    `gorm:"primaryKey;autoIncrement"`
	Symbol   string `gorm:"size:100;index"`
	UserID   int    `gorm:"index"`
	User     User   `gorm:"foreignKey:UserID"`
	Type     string `gorm:"size:50;check:type IN ('STOCK', 'CRYPTO')"`
	CryptoId string `gorm:"size:200;default:null"`
}

var (
	HttpRequestCount metric.Int64Counter
	// watchlistAddAttempts    metric.Int64Counter
	ExternalAPICallDuration metric.Float64Histogram
	DbQueryCount            metric.Int64Counter
	DbQueryDuration         metric.Float64Histogram
	LoginAttempts           metric.Int64Counter
	RegisterAttempts        metric.Int64Counter
	AuthDuration            metric.Float64Histogram
)

var (
	TracerProvider *trace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	Logger         *slog.Logger
)
