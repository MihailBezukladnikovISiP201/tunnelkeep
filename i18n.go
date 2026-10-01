package main

import (
	"strings"
)

var (
	kernel32DLLProcGetUserDefaultUILanguage = kernel32DLL.NewProc("GetUserDefaultUILanguage")
)

type Language string

const (
	LangRU Language = "ru"
	LangEN Language = "en"
)

type Messages struct {
	StatusHeader       string
	StatusConnected    string
	StatusReconnecting string
	StatusPaused       string
	StatusPausedUntil  string
	TipConnected       string
	TipReconnecting    string
	TipPaused          string
	TipPausedUntil     string
	MenuConnectAuto    string
	MenuResume         string
	MenuDisconnect     string
	MenuPause30m       string
	MenuPause1h        string
	MenuOpenConfig     string
	MenuOpenLogs       string
	MenuExit           string
	BalloonDisconnect  string
	BalloonReconnected string
	BalloonPaused30m   string
	BalloonPaused1h    string
	BalloonResumed     string
	BalloonErrorTitle  string
	BalloonConnDropped string
	BalloonRetriesFail string
}

var (
	ruMessages = Messages{
		StatusHeader:       "Статус",
		StatusConnected:    "Подключен (Автоконтроль)",
		StatusReconnecting: "Переподключение...",
		StatusPaused:       "Пауза (Ручной режим)",
		StatusPausedUntil:  "Пауза до %s",
		TipConnected:       "TunnelKeep: Подключен (%s)\nАвтоконтроль активен",
		TipReconnecting:    "TunnelKeep: Переподключение к %s...\nПопытка %d",
		TipPaused:          "TunnelKeep: Ручной режим (VPN выключен)\nАвтореконнект спит",
		TipPausedUntil:     "TunnelKeep: Пауза до %s (осталось %v)",
		MenuConnectAuto:    "⚡ Подключить и включить авторежим",
		MenuResume:         "▶ Возобновить автоконтроль",
		MenuDisconnect:     "⏹ Отключить VPN (ручная пауза)",
		MenuPause30m:       "⏸ Пауза автореконнекта на 30 мин",
		MenuPause1h:        "⏸ Пауза автореконнекта на 1 час",
		MenuOpenConfig:     "⚙ Настройки (config.json)",
		MenuOpenLogs:       "📄 Открыть лог",
		MenuExit:           "❌ Закрыть приложение",
		BalloonDisconnect:  "VPN отключен. Автореконнект приостановлен.",
		BalloonReconnected: "Связь с %s успешно восстановлена!",
		BalloonPaused30m:   "Автореконнект отключен на 30 минут (до %s)",
		BalloonPaused1h:    "Автореконнект отключен на 1 час (до %s)",
		BalloonResumed:     "Автореконнект возобновлен",
		BalloonErrorTitle:  "Ошибка подключения",
		BalloonConnDropped: "Обрыв связи с %s. Запуск восстановления...",
		BalloonRetriesFail: "Не удалось восстановить связь с %s после %d попыток. Автоконтроль приостановлен.",
	}

	enMessages = Messages{
		StatusHeader:       "Status",
		StatusConnected:    "Connected (Active Watchdog)",
		StatusReconnecting: "Reconnecting...",
		StatusPaused:       "Paused (Manual Mode)",
		StatusPausedUntil:  "Paused until %s",
		TipConnected:       "TunnelKeep: Connected (%s)\nWatchdog Active",
		TipReconnecting:    "TunnelKeep: Reconnecting to %s...\nAttempt %d",
		TipPaused:          "TunnelKeep: Manual Mode (VPN Disconnected)\nWatchdog Inactive",
		TipPausedUntil:     "TunnelKeep: Paused until %s (%v remaining)",
		MenuConnectAuto:    "⚡ Connect & Enable Watchdog",
		MenuResume:         "▶ Resume Watchdog",
		MenuDisconnect:     "⏹ Disconnect VPN (Manual Pause)",
		MenuPause30m:       "⏸ Pause Watchdog for 30 min",
		MenuPause1h:        "⏸ Pause Watchdog for 1 hour",
		MenuOpenConfig:     "⚙ Settings (config.json)",
		MenuOpenLogs:       "📄 Open Logs",
		MenuExit:           "❌ Exit Application",
		BalloonDisconnect:  "VPN disconnected. Watchdog paused.",
		BalloonReconnected: "Connection to %s restored!",
		BalloonPaused30m:   "Watchdog paused for 30 minutes (until %s)",
		BalloonPaused1h:    "Watchdog paused for 1 hour (until %s)",
		BalloonResumed:     "Watchdog resumed",
		BalloonErrorTitle:  "Connection Error",
		BalloonConnDropped: "Connection dropped with %s. Recovering...",
		BalloonRetriesFail: "Failed to reconnect to %s after %d attempts. Watchdog paused.",
	}

	currentMessages Messages
	currentLang     Language
)

func initI18n(langPreference string) {
	lang := resolveLanguage(langPreference)
	currentLang = lang
	if lang == LangRU {
		currentMessages = ruMessages
	} else {
		currentMessages = enMessages
	}
}

func resolveLanguage(pref string) Language {
	pref = strings.ToLower(strings.TrimSpace(pref))
	if pref == "ru" {
		return LangRU
	}
	if pref == "en" {
		return LangEN
	}

	// Auto-detect Windows UI language
	r, _, _ := kernel32DLLProcGetUserDefaultUILanguage.Call()
	langID := uint16(r)

	// Primary language ID mask (lower 10 bits)
	primaryLang := langID & 0x03FF
	switch primaryLang {
	case 0x19: // Russian (LANG_RUSSIAN = 0x19)
		return LangRU
	case 0x22: // Ukrainian
		return LangRU
	case 0x23: // Belarusian
		return LangRU
	default:
		return LangEN
	}
}

func T() *Messages {
	return &currentMessages
}
