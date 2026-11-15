package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Laboratorio Bellandi, Trastevere. Fase "detective": la lampada UV.

const (
	frameX, frameY = 170.0, 46.0
	frameW         = 300.0
	frameH         = frameW * ScreenH / ScreenW
)

// posizione del segno nascosto (coordinate schermo)
const markX, markY = frameX + frameW*0.79, frameY + frameH*0.80

type labState int

const (
	labIntro labState = iota
	labSearch
	labFound
	labEnter
)

type LabScene struct {
	t        int
	state    labState
	dlg      Dialogue
	thumb    *ebiten.Image
	hover    int // frame passati con la lampada sopra il segno
	reveal   float64
	px, py   int
	pointing bool
}

func NewLabScene() *LabScene {
	s := &LabScene{}
	s.thumb = ebiten.NewImage(ScreenW, ScreenH)
	applyGrade(s.thumb, paintingBG, 0.22, 0, 0, 0)
	s.dlg.Start(
		Line{"", "Roma, Trastevere. Laboratorio di restauro Bellandi. Le 2:14 di notte.", false},
		Line{"Elena", "Tre giorni fa Ada è morta qui dentro. «Esalazioni di solvente», ha detto la polizia. Un incidente.", false},
		Line{"Elena", "Ada lavorava con il solvente da quarant'anni. Non ha mai lasciato una finestra chiusa.", false},
		Line{"Elena", "Questo è l'ultimo quadro che stava restaurando. Nessuno sa chi l'ha portato.", false},
		Line{"Elena", "La sua lampada UV era ancora accesa, puntata qui. Vediamo cosa stava guardando.", false},
	)
	return s
}

func (s *LabScene) Update(g *Game) error {
	s.t++
	s.px, s.py, s.pointing = pointer()

	if s.dlg.Update() {
		return nil
	}
	switch s.state {
	case labIntro:
		s.state = labSearch
	case labSearch:
		if s.pointing && s.overPainting(s.px, s.py) &&
			math.Hypot(float64(s.px)-markX, float64(s.py)-markY) < 18 {
			s.hover++
		} else if s.hover > 0 {
			s.hover--
		}
		if s.hover > 40 {
			s.state = labFound
			s.dlg.Start(
				Line{"Elena", "Una «A.» incisa sotto la vernice. È la sigla di Ada... ma non è una firma da restauratrice.", false},
				Line{"Elena", "È un segno lasciato apposta. Per qualcuno che sapesse dove cercare.", false},
				Line{"", "La lampada ronza. La luce viola si allarga, e la tela sembra... respirare.", false},
			)
		}
	case labFound:
		s.state = labEnter
	case labEnter:
		s.reveal = math.Min(1, s.reveal+0.02)
		if s.reveal < 0.3 {
			break // piccola pausa: evita di entrare per sbaglio chiudendo il dialogo
		}
		if x, y, ok := pointerJustPressed(); ok && s.overPainting(x, y) {
			g.GoTo(NewPaintingScene())
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyE) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.GoTo(NewPaintingScene())
		}
	}
	return nil
}

func (s *LabScene) overPainting(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= frameX && fx <= frameX+frameW && fy >= frameY && fy <= frameY+frameH
}

