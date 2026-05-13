package initialize

import (
	"server/pkg/apperr"

	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/utils"
	"strings"

	"gorm.io/gorm"
)

const (
	defaultRootEmailFilePath     = "/run/secrets/root_email"
	defaultRootPasswordFilePath  = "/app/secrets/root_initial_password.txt"
	defaultBootstrapRootNickname = "超级管理员"
	bootstrapRootEmailEnv        = "ROOT_EMAIL_FILE"
	bootstrapRootPasswordPathEnv = "ROOT_INITIAL_PASSWORD_FILE"
	bootstrapRootPasswordLength  = 16
)

func EnsureBootstrapSuperAdmin() error {
	var superAdminCount int64
	if err := global.DB.Model(&model.User{}).Where("role = ?", model.SuperAdmin).Count(&superAdminCount).Error; err != nil {
		return apperr.New(err, "failed to count super admins")
	}
	if superAdminCount > 0 {
		logger.Infow("Bootstrap super admin skipped: existing SuperAdmin found")
		return nil
	}

	email, err := readBootstrapRootEmail()
	if err != nil {
		return err
	}

	passwordPath := getEnvOrDefault(bootstrapRootPasswordPathEnv, defaultRootPasswordFilePath)
	passwordFileHadContent, err := prepareBootstrapPasswordOutputPath(passwordPath)
	if err != nil {
		return err
	}

	password, err := generateBootstrapPassword(bootstrapRootPasswordLength)
	if err != nil {
		return apperr.New(err, "failed to generate bootstrap super admin password")
	}
	passwordHash, err := utils.PasswordToHash(hashPasswordForClientProtocol(password))
	if err != nil {
		return apperr.New(err, "failed to hash bootstrap super admin password")
	}

	rootUser := model.User{
		Role:            model.SuperAdmin,
		NickName:        defaultBootstrapRootNickname,
		Email:           email,
		IsEmailVerified: true,
		PWDHash:         passwordHash,
	}
	if err := global.DB.Create(&rootUser).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || isUniqueConstraintError(err) {
			return fmt.Errorf("bootstrap super admin failed: root email %q is already occupied", email)
		}
		return apperr.New(err, "failed to create bootstrap super admin")
	}

	if err := os.WriteFile(passwordPath, []byte(password), 0600); err != nil {
		rollbackErr := global.DB.Unscoped().Delete(&model.User{}, "id = ?", rootUser.ID).Error
		if rollbackErr != nil {
			logger.Errorw("failed to rollback bootstrap super admin after password file write error", "user_id", rootUser.ID, "error", rollbackErr)
		}
		return apperr.New(err, "failed to persist bootstrap super admin password file", "passwordpath", passwordPath)
	}

	if passwordFileHadContent {
		logger.Warnw("Bootstrap password file contained stale content and was overwritten because no SuperAdmin existed in database", "passwordpath", passwordPath)
	}
	logger.Warnw("Bootstrap super admin created", "email", email, "initial_password_file", passwordPath)
	return nil
}

func readBootstrapRootEmail() (string, error) {
	emailPath := getEnvOrDefault(bootstrapRootEmailEnv, defaultRootEmailFilePath)
	content, err := os.ReadFile(emailPath)
	if err != nil {
		return "", apperr.New(err, "bootstrap super admin requires root email file", "email", emailPath)
	}

	email := strings.TrimSpace(string(content))
	if email == "" {
		return "", fmt.Errorf("bootstrap super admin requires non-empty root email in %q", emailPath)
	}
	return email, nil
}

func prepareBootstrapPasswordOutputPath(passwordPath string) (bool, error) {
	info, err := os.Stat(passwordPath)
	hadContent := false
	if err == nil {
		if info.IsDir() {
			return false, fmt.Errorf("bootstrap password path %q must be a file, not a directory", passwordPath)
		}

		content, readErr := os.ReadFile(passwordPath)
		if readErr != nil {
			return false, apperr.New(readErr, "failed to read bootstrap password file", "passwordpath", passwordPath)
		}
		hadContent = strings.TrimSpace(string(content)) != ""
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, apperr.New(err, "failed to inspect bootstrap password file", "passwordpath", passwordPath)
	}

	dir := filepath.Dir(passwordPath)
	if stat, statErr := os.Stat(dir); statErr != nil {
		return false, apperr.New(statErr, "bootstrap password directory is unavailable", "dir", dir)
	} else if !stat.IsDir() {
		return false, fmt.Errorf("bootstrap password directory %q is not a directory", dir)
	}

	return hadContent, nil
}

func generateBootstrapPassword(length int) (string, error) {
	if length < 6 || length > 20 {
		return "", fmt.Errorf("bootstrap password length %d is outside supported range", length)
	}

	charGroups := []string{
		"ABCDEFGHJKLMNPQRSTUVWXYZ",
		"abcdefghijkmnopqrstuvwxyz",
		"23456789",
		"!@#$%^&*_-+=",
	}
	allChars := strings.Join(charGroups, "")

	password := make([]byte, 0, length)
	for _, group := range charGroups {
		ch, err := randomCharFrom(group)
		if err != nil {
			return "", err
		}
		password = append(password, ch)
	}

	for len(password) < length {
		ch, err := randomCharFrom(allChars)
		if err != nil {
			return "", err
		}
		password = append(password, ch)
	}

	if err := shuffleBytes(password); err != nil {
		return "", err
	}

	return string(password), nil
}

func getEnvOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func isUniqueConstraintError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate key") ||
		strings.Contains(lower, "unique constraint") ||
		strings.Contains(lower, "unique failed")
}

func hashPasswordForClientProtocol(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func randomCharFrom(charset string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, err
	}
	return charset[n.Int64()], nil
}

func shuffleBytes(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		data[i], data[j] = data[j], data[i]
	}
	return nil
}
