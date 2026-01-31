package model

type Statistic struct {
	Base
	User        UserStats        `gorm:"embedded" json:"user"`
	Dataset     DatasetStats     `gorm:"embedded" json:"dataset"`
	Application ApplicationStats `gorm:"embedded" json:"application"`
	Audit       AuditStats       `gorm:"embedded" json:"audit"`
	File        FileStats        `gorm:"embedded" json:"file"`
	Storage     StorageStats     `gorm:"embedded" json:"storage"`
}

type UserStats struct {
	TotalUsers      int64 `gorm:"not null" json:"totalUsers"`      // 注册用户数
	CommonUsers     int64 `gorm:"not null" json:"commonUsers"`     // 普通用户数
	AdminUsers      int64 `gorm:"not null" json:"adminUsers"`      // 管理员数
	SuperAdminUsers int64 `gorm:"not null" json:"superAdminUsers"` // 超管数
}

type DatasetStats struct {
	TotalDatasets int64 `gorm:"not null" json:"totalDatasets"` // 数据集数量
}

type ApplicationStats struct {
	TotalApplications    int64 `gorm:"not null" json:"totalApplications"`    // 总申请数量
	ApprovedApplications int64 `gorm:"not null" json:"approvedApplications"` // 通过的申请量
	PendingApplications  int64 `gorm:"not null" json:"pendingApplications"`  // 待审核/未提交
	RejectedApplications int64 `gorm:"not null" json:"rejectedApplications"` // 已拒绝
	ReturnedApplications int64 `gorm:"not null" json:"returnedApplications"` // 已退回
}

type AuditStats struct {
	Submissions   int64 `gorm:"not null" json:"submissions"`   // 总申请提交次数
	AuditApproved int64 `gorm:"not null" json:"auditApproved"` // 审批通过次数 (Audit)
	AuditRejected int64 `gorm:"not null" json:"auditRejected"` // 审批拒绝次数 (Audit)
	AuditReturned int64 `gorm:"not null" json:"auditReturned"` // 审批退回次数 (Audit)
	AuditPending  int64 `gorm:"not null" json:"auditPending"`  // 待审核次数 (Audit)
}

type FileStats struct {
	Files            int64 `gorm:"not null;default:0" json:"files"`            // 文件总数 (全量)
	DatasetFiles     int64 `gorm:"not null;default:0" json:"datasetFiles"`     // 数据集文件总数
	PublicFiles      int64 `gorm:"not null;default:0" json:"publicFiles"`      // 公开下载文件总数
	ApplicationFiles int64 `gorm:"not null;default:0" json:"applicationFiles"` // 申请附件总数
}

type StorageStats struct {
	TotalFileStorage       int64 `gorm:"not null;default:0" json:"totalFileStorage"`       // 所有文件所占空间
	PublicFileStorage      int64 `gorm:"not null;default:0" json:"publicFileStorage"`      // 公开下载文件所占空间
	DatasetFileStorage     int64 `gorm:"not null;default:0" json:"datasetFileStorage"`     // 数据集文件所占空间
	ApplicationFileStorage int64 `gorm:"not null;default:0" json:"applicationFileStorage"` // 申请附件所占空间
}
