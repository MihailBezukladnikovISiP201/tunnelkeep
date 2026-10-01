package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	shell32DLL                  = syscall.NewLazyDLL("shell32.dll")
	procShellNotifyIcon         = shell32DLL.NewProc("Shell_NotifyIconW")
	kernel32DLL                 = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandle         = kernel32DLL.NewProc("GetModuleHandleW")
	procRegisterClassEx        = user32Dll.NewProc("RegisterClassExW")
	procCreateWindowEx          = user32Dll.NewProc("CreateWindowExW")
	procDefWindowProc           = user32Dll.NewProc("DefWindowProcW")
	procDestroyWindow           = user32Dll.NewProc("DestroyWindow")
	procPostQuitMessage         = user32Dll.NewProc("PostQuitMessage")
	procPostMessage             = user32Dll.NewProc("PostMessageW")
	procGetMessage              = user32Dll.NewProc("GetMessageW")
	procTranslateMessage        = user32Dll.NewProc("TranslateMessage")
	procDispatchMessage         = user32Dll.NewProc("DispatchMessageW")
	procCreatePopupMenu         = user32Dll.NewProc("CreatePopupMenu")
	procAppendMenu              = user32Dll.NewProc("AppendMenuW")
	procDestroyMenu             = user32Dll.NewProc("DestroyMenu")
	procTrackPopupMenu          = user32Dll.NewProc("TrackPopupMenu")
	procGetCursorPos            = user32Dll.NewProc("GetCursorPos")
	procSetForegroundWindow     = user32Dll.NewProc("SetForegroundWindow")
	procRegisterWindowMessage   = user32Dll.NewProc("RegisterWindowMessageW")
)

const (
	WM_APP           = 0x8000
	WM_TRAY_CALLBACK = WM_APP + 1

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010

	NIIF_NONE    = 0x00000000
	NIIF_INFO    = 0x00000010
	NIIF_WARNING = 0x00000020
	NIIF_ERROR   = 0x00000030

	MF_STRING    = 0x00000000
	MF_GRAYED    = 0x00000001
	MF_DISABLED  = 0x00000002
	MF_SEPARATOR = 0x00000800

	TPM_BOTTOMALIGN = 0x0020
	TPM_RETURNCMD   = 0x0100
	TPM_NONOTIFY    = 0x0080

	ID_STATUS_HEADER = 1000
	ID_CONNECT_AUTO  = 1001
	ID_DISCONNECT    = 1002
	ID_PAUSE_30M     = 1003
	ID_PAUSE_1H      = 1004
	ID_RESUME        = 1005
	ID_OPEN_CONFIG   = 1006
	ID_OPEN_LOGS     = 1007
	ID_EXIT          = 1008
)

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X int32
	Y int32
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

var (
	globalTray        *TrayApp
	taskbarCreatedMsg uint32
)

type TrayApp struct {
	mu          sync.Mutex
	hwnd        uintptr
	nid         NOTIFYICONDATAW
	iconManager *IconManager
	stateMgr    *StateManager
	adapter     VPNAdapter
	cfg         Config
	stopChan    chan struct{}
	iconAdded   bool
}

func newTrayApp(adapter VPNAdapter, cfg Config, stateMgr *StateManager, iconMgr *IconManager) *TrayApp {
	t := &TrayApp{
		adapter:     adapter,
		cfg:         cfg,
		stateMgr:    stateMgr,
		iconManager: iconMgr,
		stopChan:    make(chan struct{}),
	}
	globalTray = t
	return t
}

func trayWndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	if taskbarCreatedMsg != 0 && msg == taskbarCreatedMsg {
		if globalTray != nil {
			globalTray.addOrUpdateIcon()
		}
		return 0
	}

	switch msg {
	case WM_TRAY_CALLBACK:
		switch lParam {
		case 0x0202, 0x0205: // WM_LBUTTONUP, WM_RBUTTONUP
			if globalTray != nil {
				globalTray.showContextMenu(hwnd)
			}
			return 0
		}
	case 0x0010: // WM_CLOSE
		procDestroyWindow.Call(hwnd)
		return 0
	case 0x0002: // WM_DESTROY
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (t *TrayApp) Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	taskbarStr, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	msgId, _, _ := procRegisterWindowMessage.Call(uintptr(unsafe.Pointer(taskbarStr)))
	taskbarCreatedMsg = uint32(msgId)

	className, _ := syscall.UTF16PtrFromString("VPNGuardianTrayWindowClass")
	hInstance, _, _ := procGetModuleHandle.Call(0)

	wndClass := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   syscall.NewCallback(trayWndProc),
		HInstance:     hInstance,
		LpszClassName: className,
	}

	atom, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wndClass)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassEx failed: %v", err)
	}

	windowName, _ := syscall.UTF16PtrFromString("VPNGuardianHiddenWindow")
	hwnd, _, err := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0,
		0, 0, 0, 0,
		0, 0,
		hInstance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx failed: %v", err)
	}
	t.hwnd = hwnd

	t.nid = NOTIFYICONDATAW{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAY_CALLBACK,
		HIcon:            t.iconManager.PausedHIcon,
	}
	copy(t.nid.SzTip[:], toUTF16Fixed(fmt.Sprintf("VPN Guardian [%s]", t.adapter.Name()), 128))

	// Initial icon registration attempt
	t.addOrUpdateIcon()

	// Asynchronous retry if desktop taskbar wasn't ready immediately
	go func() {
		for i := 0; i < 15; i++ {
			t.mu.Lock()
			added := t.iconAdded
			t.mu.Unlock()
			if added {
				break
			}
			time.Sleep(2 * time.Second)
			t.addOrUpdateIcon()
		}
	}()

	go t.watchState()

	// Win32 Message Loop
	var msg MSG
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	// Cleanup on exit
	t.mu.Lock()
	if t.iconAdded {
		procShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&t.nid)))
		t.iconAdded = false
	}
	t.mu.Unlock()

	close(t.stopChan)
	return nil
}

func (t *TrayApp) addOrUpdateIcon() {
	t.mu.Lock()
	defer t.mu.Unlock()

	m := T()
	state, pauseUntil, retryCount := t.stateMgr.GetState()
	var tip string
	var hIcon uintptr

	switch state {
	case StateConnected:
		tip = fmt.Sprintf(m.TipConnected, t.adapter.Name())
		hIcon = t.iconManager.ConnectedHIcon
	case StateReconnecting:
		tip = fmt.Sprintf(m.TipReconnecting, t.adapter.Name(), retryCount)
		hIcon = t.iconManager.ReconnectingHIcon
	case StatePaused:
		if !pauseUntil.IsZero() {
			remain := time.Until(pauseUntil).Round(time.Minute)
			tip = fmt.Sprintf(m.TipPausedUntil, pauseUntil.Format("15:04"), remain)
		} else {
			tip = m.TipPaused
		}
		hIcon = t.iconManager.PausedHIcon
	}

	t.nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	t.nid.HIcon = hIcon
	copy(t.nid.SzTip[:], toUTF16Fixed(tip, 128))

	if !t.iconAdded {
		r, _, _ := procShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&t.nid)))
		if r != 0 {
			t.iconAdded = true
			appLogger.Logf("Иконка в трее успешно создана.")
		}
	} else {
		procShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&t.nid)))
	}
}

func (t *TrayApp) watchState() {
	for range t.stateMgr.stateChanged {
		t.addOrUpdateIcon()
	}
}

