package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"server/global"
	"server/logger"
	"server/model"
	"strings"
	"time"

	"gorm.io/gorm"
)

type UserService struct {
	otpService   OTPService
	emailService *MailService
}

func NewUserServiceWithEmailService(emailService *MailService) *UserService {
	return &UserService{
		emailService: emailService,
	}
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

func (s *UserService) GetUserByEmail(email string) (*model.User, error) {
	ctx := context.Background()
	user, err := gorm.G[model.User](global.DB).Where("email = ?", email).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to query user by email %s: %w", email, err)
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

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *UserService) SendResetPassword(userEmail string, frontendUrl string) error {
	ctx := context.Background()
	// 验证邮箱，不存在也不报错，防止邮箱枚举攻击
	_, err := s.GetUserByEmail(userEmail)
	if err != nil {
		logger.Warnf("failed to verify email: %v", err)
		return nil
	}
	// 生成token用于重置密码url
	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %v", err)
	}

	redisKey := fmt.Sprintf("pwd_reset:%s", token)
	err = global.REDIS.Set(ctx, redisKey, userEmail, 15*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to save token to redis: %v", err)
	}

	resetUrl := fmt.Sprintf("%s?token=%s", frontendUrl, token)
	// 向用户邮箱发送密码重置邮件
	subject := "重置您的密码"
	content := fmt.Sprintf(`
		<h1>密码重置请求</h1>
		<p>您收到这封邮件是因为您请求重置密码。</p>
		<p>请点击下面的链接设置新密码（链接 15 分钟内有效）：</p>
		<a href="%s">%s</a>
		<p>如果您没有请求重置密码，请忽略此邮件。</p>`, resetUrl, resetUrl)
	logger.Infof("Sending email: %s", resetUrl)
	err = s.emailService.SendResetPassword(userEmail, subject, content)
	if err != nil {
		return fmt.Errorf("failed to send reset password email to user: %v", err)
	}

	return nil
}

func (s *UserService) ConfirmResetPassword(token string, newPassword string) error {
	ctx := context.Background()
	redisKey := fmt.Sprintf("pwd_reset:%s", token)
	email, err := global.REDIS.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return errors.New("not available")
		}
		return fmt.Errorf("failed to get email from redis: %v", err)
	}

	user, err := s.GetUserByEmail(email)
	if err != nil {
		return err
	}

	result := global.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).Update("pwd_hash", newPassword)
	if result.Error != nil {
		return fmt.Errorf("failed to update new password to user %s: %v", user.ID, result.Error)
	}

	if err := global.REDIS.Del(ctx, redisKey).Err(); err != nil {
		logger.Errorf("failed to delete token redis key %s: %v", redisKey, err)
	}

	return nil
}
