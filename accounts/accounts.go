package accounts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultAccountName = "default"

// Account 账号信息
type Account struct {
	Name        string    `json:"name"`
	CookiesFile string    `json:"cookies_file"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
}

// accountsData 账号数据文件结构
type accountsData struct {
	Active   string              `json:"active"`
	Accounts map[string]*Account `json:"accounts"`
}

// Manager 账号管理器（单例）
type Manager struct {
	mu       sync.RWMutex
	filePath string
	data     *accountsData
}

var (
	globalManager *Manager
	once          sync.Once
)

// GetManager 获取全局账号管理器单例
func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager(getAccountsFilePath())
	})
	return globalManager
}

func getAccountsFilePath() string {
	if path := os.Getenv("ACCOUNTS_PATH"); path != "" {
		return path
	}
	return "accounts.json"
}

func newManager(filePath string) *Manager {
	m := &Manager{filePath: filePath}
	if err := m.load(); err != nil {
		// 文件不存在或解析失败，初始化默认数据
		m.data = m.initDefault()
	}
	return m
}

// initDefault 初始化默认账号（兼容已有 cookies.json）
func (m *Manager) initDefault() *accountsData {
	return &accountsData{
		Active: defaultAccountName,
		Accounts: map[string]*Account{
			defaultAccountName: {
				Name:        defaultAccountName,
				CookiesFile: getDefaultCookiePath(),
				CreatedAt:   time.Now(),
				LastUsed:    time.Now(),
			},
		},
	}
}

// getDefaultCookiePath 向后兼容获取默认 cookie 路径
func getDefaultCookiePath() string {
	oldPath := filepath.Join(os.TempDir(), "cookies.json")
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}
	if path := os.Getenv("COOKIES_PATH"); path != "" {
		return path
	}
	return "cookies.json"
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}
	var d accountsData
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if d.Accounts == nil {
		d.Accounts = make(map[string]*Account)
	}
	m.data = &d
	return nil
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}

// GetActiveCookiesPath 获取当前激活账号的 cookies 文件路径
func (m *Manager) GetActiveCookiesPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	acc, ok := m.data.Accounts[m.data.Active]
	if !ok {
		return getDefaultCookiePath()
	}
	return acc.CookiesFile
}

// GetActiveAccountName 获取当前激活账号名
func (m *Manager) GetActiveAccountName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data.Active
}

// SwitchAccount 切换当前激活账号
func (m *Manager) SwitchAccount(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data.Accounts[name]; !ok {
		return fmt.Errorf("账号 %q 不存在", name)
	}
	m.data.Active = name
	m.data.Accounts[name].LastUsed = time.Now()
	return m.save()
}

// AddAccount 添加新账号，自动切换为激活账号，返回 cookies 文件路径
func (m *Manager) AddAccount(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data.Accounts[name]; ok {
		return "", fmt.Errorf("账号 %q 已存在", name)
	}

	cookiesFile := fmt.Sprintf("cookies_%s.json", name)
	m.data.Accounts[name] = &Account{
		Name:        name,
		CookiesFile: cookiesFile,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
	}
	// 添加后自动切换到新账号，便于立即登录
	m.data.Active = name
	if err := m.save(); err != nil {
		return "", err
	}
	return cookiesFile, nil
}

// RemoveAccount 删除账号及其 cookies 文件
func (m *Manager) RemoveAccount(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == m.data.Active {
		return fmt.Errorf("无法删除当前激活账号 %q，请先切换到其他账号", name)
	}
	acc, ok := m.data.Accounts[name]
	if !ok {
		return fmt.Errorf("账号 %q 不存在", name)
	}

	// 删除 cookies 文件（忽略文件不存在的错误）
	_ = os.Remove(acc.CookiesFile)
	delete(m.data.Accounts, name)
	return m.save()
}

// ListAccounts 列出所有账号，返回账号列表和当前激活账号名
func (m *Manager) ListAccounts() ([]*Account, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Account, 0, len(m.data.Accounts))
	for _, acc := range m.data.Accounts {
		list = append(list, acc)
	}
	return list, m.data.Active
}

// GetCookiesPathByName 通过账号名获取 cookies 文件路径
func (m *Manager) GetCookiesPathByName(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	acc, ok := m.data.Accounts[name]
	if !ok {
		return "", fmt.Errorf("账号 %q 不存在", name)
	}
	return acc.CookiesFile, nil
}

// GetActiveProfileDir 获取当前激活账号的 Chrome profile 目录（与 cookies 文件同级）
func (m *Manager) GetActiveProfileDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	acc, ok := m.data.Accounts[m.data.Active]
	if !ok {
		return profileDirFrom(getDefaultCookiePath(), defaultAccountName)
	}
	return profileDirFrom(acc.CookiesFile, acc.Name)
}

// GetProfileDirByName 通过账号名获取 Chrome profile 目录
func (m *Manager) GetProfileDirByName(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	acc, ok := m.data.Accounts[name]
	if !ok {
		return "", fmt.Errorf("账号 %q 不存在", name)
	}
	return profileDirFrom(acc.CookiesFile, acc.Name), nil
}

// profileDirFrom 根据 cookies 文件路径和账号名推导 profile 目录
func profileDirFrom(cookiesPath string, accountName string) string {
	return filepath.Join(filepath.Dir(cookiesPath), "profile_"+accountName)
}

// UpdateLastUsed 更新账号最近使用时间
func (m *Manager) UpdateLastUsed(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if acc, ok := m.data.Accounts[name]; ok {
		acc.LastUsed = time.Now()
		_ = m.save()
	}
}
