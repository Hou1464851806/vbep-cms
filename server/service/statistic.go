package service

import (
	"errors"
	"server/global"
	"server/model"
	"time"

	"gorm.io/gorm"
)

type StatisticService struct{}

func NewStatisticService() *StatisticService {
	return &StatisticService{}
}

// GenerateStats 计算并持久化统计数据
// 读取 config 中的 interval 和 offset
// 该函数应由 Cron 定时任务调用
func (s *StatisticService) GenerateStats() error {
	db := global.DB
	now := time.Now()

	// 读取配置并设置默认值 (24h, 8h)
	intervalHours := global.CFG.Cron.Interval
	if intervalHours <= 0 {
		intervalHours = 24
	}
	offsetHours := global.CFG.Cron.Offset
	if offsetHours == 0 {
		offsetHours = 8
	}

	interval := time.Duration(intervalHours) * time.Hour
	offset := time.Duration(offsetHours) * time.Hour

	// 计算当前统计窗口的起始时间
	// 逻辑：先加上偏移量对齐到"逻辑零时区"，取整后再减去偏移量还原
	windowStart := now.Add(offset).Truncate(interval).Add(-offset)

	var stats model.Statistic

	// 1. 用户统计
	if err := db.Model(&model.User{}).Count(&stats.User.TotalUsers).Error; err != nil {
		return err
	}
	if err := db.Model(&model.User{}).Where("role = ?", model.Common).Count(&stats.User.CommonUsers).Error; err != nil {
		return err
	}
	if err := db.Model(&model.User{}).Where("role = ?", model.Admin).Count(&stats.User.AdminUsers).Error; err != nil {
		return err
	}
	if err := db.Model(&model.User{}).Where("role = ?", model.SuperAdmin).Count(&stats.User.SuperAdminUsers).Error; err != nil {
		return err
	}

	// 2. 数据集统计
	if err := db.Model(&model.Dataset{}).Count(&stats.Dataset.TotalDatasets).Error; err != nil {
		return err
	}

	// 3. 申请统计
	if err := db.Model(&model.Application{}).Count(&stats.Application.TotalApplications).Error; err != nil {
		return err
	}

	// 4. 提交次数统计
	if err := db.Model(&model.Audit{}).Count(&stats.Audit.TotalAudits).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Audit{}).Where("action = ?", model.PENDING).Count(&stats.Audit.PendingAudits).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Audit{}).Where("action = ?", model.APPROVED).Count(&stats.Audit.ApprovedAudits).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Audit{}).Where("action = ?", model.REJECTED).Count(&stats.Audit.RejectedAudits).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Audit{}).Where("action = ?", model.RETURNED).Count(&stats.Audit.ReturnedAudits).Error; err != nil {
		return err
	}

	// 5. 通过的申请量
	if err := db.Model(&model.Application{}).Where("status = ?", model.APPROVED).Count(&stats.Application.ApprovedApplications).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Application{}).Where("status = ?", model.PENDING).Count(&stats.Application.PendingApplications).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Application{}).Where("status = ?", model.REJECTED).Count(&stats.Application.RejectedApplications).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Application{}).Where("status = ?", model.RETURNED).Count(&stats.Application.ReturnedApplications).Error; err != nil {
		return err
	}

	// 统计各单位申请数量 (用于词云)
	var affiliationCounts []struct {
		Affiliation string
		Count       int64
	}
	if err := db.Model(&model.Application{}).Select("affiliation, count(*) as count").Group("affiliation").Scan(&affiliationCounts).Error; err != nil {
		return err
	}
	stats.Application.AffiliationDistribution = make(map[string]int64)
	for _, u := range affiliationCounts {
		if u.Affiliation != "" {
			stats.Application.AffiliationDistribution[u.Affiliation] = u.Count
		}
	}

	// 6. 文件统计
	if err := db.Model(&model.File{}).Count(&stats.File.TotalFiles).Error; err != nil {
		return err
	}
	if err := db.Model(&model.File{}).Where("owner_type = ?", datasetOwnerType).Count(&stats.File.DatasetFiles).Error; err != nil {
		return err
	}
	if err := db.Model(&model.File{}).Where("owner_type = ?", publicOwnerType).Count(&stats.File.PublicFiles).Error; err != nil {
		return err
	}
	if err := db.Model(&model.File{}).Where("owner_type = ?", applicationOwnerType).Count(&stats.File.ApplicationFiles).Error; err != nil {
		return err
	}

	// 7. 文件存储统计
	// 总空间
	if err := db.Model(&model.File{}).Select("COALESCE(SUM(size), 0)").Scan(&stats.Storage.TotalFileStorage).Error; err != nil {
		return err
	}
	// 公开文件
	if err := db.Model(&model.File{}).Where("owner_type = ?", publicOwnerType).Select("COALESCE(SUM(size), 0)").Scan(&stats.Storage.PublicFileStorage).Error; err != nil {
		return err
	}
	// 数据集文件
	if err := db.Model(&model.File{}).Where("owner_type = ?", datasetOwnerType).Select("COALESCE(SUM(size), 0)").Scan(&stats.Storage.DatasetFileStorage).Error; err != nil {
		return err
	}
	// 申请附件
	if err := db.Model(&model.File{}).Where("owner_type = ?", applicationOwnerType).Select("COALESCE(SUM(size), 0)").Scan(&stats.Storage.ApplicationFileStorage).Error; err != nil {
		return err
	}

	// 7. 持久化 (如果当前时间窗口内已存在记录则更新，不存在则创建)
	var existing model.Statistic
	// 查找 created_at 在当前窗口起始时间之后的最新一条记录
	err := db.Where("created_at >= ?", windowStart).Order("created_at desc").First(&existing).Error
	if err == nil {
		stats.ID = existing.ID
		stats.CreatedAt = existing.CreatedAt // 更新时：保持原有的创建时间不变
		return db.Save(&stats).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		stats.CreatedAt = windowStart // 创建时：强制使用计算出的窗口起始时间（如 00:00:00），而非当前执行时间
		return db.Create(&stats).Error
	}
	return err
}

// GetLatestStats 获取最新的统计数据（直接读取 Statistic 表）
func (s *StatisticService) GetLatestStats() ([]model.Statistic, error) {
	var stats []model.Statistic
	// 获取最近的 10 条数据 (基于 CreatedAt 倒序，下标 0 为最新)
	if err := global.DB.Order("created_at desc").Limit(10).Find(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}
