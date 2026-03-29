package browser

import (
	"net/url"
	"os"

	"github.com/lisiyuan/xiaohongshu-mcp-pro/cookies"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
)

type browserConfig struct {
	binPath    string
	cookiePath string
}

type Option func(*browserConfig)

func WithBinPath(binPath string) Option {
	return func(c *browserConfig) {
		c.binPath = binPath
	}
}

// WithCookiePath 指定 cookies 文件路径（覆盖默认路径）
func WithCookiePath(path string) Option {
	return func(c *browserConfig) {
		c.cookiePath = path
	}
}

// maskProxyCredentials masks username and password in proxy URL for safe logging.
func maskProxyCredentials(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil || u.User == nil {
		return proxyURL
	}
	if _, hasPassword := u.User.Password(); hasPassword {
		u.User = url.UserPassword("***", "***")
	} else {
		u.User = url.User("***")
	}
	return u.String()
}

func NewBrowser(headless bool, options ...Option) *headless_browser.Browser {
	cfg := &browserConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	opts := []headless_browser.Option{
		headless_browser.WithHeadless(headless),
	}
	if cfg.binPath != "" {
		opts = append(opts, headless_browser.WithChromeBinPath(cfg.binPath))
	}

	// Read proxy from environment variable
	if proxy := os.Getenv("XHS_PROXY"); proxy != "" {
		opts = append(opts, headless_browser.WithProxy(proxy))
		logrus.Infof("Using proxy: %s", maskProxyCredentials(proxy))
	}

	// 加载 cookies（优先使用显式指定的路径）
	cookiePath := cfg.cookiePath
	if cookiePath == "" {
		cookiePath = cookies.GetCookiesFilePath()
	}
	cookieLoader := cookies.NewLoadCookie(cookiePath)

	if data, err := cookieLoader.LoadCookies(); err == nil {
		opts = append(opts, headless_browser.WithCookies(string(data)))
		logrus.Debugf("loaded cookies from filesuccessfully")
	} else {
		logrus.Warnf("failed to load cookies: %v", err)
	}

	return headless_browser.New(opts...)
}
