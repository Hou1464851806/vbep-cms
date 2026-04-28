package service

import (
	"server/pkg/apperr"

	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"server/global"
	"server/internal/tasks"
	"server/internal/templates"
	"server/logger"
	"server/model"
	"server/pkg/utils"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type UserService struct {
	otpService   *OTPService
	emailService *MailService
}

func NewUserServiceWithEmailService(emailService *MailService, otpService *OTPService) *UserService {
	return &UserService{
		otpService:   otpService,
		emailService: emailService,
	}
}

func (s *UserService) CreateUser(user *model.User) error {
	ctx := context.Background()
	result := gorm.WithResult()
	err := gorm.G[model.User](global.DB, result).Create(ctx, user)
	if err != nil {
		return apperr.New(err, "failed to create user")
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
			return nil, apperr.New(err, "failed to find user")
		}
	}
	//if password != user.PWDHash {
	if err := bcrypt.CompareHashAndPassword([]byte(user.PWDHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid user: %v", err)
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
		return nil, apperr.New(err, "failed to query user by id", "user_id", id)
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
		return nil, apperr.New(err, "failed to query user by email", "email", email)
	}

	return &user, nil
}

// UserListQuery 前端表格的筛选/排序/分页参数
type UserListQuery struct {
	Page      int
	PageSize  int
	SortBy    string // 前端列 prop，如 "nickname", "email", "createdAt", "role"
	Order     string // "asc" 或 "desc"
	Role      string // "" 表示不筛选，"0"/"1"/"2" 筛选具体角色
	Keyword   string // 模糊搜索（姓名 / 邮箱）
	IdPrefix  string // ID前缀匹配
	Nickname  string // 姓名精确/模糊匹配
	Email     string // 邮箱精确/模糊匹配
	DateStart string // "YYYY-MM-DD"
	DateEnd   string // "YYYY-MM-DD"
}

// userSortWhitelist 将前端 el-table prop 映射到安全的数据库列
var userSortWhitelist = map[string]string{
	"nickname":  "users.nick_name",
	"email":     "users.email",
	"createdAt": "users.created_at",
	"role":      "users.role",
}

func (s *UserService) GetAllUsers(q UserListQuery) ([]*model.User, int64, error) {
	ctx := context.Background()
	var users []*model.User
	var total int64

	baseQuery := global.DB.WithContext(ctx).Model(&model.User{}).Unscoped()
	// ── 角色筛选 ──
	if q.Role != "" {
		if r, err := strconv.Atoi(q.Role); err == nil {
			baseQuery = baseQuery.Where("users.role = ?", r)
		}
	}
	// ── 关键词搜索（姓名 / 邮箱 回退兼容）──
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		baseQuery = baseQuery.Where("users.nick_name LIKE ? OR users.email LIKE ?", kw, kw)
	}
	// ── 独立字段搜索 ──
	if q.IdPrefix != "" {
		idkw := q.IdPrefix + "%"
		baseQuery = baseQuery.Where("users.id LIKE ?", idkw)
	}
	if q.Nickname != "" {
		if len(q.Nickname) == 36 || (len(q.Nickname) >= 8 && utils.IsHex(q.Nickname)) {
			baseQuery = baseQuery.Where("users.nick_name LIKE ? OR users.id LIKE ?", "%"+q.Nickname+"%", q.Nickname+"%")
		} else {
			baseQuery = baseQuery.Where("users.nick_name LIKE ?", "%"+q.Nickname+"%")
		}
	}
	if q.Email != "" {
		if len(q.Email) == 36 || (len(q.Email) >= 8 && utils.IsHex(q.Email)) {
			baseQuery = baseQuery.Where("users.email LIKE ? OR users.id LIKE ?", "%"+q.Email+"%", q.Email+"%")
		} else {
			baseQuery = baseQuery.Where("users.email LIKE ?", "%"+q.Email+"%")
		}
	}
	// ── 日期范围（基于 created_at）──
	loc := time.Local
	if q.DateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateStart, loc); err == nil {
			baseQuery = baseQuery.Where("users.created_at >= ?", t)
		} else {
			logger.Errorw("Failed to parse DateStart. Value: ', Error", "datestart", q.DateStart, "error", err)
		}
	}
	if q.DateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateEnd, loc); err == nil {
			baseQuery = baseQuery.Where("users.created_at < ?", t.AddDate(0, 0, 1))
		} else {
			logger.Errorw("Failed to parse DateEnd. Value: ', Error", "dateend", q.DateEnd, "error", err)
		}
	}
	// ── 计数 ──
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, apperr.New(err, "failed to count users")
	}
	// ── 排序（白名单校验）──
	orderClause := "users.created_at DESC"
	if col, ok := userSortWhitelist[q.SortBy]; ok {
		dir := "ASC"
		if q.Order == "desc" {
			dir = "DESC"
		}
		orderClause = col + " " + dir
	}
	// ── 分页查询 ──
	offset := (q.Page - 1) * q.PageSize
	if err := baseQuery.
		Order(orderClause).
		Offset(offset).
		Limit(q.PageSize).
		Find(&users).Error; err != nil {
		return nil, 0, apperr.New(err, "failed to query users")
	}

	return users, total, nil
}

func (s *UserService) DeleteUser(id string) error {
	ctx := context.Background()
	// 直接使用 GORM 的 Delete 方法，它会自动软删除
	result := global.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.User{})

	if result.Error != nil {
		return apperr.New(result.Error, "failed to delete user")
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
		logger.Warnw("failed to verify email", "error", err)
		return nil
	}
	// 生成token用于重置密码url
	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %v", err)
	}

	redisKey := fmt.Sprintf("pwd_reset:%s", token)
	expireMinute := 15
	err = global.REDIS.Set(ctx, redisKey, userEmail, time.Duration(expireMinute)*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to save token to redis: %v", err)
	}

	resetUrl := fmt.Sprintf("%s?token=%s", frontendUrl, token)

	// 向用户邮箱发送密码重置邮件
	subject := "重置您的密码"
	content := templates.ResetPasswordContent{
		Link:       resetUrl,
		ExpireTime: expireMinute,
	}
	return s.emailService.SendNewMail(userEmail, subject, templates.ResetPasswordHtml, content, tasks.QueueCritical, 3)
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

	passwordHash, err := utils.PasswordToHash(newPassword)
	if err != nil {
		return err
	}
	result := global.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).Update("pwd_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("failed to update new password to user %s: %v", user.ID, result.Error)
	}

	if err := global.REDIS.Del(ctx, redisKey).Err(); err != nil {
		logger.Errorw("failed to delete token redis key", "rediskey", redisKey, "error", err)
	}

	return nil
}

func (s *UserService) GetUserFiles(userID string, page int, pageSize int) ([]model.File, int64, error) {
	var files []model.File
	var total int64
	ctx := context.Background()
	// 1. 构建基础查询，条件是 creator_id 匹配
	db := global.DB.WithContext(ctx).Model(&model.File{}).Where("creator_id = ?", userID)
	// 2. 先执行 Count 操作获取总数，用于分页
	if err := db.Count(&total).Error; err != nil {
		logger.Errorw("Failed to count files for user", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("database error while counting files")
	}
	// 如果总数为0，没必要继续查询，直接返回
	if total == 0 {
		return []model.File{}, 0, nil
	}
	// 3. 再执行 Find 操作，带上分页和排序
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		logger.Errorw("Failed to find files for user", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("database error while finding files")
	}
	// 4. 返回查询到的文件列表、总数和nil错误
	return files, total, nil
}
