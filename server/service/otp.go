package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/mojocn/base64Captcha"
)

var (
	otpStore     sync.Map // map[string]otpData
	captchaStore = base64Captcha.DefaultMemStore
)

const ExpireMinute = 10

type otpData struct {
	Code      string
	ExpiresAt time.Time
}

type OTPService struct {
	emailService *MailService
}

func NewOPTService(emailService *MailService) *OTPService {
	return &OTPService{
		emailService: emailService,
	}
}

func (s *OTPService) GenerateCaptcha() (string, string) {
	// 切换为字符模式：高度80, 宽度240, 噪点2, 线条2, 长度4, 字符源(数字+字母)
	driver := base64Captcha.NewDriverString(80, 240, 2, 2, 4, "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", nil, nil, nil)
	cp := base64Captcha.NewCaptcha(driver, captchaStore)
	id, b64s, _, err := cp.Generate()
	if err != nil {
		return "", ""
	}
	return id, b64s
}

func (s *OTPService) VerifyCaptcha(id, answer string) bool {
	// 获取存储的验证码答案（false表示先不删除）
	val := captchaStore.Get(id, false)
	if val == "" {
		return false
	}
	captchaStore.Get(id, true)            // 立即删除，防止重放
	return strings.EqualFold(val, answer) // 不区分大小写比较
}

func (s *OTPService) SendOTP(target string) error {
	// Generate 6-digit secure random code
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", n.Int64())

	otpStore.Store(target, otpData{
		Code:      code,
		ExpiresAt: time.Now().Add(time.Duration(ExpireMinute) * time.Minute),
	})

	return s.emailService.SendVerificationCode(target, "您的账户注册验证码", code, ExpireMinute)
}

func (s *OTPService) VerifyOTP(target, code string) bool {
	val, ok := otpStore.Load(target)
	if !ok {
		return false
	}
	data := val.(otpData)
	if time.Now().After(data.ExpiresAt) {
		otpStore.Delete(target)
		return false
	}
	if data.Code == code {
		otpStore.Delete(target)
		return true
	}
	return false
}
