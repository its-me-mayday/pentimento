package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Stato della storia condiviso tra le scene.
var story struct {
	ch2Choice int // 0 mentire a Corrado, 1 dirgli la verità, 2 chiedere del collezionista
}

// ChapterCard: cartello nero tra un capitolo e l'altro.
type ChapterCard struct {
	t          int
	title, sub string
	next       func() Scene
	ready      func() bool
	theme      int
}

func NewChapter2Card() *ChapterCard {
	PrewarmCaravaggio()
	return &ChapterCard{
		title: "Capitolo 2", sub: "La chiamata", theme: 1,
		next:  func() Scene { return NewLab2Scene(false) },
		ready: caravaggioReady,
	}
}

func caravaggioReady() bool {
	select {
	case img := <-cravReady:
		cravReady <- img
		return true
	default:
		return false
	}
}

func (s *ChapterCard) Update(g *Game) error {
	s.t++
	SetTheme(s.theme)
	SetMood(Mood{Pad: 0.9, Arp: 0.15, Tension: 0.1})
	if s.t > 40 && (s.ready == nil || s.ready()) && confirmPressed() {
		g.GoTo(s.next())
	}
	return nil
}

func (s *ChapterCard) Draw(screen *ebiten.Image) {
	screen.Fill(rgba(6, 5, 4, 1))
	a := math.Min(1, float64(s.t)/60)
	drawText(screen, s.title, faceItalic, ScreenW/2, 120, rgba(200, 180, 140, a), 0)
	drawText(screen, s.sub, faceTitle, ScreenW/2, 145, rgba(214, 178, 104, a), 0)
	vector.StrokeLine(screen, ScreenW/2-60, 225, ScreenW/2+60, 225, 1, rgba(214, 178, 104, a*0.5), false)
	if s.ready != nil && !s.ready() {
		drawText(screen, "Preparazione della tela...", faceSmall, ScreenW/2, 290, rgba(180, 170, 150, 0.7), 0)
	} else if s.t > 40 && (s.t/30)%2 == 0 {
		drawText(screen, "Clicca per continuare", faceSmall, ScreenW/2, 290, rgba(200, 190, 170, 0.8), 0)
	}
}
