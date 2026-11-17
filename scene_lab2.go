package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Capitolo 2, fuori dal quadro: Corrado Valli porta la "Vocazione di san Matteo".
// Fase detective con tre strumenti, mentre Corrado ti osserva.

type labTool int

const (
	toolUV labTool = iota
	toolRaking
	toolLens
)

var toolNames = []string{"1  UV", "2  Radente", "3  Lente"}

type evidence struct {
	wx, wy float64
	tool   labTool
	found  bool
	hover  int
	lines  []Line
}

type lab2State int

const (
	l2Intro lab2State = iota
	l2Exam
	l2Ask
	l2Choice
	l2After
	l2Enter
	l2Fail
)

type Lab2Scene struct {
	t        int
	state    lab2State
	dlg      Dialogue
	full     *ebiten.Image // dipinto a piena risoluzione (per la lente)
	tool     labTool
	px, py   int
	pointing bool
	ev       []*evidence
	susp     float64
	corrT    int
	reveal   float64
	hover    int // opzione evidenziata nella scelta
}

// coordinate mondo → schermo (dipinto sul cavalletto)
func thumbPos(wx, wy float64) (float64, float64) {
	return frameX + wx*frameW/cravW, frameY + wy*frameH/cravH
}

func NewLab2Scene(retry bool) *Lab2Scene {
	s := &Lab2Scene{}
	src := CaravaggioImage()
	s.full = ebiten.NewImage(int(cravW), int(cravH))
	applyGrade(s.full, src, 0.3, 0, 0, 0)
	s.ev = []*evidence{
		{wx: 470, wy: 245, tool: toolUV, lines: []Line{
			{"Elena", "Ritocchi recenti: sotto la UV brillano come lucciole. Nel buio sopra il tavolo disegnano un numero... 4.", false},
			{"Elena", "Copia n. 4. La Notte stellata era la numero 3. Il Collegio li conta.", false},
		}},
		{wx: 900, wy: 480, tool: toolRaking, lines: []Line{
			{"Elena", "Con la luce radente la crettatura viene fuori: crepe tutte uguali, regolari come una rete.", false},
			{"Elena", "Non è il tempo ad averle fatte. È un forno. Questo quadro ha pochi mesi.", false},
		}},
		{wx: 230, wy: 455, tool: toolLens, lines: []Line{
			{"Elena", "Su una delle monete, sotto la lente, una lettera graffiata nella pittura ancora morbida: «C.»", false},
			{"Elena", "La stessa firma della Notte stellata.", false},
		}},
	}
	if retry {
		s.state = l2Exam
		s.dlg.Start(Line{"", "Riprovi. Corrado è di nuovo al telefono, vicino alla finestra.", false})
	} else {
		s.dlg.Start(
			Line{"", "Sette giorni dopo. Laboratorio Bellandi, 23:40.", false},
			Line{"", "Bussano. Sulla porta c'è Corrado Valli: cappotto color cammello, una cassa di legno sotto il braccio.", false},
			Line{"Corrado", "Elena. Mi dispiace per Ada, davvero. Eravamo amici da trent'anni.", false},
			Line{"Corrado", "Ho un lavoro che lei aveva cominciato a settembre. Una copia d'epoca della Vocazione di san Matteo, scuola caravaggesca. Per un collezionista svizzero.", false},
			Line{"Corrado", "Pulitura e vernice, niente di più. Pago il doppio. Mi serve entro venerdì.", false},
			Line{"Elena", "«Il prossimo quadro passerà dal laboratorio fra sette giorni.» Ada lo sapeva.", false},
			Line{"Corrado", "Faccio una telefonata. Tu intanto dagli un'occhiata.", false},
			Line{"", "Esamina il dipinto con i tre strumenti (tasti 1-3 o pulsanti in alto). Quando Corrado si gira, non farti vedere con la lampada UV.", false},
		)
	}
	return s
}

// Corrado: al telefono → abbassa il telefono (avviso) → ti guarda.
const (
	corrPhone = 330
	corrWarn  = 60
	corrWatch = 170
)

