package initialize

import (
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
		return fmt.Errorf("failed to count super admins: %w", err)
	}
	if superAdminCount > 0 {
		logger.Infof("Bootstrap super admin skipped: existing SuperAdmin found")
		return nil
	}

	email, err := readBootstrapRootEmail()
	if err != nil {
		return err
	}

	passwordPath := getEnvOrDefault(bootstrapRootPasswordPathEnv, defaultRootPasswordFilePath)
	if err := validateBootstrapPasswordOutputPath(passwordPath); err != nil {
		return err
	}

	password, err := generateBootstrapPassword(bootstrapRootPasswordLength)
	if err != nil {
		return fmt.Errorf("failed to generate bootstrap super admin password: %w", err)
	}
	passwordHash, err := utils.PasswordToHash(hashPasswordForClientProtocol(password))
	if err != nil {
		return fmt.Errorf("failed to hash bootstrap super admin password: %w", err)
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
		return fmt.Errorf("failed to create bootstrap super admin: %w", err)
	}

	if err := os.WriteFile(passwordPath, []byte(password), 0600); err != nil {
		rollbackErr := global.DB.Unscoped().Delete(&model.User{}, "id = ?", rootUser.ID).Error
		if rollbackErr != nil {
			logger.Errorf("failed to rollback bootstrap super admin %s after password file write error: %v", rootUser.ID, rollbackErr)
		}
		return fmt.Errorf("failed to persist bootstrap super admin password file %q: %w", passwordPath, err)
	}

	logger.Warnf("Bootstrap super admin created: email=%s initial_password_file=%s", email, passwordPath)
	return nil
}

func readBootstrapRootEmail() (string, error) {
	emailPath := getEnvOrDefault(bootstrapRootEmailEnv, defaultRootEmailFilePath)
	content, err := os.ReadFile(emailPath)
	if err != nil {
		return "", fmt.Errorf("bootstrap super admin requires root email file %q: %w", emailPath, err)
	}

	email := strings.TrimSpace(string(content))
	if email == "" {
		return "", fmt.Errorf("bootstrap super admin requires non-empty root email in %q", emailPath)
	}
	return email, nil
}

func validateBootstrapPasswordOutputPath(passwordPath string) error {
	info, err := os.Stat(passwordPath)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("bootstrap password path %q must be a file, not a directory", passwordPath)
		}

		content, readErr := os.ReadFile(passwordPath)
		if readErr != nil {
			return fmt.Errorf("failed to read bootstrap password file %q: %w", passwordPath, readErr)
		}
		if strings.TrimSpace(string(content)) != "" {
			return fmt.Errorf("bootstrap password file %q already contains data; refusing to overwrite", passwordPath)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect bootstrap password file %q: %w", passwordPath, err)
	}

	dir := filepath.Dir(passwordPath)
	if stat, statErr := os.Stat(dir); statErr != nil {
		return fmt.Errorf("bootstrap password directory %q is unavailable: %w", dir, statErr)
	} else if !stat.IsDir() {
		return fmt.Errorf("bootstrap password directory %q is not a directory", dir)
	}

	return nil
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
