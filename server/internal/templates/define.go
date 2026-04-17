package templates

const (
	VerificationCodeHtml      = "verification_code.html"
	ResetPasswordHtml         = "reset_password.html"
	NewAppNotificationHtml    = "new_app_notification.html"
	AppChangeNotificationHtml = "app_change_notification.html"
)

type Data struct {
	Subject string // html 标题内容
	Content any    // html 正文内容
}

type VerificationCodeContent struct {
	Code       string // 验证码
	ExpireTime int    // 验证码过期时间
}

type ResetPasswordContent struct {
	Link       string // 重置密码链接
	ExpireTime int    // 链接过期时间
}

type NewAppNotificationContent struct {
	DatasetTitle string
	Url          string
}

type AppChangeNotificationContent struct {
	DatasetTitle string
	Status       string
	Url          string
}
