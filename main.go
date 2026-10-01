package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	logger, err := initLogger()
	if err != nil {
		fmt.Printf("Не удалось инициализировать логгер: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Auto-detect default VPN
	detectedVPN := DetectPrimaryVPN()

	// Load or create config.json in %APPDATA%\VPNGuardian\config.json
	cfg, err := loadOrCreateConfig(detectedVPN)
	if err != nil {
		logger.Logf("Предупреждение при загрузке config.json: %v", err)
	}

	// CLI flags (can override config)
	vpnNameFlag := flag.String("vpn", "", "Имя VPN-подключения (переопределяет config.json)")
	vpnTypeFlag := flag.String("type", "", "Тип VPN: rasdial, wireguard, openvpn")
	intervalFlag := flag.Duration("interval", 0, "Интервал проверки статуса соединения")
	maxRetriesFlag := flag.Int("max-retries", 0, "Максимальное число попыток переподключения")
	retryDelayFlag := flag.Duration("retry-delay", 0, "Пауза между попытками переподключения")
	autoConnectFlag := flag.Bool("auto-connect", false, "Принудительно подключить VPN при старте программы")
	flag.Parse()

	if *vpnNameFlag != "" {
		cfg.VPNName = *vpnNameFlag
	}
	if *vpnTypeFlag != "" {
		cfg.VPNType = *vpnTypeFlag
	}
	if *intervalFlag > 0 {
		cfg.CheckIntervalSeconds = int(intervalFlag.Seconds())
	}
	if *maxRetriesFlag > 0 {
		cfg.MaxRetries = *maxRetriesFlag
	}
	if *retryDelayFlag > 0 {
		cfg.RetryDelaySeconds = int(retryDelayFlag.Seconds())
	}
	if *autoConnectFlag {
		cfg.AutoConnectAtStart = true
	}

	initI18n(cfg.Language)

	adapter := NewVPNAdapter(cfg)

	logger.Logf("=== Запуск VPN Guardian v1.0.0 ===")
	logger.Logf("Язык интерфейса: [%s], Адаптер: [%s], Имя/Цель: [%s]", currentLang, cfg.VPNType, adapter.Name())
	logger.Logf("Конфигурация: %s", getConfigPath())

	iconMgr, err := newIconManager()
	if err != nil {
		logger.Logf("Ошибка инициализации иконок: %v", err)
		os.Exit(1)
	}
	defer iconMgr.Close()

	stateMgr := newStateManager()

	// Initial status probe
	initialConnected, err := adapter.IsConnected()
	if err != nil {
		logger.Logf("Предупреждение при начальной проверке VPN: %v", err)
	}

	if initialConnected {
		logger.Logf("VPN [%s] уже подключен. Активирован режим автоконтроля.", adapter.Name())
		stateMgr.SetConnected()
	} else if cfg.AutoConnectAtStart {
		logger.Logf("VPN [%s] отключен. Запуск автоподключения по настройке auto_connect_at_start...", adapter.Name())
		stateMgr.SetConnected()
		go func() {
			if err := adapter.Connect(); err != nil {
				logger.Logf("Ошибка стартового автоподключения: %v", err)
			}
		}()
	} else {
		logger.Logf("VPN [%s] отключен. Приложение запущено в режиме ожидания (ручной режим).", adapter.Name())
		stateMgr.SetPaused(time.Time{})
	}

	tray := newTrayApp(adapter, cfg, stateMgr, iconMgr)

	// Background watchdog loop
	go runWatchdog(adapter, cfg, stateMgr, tray)

	// Run tray loop on main thread
	if err := tray.Run(); err != nil {
		logger.Logf("Фатальная ошибка Tray: %v", err)
	}

	logger.Logf("=== VPN Guardian остановлен ===")
}

func runWatchdog(adapter VPNAdapter, cfg Config, stateMgr *StateManager, tray *TrayApp) {
	interval := time.Duration(cfg.CheckIntervalSeconds) * time.Second
	retryDelay := time.Duration(cfg.RetryDelaySeconds) * time.Second
	maxRetries := cfg.MaxRetries

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if stateMgr.IsPaused() {
			continue
		}

		connected, err := adapter.IsConnected()
		if err != nil {
			appLogger.Logf("Ошибка проверки статуса VPN: %v", err)
			continue
		}

		if connected {
			stateMgr.SetConnected()
			continue
		}

		m := T()
		// VPN connection dropped unexpectedly
		appLogger.Logf("ВНИМАНИЕ: Обнаружен разрыв соединения с VPN [%s]!", adapter.Name())
		tray.ShowBalloon("VPN Guardian", fmt.Sprintf(m.BalloonConnDropped, adapter.Name()), true)

		reconnected := false
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if stateMgr.IsPaused() {
				appLogger.Logf("Пользователь перевел режим в паузу во время восстановления. Отмена попыток.")
				break
			}

			stateMgr.SetReconnecting(attempt, "Попытка подключения...")
			appLogger.Logf("Попытка переподключения %d из %d...", attempt, maxRetries)

			if err := adapter.Connect(); err != nil {
				appLogger.Logf("Попытка %d не удалась: %v", attempt, err)
			} else {
				time.Sleep(1 * time.Second)
				if isUp, _ := adapter.IsConnected(); isUp {
					appLogger.Logf("VPN [%s] успешно переподключен (попытка %d)!", adapter.Name(), attempt)
					stateMgr.SetConnected()
					tray.ShowBalloon("VPN Guardian", fmt.Sprintf(m.BalloonReconnected, adapter.Name()), false)
					reconnected = true
					break
				}
			}

			time.Sleep(retryDelay)
		}

		if !reconnected && !stateMgr.IsPaused() {
			appLogger.Logf("Не удалось восстановить связь после %d попыток. Переход в паузу.", maxRetries)
			tray.ShowBalloon("VPN Guardian", fmt.Sprintf(m.BalloonRetriesFail, adapter.Name(), maxRetries), true)
			stateMgr.SetPaused(time.Time{})
		}
	}
}
