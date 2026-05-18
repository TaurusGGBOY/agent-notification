package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	toastCardWidth  = 364
	toastCardHeight = 180
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

func renderToastCard(path string, card ToastCard) error {
	img := image.NewRGBA(image.Rect(0, 0, toastCardWidth, toastCardHeight))
	bg := color.RGBA{R: 18, G: 24, B: 38, A: 255}
	accent := color.RGBA{R: 74, G: 222, B: 128, A: 255}
	if card.Event == "stop" {
		accent = color.RGBA{R: 248, G: 113, B: 113, A: 255}
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, 8, toastCardHeight), &image.Uniform{C: accent}, image.Point{}, draw.Src)
	drawCircle(img, 44, 44, 24, accent)
	drawText(img, 36, 49, agentInitial(card.Agent), color.RGBA{R: 15, G: 23, B: 42, A: 255})
	drawText(img, 84, 37, truncateText(card.Title, 38), color.RGBA{R: 248, G: 250, B: 252, A: 255})
	drawText(img, 84, 65, truncateText(card.Project, 42), color.RGBA{R: 148, G: 163, B: 184, A: 255})
	drawText(img, 24, 112, truncateText(card.Message, 52), color.RGBA{R: 203, G: 213, B: 225, A: 255})
	drawText(img, 24, 148, strings.ToUpper(card.Event), accent)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawText(img *image.RGBA, x, y int, text string, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
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