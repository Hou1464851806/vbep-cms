package model

type UserRole uint

// 角色权限枚举
const (
	Common UserRole = iota
	Admin
	SuperAdmin
)

// User Model 用户表模型
type User struct {
	Base
	Role     UserRole `gorm:"not null;default:0"` // 角色权限
	NickName string   `gorm:"not null"`           // 昵称
	Phone    *string  `gorm:"uniqueIndex:idx_unique_phone"`
	Email    *string  `gorm:"uniqueIndex:idx_unique_email"`
	PWDHash  string   `gorm:"not null"` // 密码哈希 bcrypt加密

	// 让gorm识别has many关系
	Datasets     []Dataset     `gorm:"foreignKey:CreatorID;references:ID"`
	Applications []Application `gorm:"foreignKey:ApplicantID;references:ID"`
}
