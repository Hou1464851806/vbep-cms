package model

import "time"

type Statistic struct {
	Base
	MetricKey  string    `gorm:"not null;index:idx_key_date"` // 统计指标的Key: user_total, user_new, application_total, ...
	Value      float64   `gorm:"not null"`                    // 统计指标的Value
	RecordDate time.Time `gorm:"not null;index:idx_key_date"` // 统计生成的时间，只记录年月日
}
