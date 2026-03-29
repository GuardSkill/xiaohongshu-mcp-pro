package xiaohongshu

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const creatorLoginURL = "https://creator.xiaohongshu.com/login"

// CreatorLoginAction creator 手机号登录流程
type CreatorLoginAction struct {
	page *rod.Page
}

func NewCreatorLogin(page *rod.Page) *CreatorLoginAction {
	return &CreatorLoginAction{page: page}
}

// NavigateToLogin 导航到 creator 登录页，截图返回给用户确认
func (a *CreatorLoginAction) NavigateToLogin() ([]byte, error) {
	pp := a.page.Timeout(20 * time.Second)
	if err := pp.Navigate(creatorLoginURL); err != nil {
		return nil, errors.Wrap(err, "导航到 creator 登录页失败")
	}
	if err := pp.WaitLoad(); err != nil {
		logrus.Warnf("creator 登录页加载超时（non-fatal）: %v", err)
	}
	time.Sleep(2 * time.Second)
	return pp.Screenshot(false, nil)
}

// SendOTP 填写手机号并点击发送验证码，返回截图供用户确认
func (a *CreatorLoginAction) SendOTP(phone string) ([]byte, error) {
	pp := a.page.Timeout(15 * time.Second)

	// 用 JS native setter 设置手机号，确保触发 Vue 响应式事件
	// creator 登录页的 input 没有 type="tel"，且普通 Input() 不触发 Vue 的双向绑定
	set, err := a.page.Eval(fmt.Sprintf(`() => {
		const inp = document.querySelector("input[placeholder*='手机']");
		if (!inp) return false;
		const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
		setter.call(inp, %q);
		inp.dispatchEvent(new Event('input',  { bubbles: true }));
		inp.dispatchEvent(new Event('change', { bubbles: true }));
		inp.dispatchEvent(new Event('blur',   { bubbles: true }));
		return true;
	}`, phone))
	if err != nil || !set.Value.Bool() {
		shot, _ := pp.Screenshot(false, nil)
		saveDebugShot("creator-login-no-phone-input", shot)
		return nil, errors.New("未找到手机号输入框")
	}
	time.Sleep(1 * time.Second) // 等待 Vue 响应式更新按钮状态

	// 用 JS 匹配直接文本节点为"发送验证码"的元素并点击
	// 避免匹配到包含该文字的父容器
	clicked, err := a.page.Eval(`() => {
		for (const el of document.querySelectorAll('*')) {
			const direct = Array.from(el.childNodes)
				.filter(n => n.nodeType === 3)
				.map(n => n.textContent.trim())
				.join('');
			if (direct === '发送验证码') {
				el.click();
				return true;
			}
		}
		return false;
	}`)
	if err != nil || !clicked.Value.Bool() {
		shot, _ := pp.Screenshot(false, nil)
		saveDebugShot("creator-login-no-otp-btn", shot)
		return nil, errors.New("未找到发送验证码按钮")
	}
	time.Sleep(2 * time.Second)

	return pp.Screenshot(false, nil)
}

// VerifyOTP 填写验证码并提交登录
func (a *CreatorLoginAction) VerifyOTP(otp string) error {
	// 用 JS native setter 填验证码，确保触发 Vue 响应式
	set, err := a.page.Eval(fmt.Sprintf(`() => {
		const inp = document.querySelector("input[placeholder*='验证码']");
		if (!inp) return false;
		const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
		setter.call(inp, %q);
		inp.dispatchEvent(new Event('input',  { bubbles: true }));
		inp.dispatchEvent(new Event('change', { bubbles: true }));
		inp.dispatchEvent(new Event('blur',   { bubbles: true }));
		return true;
	}`, otp))
	if err != nil || !set.Value.Bool() {
		return errors.New("未找到验证码输入框")
	}
	time.Sleep(500 * time.Millisecond)

	// 截图：点击前的页面状态（用于调试）
	if shot, e := a.page.Screenshot(false, nil); e == nil {
		saveDebugShot("creator-before-login-click", shot)
	}

	// 检查登录按钮是否可点击；若 disabled 则先用 JS 强制移除限制
	pp := a.page.Timeout(10 * time.Second)
	loginBtn, err := pp.Element(".beer-login-btn")
	if err != nil {
		return errors.New("未找到登录按钮")
	}
	// 用 JS 强制触发点击，绕过 disabled 限制
	_, _ = a.page.Eval(`() => {
		const btn = document.querySelector(".beer-login-btn");
		if (btn) { btn.removeAttribute("disabled"); btn.click(); }
	}`)
	time.Sleep(5 * time.Second)

	// 截图：点击后的页面状态
	if shot, e := a.page.Screenshot(false, nil); e == nil {
		saveDebugShot("creator-after-login-click", shot)
	}

	_ = loginBtn // 已通过 JS 点击，rod Click 仅作备用
	// 检查是否登录成功（不再在 /login 页面）
	info, err := pp.Info()
	if err != nil {
		return errors.Wrap(err, "获取页面信息失败")
	}
	if strings.Contains(info.URL, "/login") {
		return errors.New("验证码错误或登录失败，仍在登录页")
	}
	logrus.Infof("creator 登录成功，当前 URL: %s", info.URL)
	return nil
}

// CheckCreatorLoginStatus 检查 creator 是否已登录
func (a *CreatorLoginAction) CheckCreatorLoginStatus() bool {
	pp := a.page.Timeout(15 * time.Second)
	if err := pp.Navigate("https://creator.xiaohongshu.com"); err != nil {
		return false
	}
	if err := pp.WaitLoad(); err != nil {
		logrus.Warnf("creator 首页加载超时（non-fatal）: %v", err)
	}
	time.Sleep(2 * time.Second)
	info, err := pp.Info()
	if err != nil {
		return false
	}
	return !strings.Contains(info.URL, "/login")
}

func saveDebugShot(name string, shot []byte) {
	if shot == nil {
		return
	}
	path := fmt.Sprintf("/tmp/xhs-%s-%d.png", name, time.Now().Unix())
	_ = os.WriteFile(path, shot, 0644)
	logrus.Infof("调试截图已保存: %s", path)
}
