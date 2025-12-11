package service

import (
	"context"
	"errors"
	"fmt"
	"server/global"
	"server/model"

	"gorm.io/gorm"
)

type UserService struct {
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
	user, err := gorm.G[model.User](global.DB).Where("email = ?", info).Or("phone = ?", info).First(ctx)
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

func (s *UserService) DeleteUser(id string) error {
	// 使用 Unscoped() 可以执行物理删除，但在用户系统中通常建议使用软删除
	// 这里直接使用 GORM 的 Delete 方法，它会自动更新 deleted_at 字段
	result := global.DB.Delete(&model.User{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	// 可选：如果 id 不存在，result.RowsAffected 为 0
	// 根据业务需求，你可以选择报错或视为成功（幂等性）
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (s *UserService) ChangeUserRole(operator *model.User, targetUserID string, newRole model.UserRole) error {
	ctx := context.Background()
	// 规则一：只有 SuperAdmin 才有资格修改角色
	if operator.Role != model.SuperAdmin {
		return fmt.Errorf("permission denied: only SuperAdmin can change roles")
	}

	targetUser, err := s.GetUserByID(targetUserID)
	if err != nil {
		return err
	}

	if targetUser.Role == model.SuperAdmin && newRole != model.SuperAdmin {
		// 规则二：禁止自我降级 (防自杀)
		if operator.ID == targetUserID {
			return fmt.Errorf("operation failed: you cannot demote yourself")
		}

		// 规则三：最后一人原则 (可选，但在生产环境强烈建议)
		// var count int64
		// global.DB.Model(&model.User{}).Where("role = ?", model.SuperAdmin).Count(&count)
		count, err := gorm.G[model.User](global.DB).Where("role = ?", model.SuperAdmin).Count(ctx, "*")
		if err != nil {
			return fmt.Errorf("failed to count SuperAdmins")
		}
		if count <= 1 {
			return fmt.Errorf("operation failed: system must have at least one SuperAdmin")
		}
	}

	// targetUser.Role = newRole
	// if err := global.DB.Save(targetUser).Error; err != nil {
	// 	return fmt.Errorf("failed to update role: %w", err)
	// }
	count, err := gorm.G[model.User](global.DB).Where("id = ?", targetUser.ID).Update(ctx, "role", newRole)
	if count != 1 {
		return fmt.Errorf("operation failed: more than 1 user affected")
	}
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}
