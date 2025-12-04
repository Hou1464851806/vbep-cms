package service

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"server/global"
	"server/model"
)

type UserService struct {
}

func (s *UserService) CreateUser(user *model.User) error {
	db := global.DB
	ctx := context.Background()
	result := gorm.WithResult()
	err := gorm.G[model.User](db, result).Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
