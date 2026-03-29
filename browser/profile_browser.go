package browser

import (
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

// ProfileBrowser 使用持久化 Chrome profile 目录的浏览器。
// 相比 CDP cookie 注入，Chrome 原生 profile 能跨进程、跨子域正确保持会话。
type ProfileBrowser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

// NewProfileBrowser 创建带持久化 profile 目录的浏览器实例。
// profileDir 在浏览器关闭后保留，下次启动自动加载已有 session。
func NewProfileBrowser(headless bool, profileDir string, binPath string, proxy string) *ProfileBrowser {
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		logrus.Warnf("创建 profile 目录失败: %v", err)
	}

	l := launcher.New().
		Headless(headless).
		UserDataDir(profileDir).
		Set("--no-sandbox").
		Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	if binPath != "" {
		l = l.Bin(binPath)
	}
	if proxy != "" {
		l = l.Proxy(proxy)
	}

	url := l.MustLaunch()
	b := rod.New().ControlURL(url).MustConnect()

	return &ProfileBrowser{
		browser:  b,
		launcher: l,
	}
}

// NewPage 创建启用 stealth 模式的新页面
func (b *ProfileBrowser) NewPage() *rod.Page {
	return stealth.MustPage(b.browser)
}

// Close 关闭浏览器，等待 Chrome 进程完全退出后再返回。
// 必须等进程退出而非仅关闭 CDP 连接，否则 profile 数据可能尚未写入磁盘，
// 下一个使用同一 profileDir 的浏览器启动时会读到不完整的 session。
func (b *ProfileBrowser) Close() {
	pid := b.launcher.PID()
	_ = b.browser.Close() // 忽略 error（Chrome 关闭时 CDP 连接会断开）

	// 等待 Chrome 进程退出（检查 /proc/{pid}）
	waitProcessExit(pid)
}

// waitProcessExit 轮询 /proc/{pid} 直到 Chrome 进程退出（最多等 15 秒）。
// 超时后强制 SIGKILL，确保 profile SingletonLock 被释放。
func waitProcessExit(pid int) {
	if pid <= 0 {
		return
	}
	procPath := fmt.Sprintf("/proc/%d", pid)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(procPath); os.IsNotExist(err) {
			return // 进程已退出
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 超时：强制杀掉，否则下次启动同一 profile 会遇到 SingletonLock 冲突
	logrus.Warnf("Chrome 进程 %d 未在 15s 内退出，强制 SIGKILL", pid)
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}
