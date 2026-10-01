package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"syscall"
	"unsafe"
)

var (
	user32Dll                  = syscall.NewLazyDLL("user32.dll")
	procCreateIconFromResourceEx = user32Dll.NewProc("CreateIconFromResourceEx")
	procDestroyIcon            = user32Dll.NewProc("DestroyIcon")
)

type IconManager struct {
	ConnectedHIcon    uintptr
	ReconnectingHIcon uintptr
	PausedHIcon       uintptr
}

func newIconManager() (*IconManager, error) {
	// Colors
	// Green: rgb(34, 197, 94)
	greenData := generateIconPNG(32, 32, color.RGBA{R: 34, G: 197, B: 94, A: 255})
	// Orange/Yellow: rgb(245, 158, 11)
	yellowData := generateIconPNG(32, 32, color.RGBA{R: 245, G: 158, B: 11, A: 255})
	// Gray/Silver: rgb(156, 163, 175)
	grayData := generateIconPNG(32, 32, color.RGBA{R: 156, G: 163, B: 175, A: 255})

	hGreen := createHIcon(greenData, 32, 32)
	hYellow := createHIcon(yellowData, 32, 32)
	hGray := createHIcon(grayData, 32, 32)

	return &IconManager{
		ConnectedHIcon:    hGreen,
		ReconnectingHIcon: hYellow,
		PausedHIcon:       hGray,
	}, nil
}

func (im *IconManager) Close() {
	if im.ConnectedHIcon != 0 {
		procDestroyIcon.Call(im.ConnectedHIcon)
	}
	if im.ReconnectingHIcon != 0 {
		procDestroyIcon.Call(im.ReconnectingHIcon)
	}
	if im.PausedHIcon != 0 {
		procDestroyIcon.Call(im.PausedHIcon)
	}
}

func createHIcon(pngBytes []byte, width, height int) uintptr {
	if len(pngBytes) == 0 {
		return 0
	}
	hIcon, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&pngBytes[0])),
		uintptr(len(pngBytes)),
		1,          // fIcon = TRUE
		0x00030000, // dwVersion (Windows 3.0+)
		uintptr(width),
		uintptr(height),
		0, // LR_DEFAULTCOLOR
	)
	return hIcon
}

func generateIconPNG(w, h int, baseColor color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2.0, float64(h)/2.0
	outerRadius := float64(w)/2.0 - 2.0
	innerRadius := outerRadius - 2.0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist <= innerRadius {
				// Inner filled circle
				img.SetRGBA(x, y, baseColor)
			} else if dist <= outerRadius {
				// Dark outline/border for contrast against white or dark taskbars
				img.SetRGBA(x, y, color.RGBA{R: 30, G: 41, B: 59, A: 240})
			} else if dist <= outerRadius+1.2 {
				// Subtle anti-aliasing edge
				alpha := uint8(255 * (1.0 - (dist-outerRadius)/1.2))
				img.SetRGBA(x, y, color.RGBA{R: 30, G: 41, B: 59, A: alpha})
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}
