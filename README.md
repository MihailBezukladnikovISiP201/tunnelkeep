# VPN Guardian 🛡️

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![Platform](https://img.shields.io/badge/Platform-Windows%2010%20%2F%2011-0078D6?logo=windows)](https://microsoft.com)
[![Permissions](https://img.shields.io/badge/Permissions-Non--Admin%20%2F%20User%20Space-success)](#)
[![Architecture](https://img.shields.io/badge/Arch-amd64%20%7C%20386%20%7C%20arm64-brightgreen)](#)

> **VPN Guardian** — ультралегковесный сторожевой сервис (Watchdog) для Windows в системном трее, работающий **полностью без прав администратора**. Автоматически восстанавливает разорванные корпоративные VPN-соединения в фоне, избавляя от рутинных ручных переподключений и потери фокуса.

---

## ⚡ Какую реальную проблему решает проект?

### 1. Отсутствие прав администратора на корпоративных ПК (Главная боль)
На рабочих ноутбуках и офисных машинах у пользователей почти никогда **нет прав локального администратора**. 
Из-за этого «ванильные» решения не работают:
- ❌ Нельзя установить сторонние тяжелые VPN-менеджеры.
- ❌ Нельзя установить системные драйверы, туннельные адаптеры или службы Windows.
- ❌ Нельзя настроить задачи в Планировщике Windows с повышенными привилегиями.

**VPN Guardian работает на 100% в пространстве пользователя (User Space):**
- Не требует прав администратора (UAC).
- Читает профили напрямую из пользовательской телефонной книги Windows (`rasphone.pbk`).
- Управляет туннелем через встроенную штатную утилиту `rasdial`.
- Запускается и работает автономно в трее пользователя.

### 2. Постоянные обрывы и прерывание рабочего процесса
Корпоративные VPN часто тихо разрываются (мигание Wi-Fi, смена IP, таймаут провайдера). В результате:
- Падают SSH-сессии, прерываются `git fetch / git push`, отваливаются внутренние корпоративные порталы и базы данных.
- Приходится постоянно отвлекаться от задач, открывать сетевые настройки и вручную нажимать «Подключить».

**VPN Guardian берет удержание связи на себя:** автоматически фиксирует обрыв и прозрачно для вас поднимает туннель обратно за секунды.

### 3. Умная пауза при осознанном выключении (Встроенная фича)
Приложение отличает аварийный сбой сети от намеренного выключения пользователем:
- 🟢 **Автоконтроль:** следит за туннелем в фоне и спасает при обрывах.
- ⚪ **Ручной режим (Пауза):** при клике *«⏹ Отключить VPN (ручная пауза)»* утилита корректно закрывает туннель и **замораживает мониторинг**, не пытаясь включить его обратно.
- ⏸ **Временная пауза:** быстрая пауза автореконнекта на 30 минут или 1 час.

---

## 📊 Архитектура и оптимизация

```mermaid
stateDiagram-v2
    [*] --> MONITORING : Запуск (VPN активен)
    [*] --> PAUSED : Запуск (VPN выключен)

    state MONITORING {
        [*] --> CheckStatus
        CheckStatus --> Connected : VPN активен (сон N сек)
        CheckStatus --> Reconnecting : Обрыв сети!
        Reconnecting --> Connected : Связь восстановлена
        Reconnecting --> PAUSED : Превышен лимит попыток
    }

    MONITORING --> PAUSED : Клик "Отключить VPN" / "Пауза"
    PAUSED --> MONITORING : Клик "Подключить и включить авторежим"
```

- **Потребление RAM:** всего **8–12 МБ** (в отличие от 100+ МБ у Electron/Python).
- **Потребление CPU:** **0.0%** в режиме ожидания (блокирующий цикл сообщений Win32 API `GetMessageW`).
- **Zero CGO & Zero Runtime:** собран на чистом Go с прямыми системными вызовами (`shell32.dll`, `user32.dll`), без внешних зависимостей.
- **Устойчивость к сбоям Explorer:** слушает системное событие Windows `TaskbarCreated` и мгновенно пересоздает иконку в трее при перезапуске рабочего стола.

---

## 🔌 Поддерживаемые клиенты и протоколы

| Адаптер | Поддерживаемые протоколы | Права администратора | Метод интеграции |
|---|---|---|---|
| **Windows Native (`rasdial`)** | SSTP, L2TP/IPSec, IKEv2, PPTP | ❌ **Не требуются (User Space)** | Windows RAS API / телефонная книга |
| **WireGuard** | WireGuard Windows Client | Требуются только при установке службы | `/installtunnelservice` |
| **OpenVPN** | OpenVPN GUI | Не требуются для запуска | CLI интерфейс (`openvpn-gui.exe`) |

---

## 🚀 Быстрый запуск

### 1. Скачать готовый бинарник
Скачайте свежий `vpn-guardian.exe` со страницы [**Releases**](../../releases).
Запустите файл двойным кликом — он сразу свернется в трей возле часов и подхватит ваши сохраненные корпоративные подключения.

### 2. Сборка из исходников
```powershell
git clone https://github.com/MihailBezukladnikovISiP201/vpn-guardian.git
cd vpn-guardian
go build -ldflags="-H=windowsgui" -o vpn-guardian.exe
.\vpn-guardian.exe
```

---

## ⚙️ Конфигурация (`config.json`)

Настройки хранятся в `%APPDATA%\VPNGuardian\config.json` и открываются в 1 клик прямо из меню трея:

```json
{
  "vpn_type": "rasdial",
  "vpn_name": "SG",
  "check_interval_seconds": 5,
  "max_retries": 10,
  "retry_delay_seconds": 4,
  "auto_connect_at_start": false,
  "enable_notifications": true,
  "wireguard_config_path": "",
  "openvpn_profile_name": "",
  "donate_url": "https://boosty.to/your_profile",
  "referral_url": "https://timeweb.cloud/?ref=your_ref_code"
}
```

---

## ⏰ Автозагрузка Windows (без админских прав)

Чтобы утилита стартовала при входе пользователя в Windows:
```powershell
powershell -ExecutionPolicy Bypass -File .\install-startup.ps1
```
Для удаления из автозагрузки:
```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-startup.ps1
```

---

## ☕ Поддержка автора и развитие проекта

Если утилита сэкономила вам рабочее время и нервы:
- **Boosty / CloudTips:** [boosty.to/your_profile](https://boosty.to)

### 🌐 Надежные VPS для поднятия личного VPN
Для стабильной удаленной работы рекомендуем проверенных провайдеров:
- 👉 [**Timeweb Cloud**](https://timeweb.cloud) — VPS с предустановленным VPN в 1 клик.
- 👉 [**Aeza**](https://aeza.net) — высокоскоростные каналы без лимита трафика.

---

## 📄 Лицензия

Распространяется под лицензией [MIT](LICENSE).