func (s *Lab2Scene) corrPhase() int {
	if s.state != l2Exam {
		return 0
	}
	t := s.corrT % (corrPhone + corrWarn + corrWatch)
	switch {
	case t < corrPhone:
		return 0
	case t < corrPhone+corrWarn:
		return 1
	default:
		return 2
	}
}

func (s *Lab2Scene) overPainting(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= frameX && fx <= frameX+frameW && fy >= frameY && fy <= frameY+frameH
}

func (s *Lab2Scene) Update(g *Game) error {
	s.t++
	SetTheme(1)
	s.px, s.py, s.pointing = pointer()
	m := Mood{Pad: 0.85, Arp: 0.15, Tension: 0.1 + s.susp*0.8}
	if s.corrPhase() == 2 {
		m.Tension += 0.15
	}
	SetMood(m)

	if s.dlg.Update() {
		return nil
	}
	switch s.state {
	case l2Intro:
		s.state = l2Exam
	case l2Exam:
		s.corrT++
		s.updateTools()
		if debugCheats && inpututil.IsKeyJustPressed(ebiten.KeyN) {
			for _, e := range s.ev {
				if !e.found {
					e.hover = 99
					s.tool = e.tool
					s.px, s.py = func() (int, int) { x, y := thumbPos(e.wx, e.wy); return int(x), int(y) }()
					s.pointing = true
					break
				}
			}
		}
		if s.corrPhase() == 2 && s.tool == toolUV && s.pointing && s.overPainting(s.px, s.py) {
			s.susp += 0.012
		} else {
			s.susp = math.Max(0, s.susp-0.0008)
		}
		if s.susp >= 1 {
			s.state = l2Fail
			PlaySFX("hit")
			s.dlg.Start(
				Line{"Corrado", "Che cosa cerchi con quella lampada, Elena?", false},
				Line{"Corrado", "Lascia stare. Ada aveva le stesse manie. Me lo riprendo domattina.", false},
			)
			return nil
		}
		for _, e := range s.ev {
			if e.found {
				continue
			}
			tx, ty := thumbPos(e.wx, e.wy)
			if s.tool == e.tool && s.pointing && math.Hypot(float64(s.px)-tx, float64(s.py)-ty) < 16 {
				e.hover++
			} else if e.hover > 0 {
				e.hover--
			}
			if e.hover > 40 {
				e.found = true
				PlaySFX("evidence")
				s.dlg.Start(e.lines...)
				return nil
			}
		}
		if s.allFound() {
			s.state = l2Ask
			s.dlg.Start(Line{"Corrado", "Allora? Che cosa ne pensi?", false})
		}
	case l2Ask:
		s.state = l2Choice
	case l2Choice:
		s.updateChoice()
	case l2After:
		s.state = l2Enter
	case l2Enter:
		s.reveal = math.Min(1, s.reveal+0.02)
		s.tool = toolUV
		if s.reveal < 0.3 {
			break
		}
		if x, y, ok := pointerJustPressed(); ok && s.overPainting(x, y) {
			g.GoTo(NewPainting2Scene(false))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyE) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.GoTo(NewPainting2Scene(false))
		}
	case l2Fail:
		if confirmPressed() {
			g.GoTo(NewLab2Scene(true))
		}
	}
	return nil
}

func (s *Lab2Scene) allFound() bool {
	for _, e := range s.ev {
		if !e.found {
			return false
		}
	}
	return true
}

func toolButtonRect(i int) (float64, float64, float64, float64) {
	return 12 + float64(i)*84, 10, 78, 22
}

func (s *Lab2Scene) updateTools() {
	keys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3}
	for i, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			s.tool = labTool(i)
		}
	}
	if x, y, ok := pointerJustPressed(); ok {
		for i := range toolNames {
			bx, by, bw, bh := toolButtonRect(i)
			if float64(x) >= bx && float64(x) <= bx+bw && float64(y) >= by && float64(y) <= by+bh {
				s.tool = labTool(i)
			}
		}
	}
}

var ch2Options = []string{
	"«È un'ottima copia d'epoca. Venerdì è pronta.»",
	"«Non è d'epoca. La crettatura l'hanno fatta in forno.»",
	"«Chi è il collezionista svizzero?»",
}

