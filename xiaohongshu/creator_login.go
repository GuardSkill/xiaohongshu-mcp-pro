package xiaohongshu

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
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

// VerifyOTP 填写验证码并提交登录。
// 返回值：(安全验证二维码截图, error)
// 若小红书弹出"安全验证扫码"弹窗，会将截图返回给调用方展示给用户，
// 同时在后台等待最多 120 秒直到 web_session 出现（用户扫码后写入）。
func (a *CreatorLoginAction) VerifyOTP(otp string) ([]byte, error) {
	pp := a.page.Timeout(10 * time.Second)

	// 找到验证码输入框并用 rod 模拟真实键盘输入，确保触发 Vue 响应式
	otpInput, err := pp.Element("input[placeholder*='验证码']")
	if err != nil {
		return nil, errors.New("未找到验证码输入框")
	}
	if err := otpInput.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, errors.Wrap(err, "点击验证码输入框失败")
	}
	// 清空已有内容，再输入验证码，确保 Vue 响应式绑定生效
	if err := otpInput.SelectAllText(); err != nil {
		logrus.Warnf("SelectAllText failed (non-fatal): %v", err)
	}
	if err := otpInput.Input(otp); err != nil {
		return nil, errors.Wrap(err, "输入验证码失败")
	}
	time.Sleep(1 * time.Second)

	if shot, e := a.page.Screenshot(false, nil); e == nil {
		saveDebugShot("creator-before-login-click", shot)
	}

	// 等待登录按钮变为可点击（Vue 验证通过），最多 5 秒
	loginBtn, err := pp.Element(".beer-login-btn")
	if err != nil {
		return nil, errors.New("未找到登录按钮")
	}
	for i := 0; i < 10; i++ {
		d, _ := loginBtn.Attribute("disabled")
		if d == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err := loginBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, errors.Wrap(err, "点击登录按钮失败")
	}
	time.Sleep(3 * time.Second)

	if shot, e := a.page.Screenshot(false, nil); e == nil {
		saveDebugShot("creator-after-login-click", shot)
	}

	// 以 creator 页面离开 /login 为登录成功的信号（比 web_session 更准确）
	if a.creatorLoginDone() {
		return nil, nil
	}

	// 仍在 /login 页面：说明弹出了安全验证二维码，截图后等待用户扫码（最多 120 秒）
	shot, _ := a.page.Screenshot(false, nil)
	logrus.Infof("检测到安全验证弹窗，等待用户扫码（最多 120 秒）")
	for i := 0; i < 60; i++ {
		time.Sleep(2 * time.Second)
		if a.creatorLoginDone() {
			logrus.Infof("扫码验证完成，creator 登录成功")
			return shot, nil
		}
	}
	return nil, errors.New("安全验证超时（120 秒），请重新登录")
}

// creatorLoginDone 检查 creator 页面是否已离开 /login（登录完成的信号）
func (a *CreatorLoginAction) creatorLoginDone() bool {
	info, err := a.page.Info()
	if err != nil {
		return false
	}
	done := !strings.Contains(info.URL, "/login")
	if done {
		logrus.Infof("creator 登录完成，当前 URL: %s", info.URL)
	}
	return done
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
