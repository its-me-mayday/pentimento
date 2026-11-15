package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type TitleScene struct{ t int }

func NewTitleScene() *TitleScene { return &TitleScene{} }

func (s *TitleScene) Update(g *Game) error {
	s.t++
	SetMood(Mood{Pad: 1, Arp: 0.3})
	if s.t > 20 && confirmPressed() {
		g.GoTo(NewLabScene())
	}
	return nil
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	applyGrade(screen, paintingBG, 0.12+0.1*math.Sin(ft/90), ft, 0, 0)
	vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.55), false)

	gold := rgba(214, 178, 104, 1)
	drawText(screen, "PENTIMENTO", faceTitle, ScreenW/2, 92, gold, 0)
	drawText(screen, "Sotto ogni quadro c'è un altro quadro.", faceItalic, ScreenW/2, 168, rgba(230, 222, 205, 0.95), 0)

	if (s.t/30)%2 == 0 {
		drawText(screen, "Clicca o premi INVIO per iniziare", faceText, ScreenW/2, 232, rgba(240, 240, 240, 0.9), 0)
	}
	drawText(screen, "M: musica on/off", faceSmall, ScreenW-12, 12, rgba(180, 170, 150, 0.6), 1)
	drawText(screen, "pentiménto  s.m.  —  traccia di una versione precedente, nascosta sotto la pittura finale.",
		faceSmall, ScreenW/2, 326, rgba(180, 170, 150, 0.8), 0)
}
