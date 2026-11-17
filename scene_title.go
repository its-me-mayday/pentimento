package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type TitleScene struct {
	t   int
	sel int
}

func NewTitleScene() *TitleScene { return &TitleScene{} }

func (s *TitleScene) Update(g *Game) error {
	s.t++
	SetTheme(0)
	SetMood(Mood{Pad: 1, Arp: 0.3})
	PrewarmCaravaggio()
	if s.t < 20 {
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.sel = 1 - s.sel
	}
	start := -1
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		start = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		start = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		start = s.sel
	}
	if x, y, ok := pointer(); ok {
		if i := titleOptionAt(x, y); i >= 0 {
			s.sel = i
		}
	}
	if x, y, ok := pointerJustPressed(); ok {
		if i := titleOptionAt(x, y); i >= 0 {
			start = i
		}
	}
	switch start {
	case 0:
		g.GoTo(NewLabScene())
	case 1:
		g.GoTo(NewChapter2Card())
	}
	return nil
}

var titleOptions = []string{"Capitolo 1  —  La notte stellata", "Capitolo 2  —  La chiamata"}

func titleOptionY(i int) float64 { return 222 + float64(i)*32 }

func titleOptionAt(x, y int) int {
	for i := range titleOptions {
		ty := titleOptionY(i)
		if float64(y) >= ty-4 && float64(y) <= ty+22 && x > ScreenW/2-150 && x < ScreenW/2+150 {
			return i
		}
	}
	return -1
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	applyGrade(screen, paintingBG, 0.12+0.1*math.Sin(ft/90), ft, 0, 0)
	vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.55), false)

	gold := rgba(214, 178, 104, 1)
	drawText(screen, "PENTIMENTO", faceTitle, ScreenW/2, 92, gold, 0)
	drawText(screen, "Sotto ogni quadro c'è un altro quadro.", faceItalic, ScreenW/2, 168, rgba(230, 222, 205, 0.95), 0)

	for i, o := range titleOptions {
		y := titleOptionY(i)
		c := rgba(200, 192, 176, 0.75)
		if i == s.sel {
			c = rgba(255, 236, 190, 1)
			vector.DrawFilledRect(screen, ScreenW/2-150, float32(y)-4, 300, 26, rgba(214, 178, 104, 0.15), false)
		}
		drawText(screen, o, faceText, ScreenW/2, y, c, 0)
	}
	drawText(screen, "M: musica on/off", faceSmall, ScreenW-12, 12, rgba(180, 170, 150, 0.6), 1)
	drawText(screen, "pentiménto  s.m.  —  traccia di una versione precedente, nascosta sotto la pittura finale.",
		faceSmall, ScreenW/2, 326, rgba(180, 170, 150, 0.8), 0)
}
