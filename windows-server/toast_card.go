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
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = filepath.Join(os.TempDir(), "AgentNotify")
	} else {
		base = filepath.Join(base, "AgentNotify")
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	return filepath.Join(base, "toast-card.png"), nil
}

func loadTTFFont(sizePt float64) (font.Face, error) {
	var fontPaths []string
	if runtime.GOOS == "windows" {
		fontPaths = []string{
			`C:\Windows\Fonts\segoeui.ttf`,
			`C:\Windows\Fonts\seguisb.ttf`,
			`C:\Windows\Fonts\consola.ttf`,
			`C:\Windows\Fonts\lucon.ttf`,
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
	accent := color.RGBA{R: 74, G: 222, B: 128, A: 255}
	if card.Event == "stop" {
		accent = color.RGBA{R: 248, G: 113, B: 113, A: 255}
	}

	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	// Left accent bar
	draw.Draw(img, image.Rect(0, 0, scale(16), imgH), &image.Uniform{C: accent}, image.Point{}, draw.Src)

	// Avatar circle
	avatarR := scale(36)
	avatarX := scale(70)
	avatarY := scale(70)
	drawCircle(img, avatarX, avatarY, avatarR, accent)

	// Agent initial - try custom font first, fallback to basicfont
	if face, err := loadTTFFont(72); err == nil {
		drawTextWithFace(img, avatarX-scale(15), avatarY+scale(12), agentInitial(card.Agent), color.RGBA{R: 15, G: 23, B: 42, A: 255}, face)
	} else {
		drawTextWithFace(img, avatarX-scale(10), avatarY+scale(8), agentInitial(card.Agent), color.RGBA{R: 15, G: 23, B: 42, A: 255}, basicfont.Face7x13)
	}

	// Title - large and prominent
	if face, err := loadTTFFont(72); err == nil {
		drawTextWithFace(img, scale(140), scale(70), truncateText(card.Title, 18), color.RGBA{R: 248, G: 250, B: 252, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(140), scale(65), truncateText(card.Title, 20), color.RGBA{R: 248, G: 250, B: 252, A: 255}, basicfont.Face7x13)
	}

	// Project - medium
	if face, err := loadTTFFont(56); err == nil {
		drawTextWithFace(img, scale(140), scale(110), truncateText(card.Project, 22), color.RGBA{R: 148, G: 163, B: 184, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(140), scale(105), truncateText(card.Project, 25), color.RGBA{R: 148, G: 163, B: 184, A: 255}, basicfont.Face7x13)
	}

	// Message - readable size
	if face, err := loadTTFFont(48); err == nil {
		drawTextWithFace(img, scale(30), scale(150), truncateText(card.Message, 40), color.RGBA{R: 203, G: 213, B: 225, A: 255}, face)
	} else {
		drawTextWithFace(img, scale(30), scale(145), truncateText(card.Message, 45), color.RGBA{R: 203, G: 213, B: 225, A: 255}, basicfont.Face7x13)
	}

	// Event badge
	if face, err := loadTTFFont(44); err == nil {
		drawTextWithFace(img, scale(30), scale(175), strings.ToUpper(card.Event), accent, face)
	} else {
		drawTextWithFace(img, scale(30), scale(170), strings.ToUpper(card.Event), accent, basicfont.Face7x13)
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