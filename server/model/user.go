package model

type UserRole uint

// 角色权限枚举
const (
	Common UserRole = iota
	Admin
	SuperAdmin
)

type User struct {
	Base
	Role            UserRole      `gorm:"not null;default:0"`
	NickName        string        `gorm:"not null"`
	Email           string        `gorm:"uniqueIndex:idx_unique_email;not null"`
	IsEmailVerified bool          `gorm:"default:false"`
	PWDHash         string        `gorm:"not null"` // 密码哈希 bcrypt加密
	Datasets        []Dataset     `gorm:"foreignKey:CreatorID;references:ID"`
	Applications    []Application `gorm:"foreignKey:ApplicantID;references:ID"`
}

type UserBrief struct {
	ID        string `json:"id"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	IsDeleted bool   `json:"isDeleted"`
}

func (u User) ToBrief() UserBrief {
	return UserBrief{
		ID:        u.ID,
		Nickname:  u.NickName,
		Email:     u.Email,
		IsDeleted: u.DeletedAt.Valid,
	}
}