func (t *TrayApp) ShowBalloon(title, message string, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.iconAdded || !t.cfg.EnableNotifications {
		return
	}

	nid := t.nid
	nid.UFlags |= NIF_INFO
	flags := uint32(NIIF_INFO)
	if isError {
		flags = NIIF_WARNING
	}
	nid.DwInfoFlags = flags
	copy(nid.SzInfoTitle[:], toUTF16Fixed(title, 64))
	copy(nid.SzInfo[:], toUTF16Fixed(message, 256))

	procShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

func (t *TrayApp) showContextMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	m := T()
	state, pauseUntil, _ := t.stateMgr.GetState()

	// Header
	statusStr := state.String()
	if state == StatePaused && !pauseUntil.IsZero() {
		statusStr = fmt.Sprintf(m.StatusPausedUntil, pauseUntil.Format("15:04"))
	}
	appendMenuItem(hMenu, MF_STRING|MF_DISABLED, ID_STATUS_HEADER, fmt.Sprintf("VPN Guardian [%s] • %s: %s", t.adapter.Name(), m.StatusHeader, statusStr))
	appendMenuItem(hMenu, MF_SEPARATOR, 0, "")

	// Quick controls
	if state == StatePaused {
		appendMenuItem(hMenu, MF_STRING, ID_CONNECT_AUTO, m.MenuConnectAuto)
		appendMenuItem(hMenu, MF_STRING, ID_RESUME, m.MenuResume)
	} else {
		appendMenuItem(hMenu, MF_STRING, ID_DISCONNECT, m.MenuDisconnect)
		appendMenuItem(hMenu, MF_STRING, ID_PAUSE_30M, m.MenuPause30m)
		appendMenuItem(hMenu, MF_STRING, ID_PAUSE_1H, m.MenuPause1h)
	}

	appendMenuItem(hMenu, MF_SEPARATOR, 0, "")
	appendMenuItem(hMenu, MF_STRING, ID_OPEN_CONFIG, m.MenuOpenConfig)
	appendMenuItem(hMenu, MF_STRING, ID_OPEN_LOGS, m.MenuOpenLogs)
	appendMenuItem(hMenu, MF_SEPARATOR, 0, "")
	appendMenuItem(hMenu, MF_STRING, ID_EXIT, m.MenuExit)

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(hwnd)

	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		TPM_BOTTOMALIGN|TPM_RETURNCMD|TPM_NONOTIFY,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		hwnd,
		0,
	)
	procPostMessage.Call(hwnd, 0, 0, 0)

	t.handleMenuCommand(uint32(cmd))
}

func (t *TrayApp) handleMenuCommand(cmd uint32) {
	m := T()
	switch cmd {
	case ID_CONNECT_AUTO:
		appLogger.Logf("Пользователь выбрал: Подключить и включить авторежим")
		t.stateMgr.SetConnected()
		go func() {
			if err := t.adapter.Connect(); err != nil {
				appLogger.Logf("Ошибка ручного подключения: %v", err)
				t.ShowBalloon(m.BalloonErrorTitle, err.Error(), true)
			} else {
				appLogger.Logf("VPN успешно подключен вручную")
				t.ShowBalloon("VPN Guardian", fmt.Sprintf(m.BalloonReconnected, t.adapter.Name()), false)
			}
		}()

	case ID_DISCONNECT:
		appLogger.Logf("Пользователь выбрал: Отключить VPN (ручная пауза)")
		t.stateMgr.SetPaused(time.Time{})
		go func() {
			if err := t.adapter.Disconnect(); err != nil {
				appLogger.Logf("Ошибка отключения: %v", err)
			} else {
				appLogger.Logf("VPN отключен пользователем")
			}
			t.ShowBalloon("VPN Guardian", m.BalloonDisconnect, false)
		}()

	case ID_PAUSE_30M:
		until := time.Now().Add(30 * time.Minute)
		appLogger.Logf("Автореконнект приостановлен на 30 минут (до %s)", until.Format("15:04:05"))
		t.stateMgr.SetPaused(until)
		t.ShowBalloon("VPN Guardian", fmt.Sprintf(m.BalloonPaused30m, until.Format("15:04")), false)

	case ID_PAUSE_1H:
		until := time.Now().Add(1 * time.Hour)
		appLogger.Logf("Автореконнект приостановлен на 1 час (до %s)", until.Format("15:04:05"))
		t.stateMgr.SetPaused(until)
		t.ShowBalloon("VPN Guardian", fmt.Sprintf(m.BalloonPaused1h, until.Format("15:04")), false)

	case ID_RESUME:
		appLogger.Logf("Пользователь возобновил автоконтроль")
		t.stateMgr.SetConnected()
		t.ShowBalloon("VPN Guardian", m.BalloonResumed, false)

	case ID_OPEN_CONFIG:
		go exec.Command("notepad.exe", getConfigPath()).Start()

	case ID_OPEN_LOGS:
		go exec.Command("notepad.exe", appLogger.GetPath()).Start()

	case ID_EXIT:
		appLogger.Logf("Выход из приложения по запросу пользователя")
		procDestroyWindow.Call(t.hwnd)
	}
}

func appendMenuItem(hMenu uintptr, flags uint32, id uintptr, text string) {
	ptr, _ := syscall.UTF16PtrFromString(text)
	procAppendMenu.Call(hMenu, uintptr(flags), id, uintptr(unsafe.Pointer(ptr)))
}

func toUTF16Fixed(s string, maxLen int) []uint16 {
	utf, _ := syscall.UTF16FromString(s)
	res := make([]uint16, maxLen)
	copy(res, utf)
	return res
}
