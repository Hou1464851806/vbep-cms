package model

type UserRole uint

// 角色权限枚举
const (
	Common UserRole = iota
	Admin
)

// User Model 用户表模型
type User struct {
	Base
	Role     UserRole `gorm:"not null;default:0"`                                   // 角色权限 RoleAdmin=1 RoleUser=0 default:RoleUser
	NickName string   `gorm:"not null"`                                             // 昵称
	Phone    *string  `gorm:"uniqueIndex:idx_unique_phone,where:phone IS NOT NULL"` // 手机号码：绑定后可以作为登录的验证，确保非空唯一
	Email    string   `gorm:"not null;unique"`                                      // 邮箱：作为注册时的主字段
	PWDHash  string   `gorm:"not null"`                                             // 密码哈希 bcrypt加密

	// 让gorm识别has many关系
	Datasets     []Dataset     `gorm:"foreignKey:CreatorID;references:ID"`
	Applications []Application `gorm:"foreignKey:ApplicantID;references:ID"`
}
