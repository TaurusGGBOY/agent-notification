package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	toastCardWidth  = 364
	toastCardHeight = 180
	toastCardScale  = 4
)

type ToastCard struct {
	Event   string
	Title   string
	Agent   string
	Project string
	Message string
}

func toastCardPath() (string, error) {
	return toastAssetPath("toast-card.png")
}

func toastAppLogoPath() (string, error) {
	return toastAssetPath("app-logo.png")
}

func toastAssetPath(fileName string) (string, error) {
	var base string
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, "Library", "Caches", "AgentNotify")
		} else {
			base = filepath.Join(os.TempDir(), "AgentNotify")
		}
	} else {
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(os.TempDir(), "AgentNotify")
		} else {
			base = filepath.Join(base, "AgentNotify")
		}
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	return filepath.Join(base, fileName), nil
}

func loadTTFFont(sizePt float64) (font.Face, error) {
	var fontPaths []string
	switch runtime.GOOS {
	case "windows":
		fontPaths = []string{
			`C:\Windows\Fonts\segoeui.ttf`,
			`C:\Windows\Fonts\seguisb.ttf`,
			`C:\Windows\Fonts\consola.ttf`,
			`C:\Windows\Fonts\lucon.ttf`,
		}
	case "darwin":
		fontPaths = []string{
			"/System/Library/Fonts/SFNS.ttf",
			"/System/Library/Fonts/SFNSText.ttf",
			"/Library/Fonts/Helvetica.ttc",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/System/Library/Fonts/Supplemental/HelveticaNeue.ttc",
		}
	}
	for _, path := range fontPaths {
		if face, err := tryLoadFont(path, sizePt); err == nil && face != nil {
			return face, nil
		}
	}
	return nil, fmt.Errorf("no font found")
}

func tryLoadFont(path string, sizePt float64) (font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	opts := &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     72,
		Hinting: font.HintingFull,
	}
	return opentype.NewFace(f, opts)
}

func renderToastCard(path string, card ToastCard) error {
	imgW := scale(toastCardWidth)
	imgH := scale(toastCardHeight)
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))

	bg := color.RGBA{R: 18, G: 24, B: 38, A: 255}
	panel := color.RGBA{R: 29, G: 39, B: 58, A: 255}
	accent := color.RGBA{R: 74, G: 222, B: 128, A: 255}
	if card.Event == "stop" {
		accent = color.RGBA{R: 248, G: 113, B: 113, A: 255}
	}
	eventAgent, directory, detail := toastCardSummaryLines(card)

	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	// Left accent bar
	draw.Draw(img, image.Rect(0, 0, scale(14), imgH), &image.Uniform{C: accent}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(scale(28), scale(22), scale(336), scale(158)), &image.Uniform{C: panel}, image.Point{}, draw.Src)

	badgeW := scale(96)
	draw.Draw(img, image.Rect(scale(44), scale(38), scale(44)+badgeW, scale(68)), &image.Uniform{C: accent}, image.Point{}, draw.Src)

	if face, err := loadTTFFont(38); err == nil {
		drawTextWithFace(img, scale(56), scale(59), eventLabel(card.Event), color.RGBA{R: 15, G: 23, B: 42, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(56), scale(57), eventLabel(card.Event), color.RGBA{R: 15, G: 23, B: 42, A: 255}, basicfont.Face7x13)
	}

	if face, err := loadTTFFont(52); err == nil {
		drawTextWithFace(img, scale(44), scale(98), truncateText(eventAgent, 30), color.RGBA{R: 248, G: 250, B: 252, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(44), scale(94), truncateText(eventAgent, 32), color.RGBA{R: 248, G: 250, B: 252, A: 255}, basicfont.Face7x13)
	}

	cardDirectory := "DIR: " + lastFolderName(directory)
	if face, err := loadTTFFont(42); err == nil {
		drawTextWithFace(img, scale(44), scale(128), truncateText(cardDirectory, 36), color.RGBA{R: 226, G: 232, B: 240, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(44), scale(124), truncateText(cardDirectory, 36), color.RGBA{R: 226, G: 232, B: 240, A: 255}, basicfont.Face7x13)
	}

	if face, err := loadTTFFont(30); err == nil {
		drawTextWithFace(img, scale(44), scale(148), truncateText(detail, 44), color.RGBA{R: 148, G: 163, B: 184, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(44), scale(145), truncateText(detail, 46), color.RGBA{R: 148, G: 163, B: 184, A: 255}, basicfont.Face7x13)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func renderToastAppLogo(path string) error {
	const size = 256
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bg := color.RGBA{R: 15, G: 124, B: 255, A: 255}
	bg2 := color.RGBA{R: 98, G: 180, B: 255, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			t := float64(x+y) / float64((size-1)*2)
			c := color.RGBA{
				R: uint8(float64(bg.R)*(1-t) + float64(bg2.R)*t),
				G: uint8(float64(bg.G)*(1-t) + float64(bg2.G)*t),
				B: uint8(float64(bg.B)*(1-t) + float64(bg2.B)*t),
				A: 255,
			}
			img.SetRGBA(x, y, c)
		}
	}

	if face, err := loadTTFFont(132); err == nil {
		drawTextWithFace(img, 78, 174, "A", white, face)
	} else {
		drawTextWithFace(img, 104, 138, "A", white, basicfont.Face7x13)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawTextWithFace(img *image.RGBA, x, y int, text string, c color.Color, face font.Face) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func scale(v int) int {
	return v * toastCardScale
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r && image.Pt(x, y).In(img.Bounds()) {
				img.Set(x, y, c)
			}
		}
	}
}

func toastCardSummaryLines(card ToastCard) (string, string, string) {
	eventAgent := strings.TrimSpace(card.Title)
	if eventAgent == "" {
		eventAgent = eventLabel(card.Event) + " · " + strings.TrimSpace(card.Agent)
	}

	directory := ""
	detailParts := []string{}
	for _, part := range strings.Split(card.Message, " | ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "DIR:") {
			if directory == "" {
				directory = strings.TrimSpace(strings.TrimPrefix(part, "DIR:"))
				continue
			}
			detailParts = append(detailParts, part)
			continue
		}
		detailParts = append(detailParts, part)
	}
	if directory == "" {
		directory = strings.TrimSpace(card.Project)
	}
	if directory == "" {
		directory = "unknown"
	}

	detail := strings.Join(detailParts, " | ")
	if detail == "" {
		detail = strings.TrimSpace(card.Message)
	}
	return eventAgent, directory, detail
}

func lastFolderName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "unknown"
	}
	sep := "/"
	if strings.Contains(path, `\`) && !strings.Contains(path, "/") {
		sep = `\`
	}
	parts := strings.Split(path, sep)
	// Remove empty parts from trailing separator
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}

func truncateText(s string, max int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
