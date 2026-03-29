package xiaohongshu

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type LoginAction struct {
	page *rod.Page
}

func NewLogin(page *rod.Page) *LoginAction {
	return &LoginAction{page: page}
}

// CheckLoginStatus 检查浏览器是否已有 web_session cookie（无需依赖 DOM 结构）。
func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	// 先导航一次，让浏览器加载当前 cookies
	pp := a.page.Context(ctx)
	if err := pp.Navigate("https://www.xiaohongshu.com/explore"); err != nil {
		return false, errors.Wrap(err, "navigate failed")
	}
	time.Sleep(1 * time.Second)

	cks, err := a.page.Browser().GetCookies()
	if err != nil {
		return false, errors.Wrap(err, "get cookies failed")
	}
	for _, c := range cks {
		if c.Name == "web_session" && c.Value != "" {
			return true, nil
		}
	}
	return false, nil
}

func (a *LoginAction) Login(ctx context.Context) error {
	pp := a.page.Context(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	// 等待一小段时间让页面完全加载
	time.Sleep(2 * time.Second)

	// 检查是否已经登录
	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		// 已经登录，直接返回
		return nil
	}

	// 等待扫码成功提示或者登录完成
	// 这里我们等待登录成功的元素出现，这样更简单可靠
	pp.MustElement(".main-container .user .link-wrapper .channel")

	return nil
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	// 导航阶段：独立 20s 超时
	navCtx, navCancel := context.WithTimeout(ctx, 20*time.Second)
	defer navCancel()

	pp := a.page.Context(navCtx)
	if err := pp.Navigate("https://www.xiaohongshu.com/explore"); err != nil {
		return "", false, errors.Wrap(err, "navigate failed")
	}
	if err := pp.WaitLoad(); err != nil {
		// WaitLoad 超时不致命，继续尝试
		logrus.Warnf("waitload timeout (non-fatal): %v", err)
	}
	time.Sleep(2 * time.Second)

	// 检查是否已登录
	pp2 := a.page.Context(ctx)
	if exists, _, _ := pp2.Has(".main-container .user .link-wrapper .channel"); exists {
		return "", true, nil
	}

	// 等待二维码元素：独立 45s 超时（JS 异步渲染需要时间）
	qrCtx, qrCancel := context.WithTimeout(ctx, 45*time.Second)
	defer qrCancel()

	pp3 := a.page.Context(qrCtx)
	el, err := pp3.Element(".qrcode-img")
	if err != nil {
		// 失败时截图，保存到 /tmp 方便诊断
		if img, serr := a.page.Screenshot(false, nil); serr == nil {
			path := fmt.Sprintf("/tmp/xhs-login-debug-%d.png", time.Now().Unix())
			_ = os.WriteFile(path, img, 0644)
			logrus.Warnf("qrcode element not found, screenshot saved to %s", path)
		}
		// 同时记录页面 HTML 前 2000 字节
		if body, herr := a.page.MustElement("body").HTML(); herr == nil && len(body) > 0 {
			preview := body
			if len(preview) > 2000 {
				preview = preview[:2000]
			}
			logrus.Warnf("page body preview: %s", preview)
		}
		return "", false, errors.Wrap(err, "qrcode element not found")
	}

	// 等待 src 填充（JS 异步写入 base64，元素出现时 src 可能还是空）
	var srcVal string
	for i := 0; i < 15; i++ {
		src, attrErr := el.Attribute("src")
		if attrErr == nil && src != nil && len(*src) > 30 {
			srcVal = *src
			break
		}
		time.Sleep(1 * time.Second)
	}
	if srcVal == "" {
		return "", false, errors.New("qrcode src is empty after waiting")
	}

	return srcVal, false, nil
}

// WaitForLogin 轮询浏览器 cookies，检测到 web_session 表示登录成功。
// 直接查 cookies 比检测 DOM 更可靠，不受页面结构变化影响。
func (a *LoginAction) WaitForLogin(ctx context.Context) bool {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			cks, err := a.page.Browser().GetCookies()
			if err != nil {
				continue
			}
			for _, c := range cks {
				if c.Name == "web_session" && c.Value != "" {
					return true
				}
			}
		}
	}
}
