package service

import (
	"context"
	"errors"
	"fmt"
	"server/global"
	"server/model"
	"strings"

	"gorm.io/gorm"
)

type UserService struct {
	otpService OTPService
}

func (s *UserService) CreateUser(user *model.User) error {
	ctx := context.Background()
	result := gorm.WithResult()
	err := gorm.G[model.User](global.DB, result).Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s *UserService) FindAndCheckUser(info string, password string) (*model.User, error) {
	ctx := context.Background()
	user, err := gorm.G[model.User](global.DB).Where("email = ?", info).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to find user (info = %s)", info)
		} else {
			return nil, fmt.Errorf("failed to find user: %w", err)
		}
	}
	if password != user.PWDHash {
		return nil, fmt.Errorf("invalid user")
	}

	return &user, nil
}

func (s *UserService) GetUserByID(id string) (*model.User, error) {
	ctx := context.Background()
	user, err := gorm.G[model.User](global.DB).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to query user by id %s: %w", id, err)
	}

	return &user, nil
}

func (s *UserService) GetAllUsers() ([]*model.User, error) {
	ctx := context.Background()
	users, err := gorm.G[*model.User](global.DB).Find(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserService) DeleteUser(id string) error {
	ctx := context.Background()
	// 直接使用 GORM 的 Delete 方法，它会自动软删除
	result := global.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.User{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	} else if result.RowsAffected > 1 {
		return fmt.Errorf("more than 1 user affected")
	}

	return nil
}

func (s *UserService) UpdateUser(operator *model.User, targetUserID string, updates map[string]interface{}) error {
	ctx := context.Background()

	// 1. 查目标用户
	var targetUser model.User
	if err := global.DB.First(&targetUser, "id = ?", targetUserID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// 2. 角色变更的安全检查
	// 只要 updates 里有 role，说明 API 层已经确认操作者是 SuperAdmin 了
	if newRoleVal, ok := updates["role"]; ok {
		newRole := newRoleVal.(model.UserRole)

		// 规则一：防止自我降级
		// 如果 target 就是操作者自己，且试图取消 SuperAdmin 身份
		if operator.ID == targetUser.ID {
			if targetUser.Role == model.SuperAdmin && newRole != model.SuperAdmin {
				return fmt.Errorf("operation failed: you cannot demote yourself")
			}
		}

		// 规则二：最后一人原则
		if targetUser.Role == model.SuperAdmin && newRole != model.SuperAdmin {
			count, err := gorm.G[model.User](global.DB).Where("role = ?", model.SuperAdmin).Count(ctx, "*")
			if err != nil {
				return fmt.Errorf("failed to count admins")
			}
			if count <= 1 {
				return fmt.Errorf("operation failed: system must have at least one SuperAdmin")
			}
		}
	}

	// 3. 邮箱 约束检查
	// Email is now mandatory (string, not null), so we don't need to check if both are nil.
	// The API layer ensures that if email is updated, it is not empty.

	// 4. 执行更新
	result := global.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", targetUserID).Updates(updates)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") {
			return fmt.Errorf("email already occupied")
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	} else if result.RowsAffected > 1 {
		return fmt.Errorf("more than 1 user affected")
	}

	return nil
}

func (s *UserService) VerifyOTP(target, code string) bool {
	return s.otpService.VerifyOTP(target, code)
}

func (s *UserService) GenerateCaptcha() (string, string) {
	return s.otpService.GenerateCaptcha()
}

func (s *UserService) VerifyCaptcha(id, answer string) bool {
	return s.otpService.VerifyCaptcha(id, answer)
}

func (s *UserService) SendOTP(target string) error {
	return s.otpService.SendOTP(target)
}