func choiceRect(i int) (float64, float64, float64, float64) {
	return 60, 200 + float64(i)*40, ScreenW - 120, 32
}

func (s *Lab2Scene) updateChoice() {
	pick := -1
	for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3} {
		if inpututil.IsKeyJustPressed(k) {
			pick = i
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		s.hover = (s.hover + 1) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.hover = (s.hover + 2) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		pick = s.hover
	}
	for i := range ch2Options {
		bx, by, bw, bh := choiceRect(i)
		inside := float64(s.px) >= bx && float64(s.px) <= bx+bw && float64(s.py) >= by && float64(s.py) <= by+bh
		if inside {
			s.hover = i
		}
	}
	if x, y, ok := pointerJustPressed(); ok {
		for i := range ch2Options {
			bx, by, bw, bh := choiceRect(i)
			if float64(x) >= bx && float64(x) <= bx+bw && float64(y) >= by && float64(y) <= by+bh {
				pick = i
			}
		}
	}
	if pick < 0 {
		return
	}
	story.ch2Choice = pick
	s.state = l2After
	lines := []Line{{"Elena", ch2Options[pick], false}}
	switch pick {
	case 0:
		lines = append(lines, Line{"Corrado", "Sapevo di potermi fidare. Ada ti ha insegnato bene.", false})
	case 1:
		lines = append(lines,
			Line{"", "Corrado resta in silenzio un secondo di troppo.", false},
			Line{"Corrado", "...Ada diceva le stesse cose. Stai attenta, Elena. Certe parole costano care.", false})
	case 2:
		lines = append(lines, Line{"Corrado", "Uno che paga e non fa domande. Ti consiglio di fare lo stesso.", false})
	}
	lines = append(lines,
		Line{"", "Corrado se ne va. La porta si chiude. Sei sola con il quadro.", false},
		Line{"Elena", "Ci hai lavorato a settembre, Ada. Allora hai lasciato qualcosa anche qui.", false},
	)
	s.dlg.Start(lines...)
}

func (s *Lab2Scene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	drawLabRoom(screen)
	if s.state < l2After {
		s.drawCorrado(screen, ft)
	}
	drawFramedThumb(screen, s.full)

	if s.state == l2Exam || s.state == l2Enter {
		s.drawTool(screen, ft)
	}
	for _, e := range s.ev {
		if e.found {
			s.drawEvidence(screen, e, 0.55+0.25*math.Sin(ft/12))
		}
	}

	if s.state == l2Exam {
		for i, n := range toolNames {
			bx, by, bw, bh := toolButtonRect(i)
			sel := labTool(i) == s.tool
			bg := rgba(30, 24, 20, 0.85)
			if sel {
				bg = rgba(90, 60, 130, 0.9)
			}
			vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bw), float32(bh), bg, false)
			vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1, rgba(214, 178, 104, 0.7), false)
			drawText(screen, n, faceSmall, bx+bw/2, by+4, rgba(240, 230, 220, 1), 0)
		}
		found := 0
		for _, e := range s.ev {
			if e.found {
				found++
			}
		}
		drawText(screen, "Prove: "+string(rune('0'+found))+"/3", faceSmall, 270, 14, rgba(220, 210, 190, 0.9), -1)
		// sospetto di Corrado
		drawText(screen, "Corrado", faceSmall, ScreenW-120, 12, rgba(220, 200, 180, 0.9), 1)
		vector.StrokeRect(screen, ScreenW-112, 14, 96, 10, 1, rgba(220, 120, 100, 0.8), false)
		vector.DrawFilledRect(screen, ScreenW-110, 16, float32(92*s.susp), 6, rgba(220, 80, 60, 0.9), false)
		switch s.corrPhase() {
		case 1:
			drawText(screen, "Corrado abbassa il telefono...", faceSmall, ScreenW-16, 30, rgba(255, 200, 150, 0.9), 1)
		case 2:
			drawText(screen, "Corrado ti sta guardando", faceSmall, ScreenW-16, 30, rgba(255, 120, 100, 1), 1)
			vector.StrokeRect(screen, 1, 1, ScreenW-2, ScreenH-2, 3, rgba(200, 60, 40, 0.35+0.15*math.Sin(ft/6)), false)
		}
	}
	if s.state == l2Choice {
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.55), false)
		drawText(screen, "Che cosa rispondi a Corrado?", faceText, ScreenW/2, 165, rgba(236, 228, 214, 1), 0)
		for i, o := range ch2Options {
			bx, by, bw, bh := choiceRect(i)
			bg := rgba(20, 16, 14, 0.9)
			if i == s.hover {
				bg = rgba(70, 50, 30, 0.95)
			}
			vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bw), float32(bh), bg, false)
			vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1, rgba(214, 178, 104, 0.8), false)
			drawText(screen, string(rune('1'+i))+".  "+o, faceText, bx+14, by+7, rgba(240, 232, 220, 1), -1)
		}
	}
	if s.state == l2Enter {
		if (s.t/25)%2 == 0 {
			drawText(screen, "Clicca sul dipinto per entrarci", faceText, ScreenW/2, 12, rgba(200, 170, 255, 1), 0)
		}
		vector.StrokeRect(screen, frameX-14, frameY-14, frameW+28, frameH+28, 2, rgba(150, 100, 255, 0.3+0.3*math.Sin(ft/8)), true)
	}
	if s.state == l2Fail && !s.dlg.Active() {
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.75), false)
		drawText(screen, "Corrado si riprende il quadro.", faceTitle, ScreenW/2, 110, rgba(220, 180, 160, 0.9), 0)
		if (s.t/30)%2 == 0 {
			drawText(screen, "Clicca per riprovare", faceText, ScreenW/2, 230, rgba(230, 220, 210, 0.9), 0)
		}
	}
	s.dlg.Draw(screen)
}

