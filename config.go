package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	VPNType              string `json:"vpn_type"`               // "rasdial", "wireguard", "openvpn"
	VPNName              string `json:"vpn_name"`               // Имя подключения в Windows или имя туннеля
	CheckIntervalSeconds int    `json:"check_interval_seconds"` // Частота проверки статуса (сек)
	MaxRetries           int    `json:"max_retries"`            // Максимум попыток переподключения
	RetryDelaySeconds    int    `json:"retry_delay_seconds"`    // Пауза между попытками (сек)
	AutoConnectAtStart   bool   `json:"auto_connect_at_start"`  // Подключать ли VPN автоматически при старте
	EnableNotifications  bool   `json:"enable_notifications"`   // Показывать ли всплывающие уведомления
	WireGuardConfigPath  string `json:"wireguard_config_path"`  // Путь к .conf файлу для WireGuard
	OpenVPNProfileName   string `json:"openvpn_profile_name"`   // Имя профиля для OpenVPN GUI
	DonateURL            string `json:"donate_url"`             // Ссылка для поддержки автора
	ReferralURL          string `json:"referral_url"`           // Партнерская ссылка на хостинг/VPS
}

var (
	configMu sync.RWMutex
	cfgPath  string
)

func getDefaultConfig(detectedVPN string) Config {
	targetVPN := "SG"
	if detectedVPN != "" {
		targetVPN = detectedVPN
	}

	return Config{
		VPNType:              "rasdial",
		VPNName:              targetVPN,
		CheckIntervalSeconds: 5,
		MaxRetries:           10,
		RetryDelaySeconds:    4,
		AutoConnectAtStart:   false,
		EnableNotifications:  true,
		WireGuardConfigPath:  "",
		OpenVPNProfileName:   "",
		DonateURL:            "https://boosty.to",
		ReferralURL:          "https://timeweb.cloud",
	}
}

func loadOrCreateConfig(detectedVPN string) (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	dir := filepath.Join(appData, "VPNGuardian")
	_ = os.MkdirAll(dir, 0755)

	cfgPath = filepath.Join(dir, "config.json")

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg := getDefaultConfig(detectedVPN)
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err == nil {
			_ = os.WriteFile(cfgPath, data, 0644)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return getDefaultConfig(detectedVPN), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return getDefaultConfig(detectedVPN), err
	}

	// Fallback defaults for missing fields
	if cfg.CheckIntervalSeconds <= 0 {
		cfg.CheckIntervalSeconds = 5
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 10
	}
	if cfg.RetryDelaySeconds <= 0 {
		cfg.RetryDelaySeconds = 4
	}
	if cfg.VPNType == "" {
		cfg.VPNType = "rasdial"
	}
	if cfg.DonateURL == "" {
		cfg.DonateURL = "https://boosty.to"
	}
	if cfg.ReferralURL == "" {
		cfg.ReferralURL = "https://timeweb.cloud"
	}

	return cfg, nil
}

func getConfigPath() string {
	return cfgPath
}
