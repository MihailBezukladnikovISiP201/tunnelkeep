package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// VPNAdapter defines the common interface for VPN controllers.
type VPNAdapter interface {
	IsConnected() (bool, error)
	Connect() error
	Disconnect() error
	Name() string
}

func NewVPNAdapter(cfg Config) VPNAdapter {
	switch strings.ToLower(cfg.VPNType) {
	case "wireguard":
		return &WireGuardAdapter{
			tunnelName: cfg.VPNName,
			configPath: cfg.WireGuardConfigPath,
		}
	case "openvpn":
		return &OpenVPNAdapter{
			profileName: cfg.OpenVPNProfileName,
		}
	default:
		return &RasDialAdapter{
			connectionName: cfg.VPNName,
		}
	}
}

// -------------------------------------------------------------
// RasDialAdapter (Windows Native VPN: SSTP, L2TP, IKEv2, PPTP)
// -------------------------------------------------------------

type RasDialAdapter struct {
	connectionName string
}

func (a *RasDialAdapter) Name() string {
	return a.connectionName
}

func (a *RasDialAdapter) IsConnected() (bool, error) {
	cmd := exec.Command("rasdial.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()
	if err != nil && strings.TrimSpace(output) == "" {
		return false, fmt.Errorf("rasdial failed: %w", err)
	}

	lines := strings.Split(output, "\n")
	target := strings.ToLower(strings.TrimSpace(a.connectionName))

	for _, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if trimmed == target {
			return true, nil
		}
	}

	return false, nil
}

func (a *RasDialAdapter) Connect() error {
	cmd := exec.Command("rasdial.exe", a.connectionName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rasdial connect error: %s (%w)", strings.TrimSpace(out.String()), err)
	}
	return nil
}

func (a *RasDialAdapter) Disconnect() error {
	cmd := exec.Command("rasdial.exe", a.connectionName, "/DISCONNECT")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rasdial disconnect error: %s (%w)", strings.TrimSpace(out.String()), err)
	}
	return nil
}

// -------------------------------------------------------------
// WireGuardAdapter (Official WireGuard Windows Tunnel Service)
// -------------------------------------------------------------

type WireGuardAdapter struct {
	tunnelName string
	configPath string
}

func (a *WireGuardAdapter) Name() string {
	if a.tunnelName != "" {
		return a.tunnelName
	}
	return filepath.Base(a.configPath)
}

func (a *WireGuardAdapter) IsConnected() (bool, error) {
	tunnel := a.tunnelName
	if tunnel == "" && a.configPath != "" {
		base := filepath.Base(a.configPath)
		tunnel = strings.TrimSuffix(base, filepath.Ext(base))
	}
	// WireGuard creates a service named "WireGuardTunnel$<TunnelName>"
	serviceName := "WireGuardTunnel$" + tunnel
	cmd := exec.Command("sc.exe", "query", serviceName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	_ = cmd.Run()
	output := out.String()
	if strings.Contains(output, "RUNNING") {
		return true, nil
	}
	return false, nil
}

func (a *WireGuardAdapter) Connect() error {
	wgExe := "wireguard.exe"
	cmd := exec.Command(wgExe, "/installtunnelservice", a.configPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wireguard connect error: %s (%w)", strings.TrimSpace(out.String()), err)
	}
	return nil
}

func (a *WireGuardAdapter) Disconnect() error {
	tunnel := a.tunnelName
	if tunnel == "" && a.configPath != "" {
		base := filepath.Base(a.configPath)
		tunnel = strings.TrimSuffix(base, filepath.Ext(base))
	}
	wgExe := "wireguard.exe"
	cmd := exec.Command(wgExe, "/uninstalltunnelservice", tunnel)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wireguard disconnect error: %s (%w)", strings.TrimSpace(out.String()), err)
	}
	return nil
}

// -------------------------------------------------------------
// OpenVPNAdapter (OpenVPN GUI CLI)
// -------------------------------------------------------------

type OpenVPNAdapter struct {
	profileName string
}

func (a *OpenVPNAdapter) Name() string {
	return a.profileName
}

func (a *OpenVPNAdapter) IsConnected() (bool, error) {
	// Query OpenVPN GUI status or network adapter
	cmd := exec.Command("netsh.exe", "interface", "show", "interface")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	_ = cmd.Run()
	output := out.String()
	// Check for connected OpenVPN TAP/Wintun adapter
	if strings.Contains(output, "Connected") && (strings.Contains(output, "TAP") || strings.Contains(output, "OpenVPN") || strings.Contains(output, a.profileName)) {
		return true, nil
	}
	return false, nil
}

func (a *OpenVPNAdapter) Connect() error {
	ovpnExe := "openvpn-gui.exe"
	cmd := exec.Command(ovpnExe, "--connect", a.profileName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("openvpn connect error: %w", err)
	}
	return nil
}

func (a *OpenVPNAdapter) Disconnect() error {
	ovpnExe := "openvpn-gui.exe"
	cmd := exec.Command(ovpnExe, "--command", "disconnect", a.profileName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("openvpn disconnect error: %w", err)
	}
	return nil
}

// -------------------------------------------------------------
// Auto-Detection
// -------------------------------------------------------------

// DetectPrimaryVPN discovers existing Windows VPN entries by reading the native PBK phonebook directly.
func DetectPrimaryVPN() string {
	// 1. User phonebook
	appData := os.Getenv("APPDATA")
	if appData != "" {
		userPbk := filepath.Join(appData, "Microsoft", "Network", "Connections", "Pbk", "rasphone.pbk")
		if name := extractFirstPbkSection(userPbk); name != "" {
			return name
		}
	}

	// 2. System / All Users phonebook
	progData := os.Getenv("ProgramData")
	if progData != "" {
		systemPbk := filepath.Join(progData, "Microsoft", "Network", "Connections", "Pbk", "rasphone.pbk")
		if name := extractFirstPbkSection(systemPbk); name != "" {
			return name
		}
	}

	return "SG"
}

func extractFirstPbkSection(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.Trim(line, "[]")
			if name != "" {
				return name
			}
		}
	}
	return ""
}