func (s *Lab2Scene) drawCorrado(screen *ebiten.Image, ft float64) {
	const cx, top = 570.0, 118.0
	coat := rgba(128, 98, 66, 1)
	// cappotto
	var p vector.Path
	p.MoveTo(cx-16, top+24)
	p.LineTo(cx+16, top+24)
	p.LineTo(cx+24, 288)
	p.LineTo(cx-24, 288)
	p.Close()
	fillPath(screen, &p, coat)
	vector.StrokeLine(screen, cx, top+28, cx+2, 288, 1.5, rgba(90, 66, 44, 1), true)
	vector.DrawFilledRect(screen, cx-8, top+20, 16, 6, rgba(200, 196, 190, 1), true) // camicia
	// testa: capelli grigi alle tempie
	vector.DrawFilledCircle(screen, cx, top+8, 11, rgba(196, 152, 118, 1), true)
	vector.DrawFilledCircle(screen, cx, top+2, 10, rgba(150, 146, 140, 1), true)
	vector.DrawFilledCircle(screen, cx, top+10, 9, rgba(196, 152, 118, 1), true)
	switch s.corrPhase() {
	case 0: // al telefono, di spalle, verso la finestra
		vector.DrawFilledRect(screen, cx+8, top+2, 4, 9, rgba(150, 200, 255, 0.9), true)
		vector.StrokeLine(screen, cx+14, top+30, cx+10, top+8, 4, coat, true)
	case 1: // abbassa il telefono
		vector.DrawFilledRect(screen, cx+16, top+48, 4, 9, rgba(150, 200, 255, 0.9), true)
		vector.DrawFilledCircle(screen, cx-3, top+9, 1.2, rgba(30, 20, 15, 1), true)
	case 2: // ti guarda
		vector.DrawFilledCircle(screen, cx-6, top+9, 1.5, rgba(30, 20, 15, 1), true)
		vector.DrawFilledCircle(screen, cx+1, top+9, 1.5, rgba(30, 20, 15, 1), true)
		vector.StrokeLine(screen, cx-8, top+5, cx-4, top+6, 1, rgba(60, 50, 45, 1), true)
	}
	// anello con sigillo
	vector.DrawFilledCircle(screen, cx-20, top+92, 2, rgba(220, 190, 90, 1), true)
}

