package service

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"server/global"
	"server/model"
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

func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	ctx := context.Background()
	user, err := gorm.G[model.User](global.DB).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to query user by id %d: %w", id, err)
	}

	return &user, nil
}
