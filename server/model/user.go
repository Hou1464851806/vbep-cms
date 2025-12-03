package model

// 角色权限枚举
const (
	RoleAdmin uint = 1
	RoleUser  uint = 0
)

// User Model 用户表模型
type User struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"` // 用户ID 自增主键
	Role     uint   `gorm:"not null;default:0"`       // 角色权限 RoleAdmin=1 RoleUser=0 default:RoleUser
	NickName string `gorm:"not null"`                 // 昵称
	Phone    string `gorm:"not null"`                 // 手机号码
	Email    string `gorm:"not null"`                 // 邮箱
	PWDHash  string `gorm:"not null"`                 // 密码哈希
	Time     Base   `gorm:"embedded"`                 // 时间信息 展开嵌套的字段
}