func (s *Lab2Scene) drawTool(screen *ebiten.Image, ft float64) {
	if !s.pointing || !s.overPainting(s.px, s.py) {
		return
	}
	x, y := float64(s.px), float64(s.py)
	switch s.tool {
	case toolUV:
		r := 30.0 + s.reveal*40
		for k := 6; k >= 1; k-- {
			vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r*float64(k)/6), rgba(120, 60, 255, 0.07), true)
		}
		s.revealNear(screen, toolUV, x, y, r)
	case toolRaking:
		r := 34.0
		for i := -5; i <= 5; i++ {
			o := float64(i) * 6
			vector.StrokeLine(screen, float32(x-r+o*0.3), float32(y+o-8), float32(x+r+o*0.3), float32(y+o+8), 2, rgba(255, 240, 210, 0.08), true)
		}
		vector.StrokeCircle(screen, float32(x), float32(y), float32(r), 1, rgba(255, 240, 210, 0.4), true)
		s.revealNear(screen, toolRaking, x, y, r)
	case toolLens:
		// lente: ingrandisce il dipinto a piena risoluzione
		const half = 40.0
		wx := (x - frameX) * cravW / frameW
		wy := (y - frameY) * cravH / frameH
		src := 70.0
		sub := s.full.SubImage(rectAround(wx, wy, src)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		b := sub.Bounds()
		op.GeoM.Scale(2*half/float64(b.Dx()), 2*half/float64(b.Dy()))
		op.GeoM.Translate(x-half, y-half)
		op.Filter = ebiten.FilterLinear
		screen.DrawImage(sub, op)
		vector.StrokeRect(screen, float32(x-half), float32(y-half), 2*half, 2*half, 3, rgba(60, 60, 70, 1), false)
		vector.StrokeRect(screen, float32(x-half), float32(y-half), 2*half, 2*half, 1, rgba(200, 210, 230, 0.8), false)
		// la «C.» sulla moneta si legge solo ingrandita
		if e := s.ev[2]; math.Abs(e.wx-wx) < src/2 && math.Abs(e.wy-wy) < src/2 {
			lx := x + (e.wx-wx)*2*half/src
			ly := y + (e.wy-wy)*2*half/src
			drawText(screen, "C.", faceBold, lx, ly-16, rgba(40, 20, 10, 0.95), 0)
		}
	}
}

func rectAround(cx, cy, size float64) image.Rectangle {
	x0 := int(clamp(cx-size/2, 0, cravW-size))
	y0 := int(clamp(cy-size/2, 0, cravH-size))
	return image.Rect(x0, y0, x0+int(size), y0+int(size))
}

// revealNear mostra la prova legata allo strumento quando lo strumento ci passa sopra.
func (s *Lab2Scene) revealNear(screen *ebiten.Image, t labTool, x, y, r float64) {
	for _, e := range s.ev {
		if e.tool != t || e.tool == toolLens {
			continue
		}
		tx, ty := thumbPos(e.wx, e.wy)
		a := clamp(1-math.Hypot(x-tx, y-ty)/r, 0, 1)
		if a > 0 {
			s.drawEvidence(screen, e, a)
		}
	}
}

func (s *Lab2Scene) drawEvidence(screen *ebiten.Image, e *evidence, a float64) {
	tx, ty := thumbPos(e.wx, e.wy)
	switch e.tool {
	case toolUV:
		drawText(screen, "4", faceBold, tx, ty-9, rgba(170, 255, 240, a), 0)
		vector.StrokeCircle(screen, float32(tx), float32(ty), 11, 1, rgba(170, 255, 240, a*0.5), true)
	case toolRaking:
		// crettatura regolare, a rete
		for i := -2; i <= 2; i++ {
			for j := -2; j <= 2; j++ {
				cx, cy := tx+float64(i)*7, ty+float64(j)*7+float64(i%2)*3.5
				vector.StrokeCircle(screen, float32(cx), float32(cy), 3.8, 0.8, rgba(255, 245, 220, a*0.7), true)
			}
		}
	case toolLens:
		if e.found {
			vector.StrokeCircle(screen, float32(tx), float32(ty), 6, 1, rgba(255, 220, 120, a), true)
		}
	}
}