func (s *LabScene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	// stanza
	for y := 0; y < ScreenH; y += 3 {
		t := float64(y) / ScreenH
		vector.DrawFilledRect(screen, 0, float32(y), ScreenW, 3, lerpColor(rgba(34, 26, 22, 1), rgba(14, 10, 9, 1), t), false)
	}
	// finestra con luce lunare
	vector.DrawFilledRect(screen, 535, 40, 70, 110, rgba(30, 46, 80, 1), false)
	vector.StrokeRect(screen, 535, 40, 70, 110, 3, rgba(20, 15, 12, 1), false)
	vector.StrokeLine(screen, 570, 40, 570, 150, 3, rgba(20, 15, 12, 1), false)
	var shaft vector.Path
	shaft.MoveTo(535, 150)
	shaft.LineTo(605, 150)
	shaft.LineTo(520, 360)
	shaft.LineTo(400, 360)
	shaft.Close()
	fillPath(screen, &shaft, rgba(90, 120, 180, 0.07))

	// tavolo da lavoro e attrezzi
	vector.DrawFilledRect(screen, 0, 290, ScreenW, 70, rgba(46, 32, 24, 1), false)
	vector.DrawFilledRect(screen, 0, 288, ScreenW, 4, rgba(70, 50, 36, 1), false)
	for i, c := range []struct{ r, g, b uint8 }{{120, 160, 150}, {150, 110, 70}, {90, 100, 140}} {
		x := float32(40 + i*26)
		vector.DrawFilledRect(screen, x, 262, 14, 28, rgba(c.r, c.g, c.b, 0.75), true)
		vector.DrawFilledRect(screen, x+4, 255, 6, 8, rgba(30, 25, 20, 1), true)
	}

	// cavalletto
	vector.StrokeLine(screen, 210, 360, 250, 40, 5, rgba(70, 48, 30, 1), true)
	vector.StrokeLine(screen, 430, 360, 390, 40, 5, rgba(70, 48, 30, 1), true)

	// cornice e quadro
	vector.DrawFilledRect(screen, frameX-10, frameY-10, frameW+20, frameH+20, rgba(110, 80, 40, 1), true)
	vector.StrokeRect(screen, frameX-6, frameY-6, frameW+12, frameH+12, 2, rgba(170, 130, 70, 1), true)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(frameW/ScreenW, frameH/ScreenH)
	op.GeoM.Translate(frameX, frameY)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(s.thumb, op)
	// craquelure
	for _, c := range [][4]float32{{200, 60, 230, 95}, {230, 95, 222, 130}, {380, 150, 420, 160}, {420, 160, 440, 190}, {300, 70, 310, 100}} {
		vector.StrokeLine(screen, c[0], c[1], c[2], c[3], 1, rgba(20, 15, 10, 0.6), true)
	}

	// lampada UV
	uvOn := s.state == labSearch || s.state == labEnter
	if uvOn && s.pointing && s.overPainting(s.px, s.py) {
		r := 30.0
		if s.state == labEnter {
			r += s.reveal * 40
		}
		for k := 6; k >= 1; k-- {
			vector.DrawFilledCircle(screen, float32(s.px), float32(s.py), float32(r*float64(k)/6), rgba(120, 60, 255, 0.07), true)
		}
		d := math.Hypot(float64(s.px)-markX, float64(s.py)-markY)
		a := clamp(1-d/r, 0, 1)
		if a > 0 {
			glow := rgba(170, 255, 240, a)
			drawText(screen, "A.", faceBold, markX, markY-9, glow, 0)
			vector.StrokeCircle(screen, markX, markY, 11, 1, rgba(170, 255, 240, a*0.5), true)
		}
	}
	if s.state == labFound || s.state == labEnter {
		drawText(screen, "A.", faceBold, markX, markY-9, rgba(170, 255, 240, 0.6+0.3*math.Sin(ft/10)), 0)
	}

	// istruzioni
	switch s.state {
	case labSearch:
		drawText(screen, "Muovi la lampada UV sul dipinto (mouse o dito). Cerca ciò che Ada ha nascosto.", faceSmall, ScreenW/2, 12, rgba(220, 200, 255, 0.85), 0)
	case labEnter:
		if (s.t/25)%2 == 0 {
			drawText(screen, "Clicca sul dipinto per entrarci", faceText, ScreenW/2, 12, rgba(200, 170, 255, 1), 0)
		}
		vector.StrokeRect(screen, frameX-14, frameY-14, frameW+28, frameH+28, 2, rgba(150, 100, 255, 0.3+0.3*math.Sin(ft/8)), true)
	}

	s.dlg.Draw(screen)
}
