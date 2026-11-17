package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Capitolo 2 — dentro "La vocazione di san Matteo".
// Qui la luce non rivela: chiama. Il fascio della finestra, gli sguardi dei gabellieri
// e il gesto di Cristo scoprono chi ci finisce dentro. La lampada UV rivela i pigmenti,
// ma rende Elena più visibile.

const pZinco pigment = 10

type gaze struct {
	x, y          float64
	a0, a1        float64 // gradi, 0 = destra, 90 = giù
	half, reach   float64
	period, phase int
}

func (g gaze) angle(t int) float64 {
	p := float64((t+g.phase)%g.period) / float64(g.period)
	k := 0.5 - 0.5*math.Cos(2*math.Pi*p)
	return (g.a0 + (g.a1-g.a0)*k) * math.Pi / 180
}

func (g gaze) sees(t int, x, y float64) bool {
	dx, dy := x-g.x, y-g.y
	d := math.Hypot(dx, dy)
	if d > g.reach || d < 1 {
		return false
	}
	diff := math.Atan2(dy, dx) - g.angle(t)
	diff = math.Atan2(math.Sin(diff), math.Cos(diff))
	return math.Abs(diff) < g.half*math.Pi/180
}

const (
	rayIdle   = 380
	rayWarn   = 100
	rayActive = 75
	rayPeriod = rayIdle + rayWarn + rayActive
)

type Painting2Scene struct {
	t     int
	phase paintPhase
	dlg   Dialogue
	off   *ebiten.Image
	bg    *ebiten.Image
	rng   *rand.Rand

	el        *Elena
	x, y      float64
	tx, ty    float64
	hasTarget bool
	camX      float64
	camY      float64
	lampOn    bool

	frags   []*fragment
	taken   int
	clueIdx int
	gazes   []gaze

	checkpoints [][2]float64
	cp          int
	lives       int
	exposure    float64
	caughtT     int
	graceT      int

	restore, shake, revealT float64
	hint                    string
	hintT                   int
	hintSeen                map[string]bool
}

func ch2Clues() []string {
	first := "Elena. Se sei qui, Corrado ti ha portato il quadro. Ci ho lavorato a settembre. Qualunque cosa vedi, non dirgliela."
	return []string{
		first,
		"Caravaggio dipingeva le persone nell'attimo in cui vengono scelte. Il Collegio sceglie così: musei piccoli, prestiti lunghi, direttori stanchi.",
		"Corrado mi pagava da tre anni. Credevo fosse solo un perito avido. Poi ho visto la sua faccia sotto la pittura.",
	}
}

const zincoClue = "La camicia bianca del ragazzo: bianco di zinco. In commercio dal 1834. Caravaggio usava la biacca. Un'altra copia, Elena."

func NewPainting2Scene(retry bool) *Painting2Scene {
	s := &Painting2Scene{
		off:      ebiten.NewImage(ScreenW, ScreenH),
		bg:       CaravaggioImage(),
		rng:      rand.New(rand.NewSource(1600)),
		lives:    maxLives,
		hintSeen: map[string]bool{},
		checkpoints: [][2]float64{
			{70, 650}, {60, 190}, {720, 660}, {1180, 650},
		},
	}
	s.x, s.y = s.checkpoints[0][0], s.checkpoints[0][1]
	s.el = NewElena(s.x, s.y)
	s.camX = 0
	s.camY = clamp(s.y-ScreenH*0.5, 0, cravH-ScreenH)

	defs := []struct {
		kind    pigment
		name    string
		r, g, b uint8
		x, y    float64
	}{
		{pNero, "Terra d'ombra", 120, 80, 46, 75, 300},
		{pBlu, "Vermiglione", 215, 45, 32, 405, 560},
		{pGiallo, "Ocra gialla", 214, 160, 60, 615, 205},
		{pZinco, "Bianco di zinco", 245, 245, 240, 760, 440},
	}
	for _, d := range defs {
		s.frags = append(s.frags, &fragment{x: d.x, y: d.y, kind: d.kind, name: d.name, r: d.r, g: d.g, b: d.b})
	}
	s.gazes = []gaze{
		{x: 300, y: 355, a0: 150, a1: 205, half: 17, reach: 270, period: 420},
		{x: 650, y: 380, a0: 95, a1: 165, half: 17, reach: 320, period: 360, phase: 120},
		{x: 535, y: 352, a0: 5, a1: 55, half: 15, reach: 300, period: 480, phase: 260},
	}

	if retry {
		s.dlg.Start(Line{"", "Riapri gli occhi nell'ombra. I pigmenti sono tornati dove li avevi trovati.", false})
	} else {
		s.dlg.Start(
			Line{"", "Buio. Odore di vino e di cera fredda. Un fascio di luce taglia la stanza dall'alto.", false},
			Line{"Elena", "La vocazione di san Matteo... Qui la luce non rivela niente. Chiama.", false},
			Line{"", "Resta nell'ombra: se il fascio della finestra o uno sguardo ti tengono addosso troppo a lungo, vieni «chiamata».", false},
			Line{"", "La lampada (L, SPAZIO o il pulsante) rivela i pigmenti, ma ti rende più visibile. La mano di Cristo, ogni tanto, indica tutta la stanza: guardala.", false},
		)
	}
	return s
}

func (s *Painting2Scene) say(key, msg string) {
	if key != "" {
		if s.hintSeen[key] {
			return
		}
		s.hintSeen[key] = true
	}
	s.hint, s.hintT = msg, 240
}

func (s *Painting2Scene) sweep() float64 { return 0.07 * math.Sin(float64(s.t)/170) }

func (s *Painting2Scene) rayState() int { // 0 riposo, 1 avviso, 2 attivo
	p := s.t % rayPeriod
	switch {
	case p < rayIdle:
		return 0
	case p < rayIdle+rayWarn:
		return 1
	default:
		return 2
	}
}

// lit: Elena (con un margine se la lampada è accesa) è in una zona di luce?
func (s *Painting2Scene) lit(x, y float64) bool {
	m := 0.0
	if s.lampOn {
		m = 16
	}
	bp := beamPolygon(s.sweep())
	pts := [][2]float64{{x, y - 14}, {x + m, y - 14}, {x - m, y - 14}, {x, y - 14 - m}, {x, y - 14 + m}}
	for _, p := range pts {
		if pointInQuad(p[0], p[1], bp) {
			return true
		}
		for _, g := range s.gazes {
			if g.sees(s.t, p[0], p[1]) {
				return true
			}
		}
		if s.rayState() == 2 && p[0] < cravHand[0] && math.Abs(p[1]-cravHand[1]) < 24 {
			return true
		}
	}
	return false
}

func (s *Painting2Scene) Update(g *Game) error {
	s.t++
	SetTheme(1)
	target := float64(s.taken) / float64(len(s.frags))
	s.restore += (target - s.restore) * 0.03
	s.shake *= 0.9
	if s.hintT > 0 {
		s.hintT--
	}
	if s.caughtT > 0 {
		s.caughtT--
	}
	s.updateMood()
	s.updateCamera()

	if s.dlg.Update() {
		return nil
	}
	switch s.phase {
	case phEnter:
		s.phase = phPlay
	case phPlay:
		if debugCheats && inpututil.IsKeyJustPressed(ebiten.KeyN) {
			for _, f := range s.frags {
				if !f.taken {
					s.x, s.y = f.x+28, f.y+10
					break
				}
			}
		}
		if debugCheats && inpututil.IsKeyJustPressed(ebiten.KeyR) {
			for _, f := range s.frags[1:] {
				f.taken = true
			}
			s.taken, s.clueIdx = len(s.frags)-1, 2
			s.x, s.y = s.frags[0].x, s.frags[0].y+10
		}
		s.updatePlayer()
		s.updateLampAndLight()
		s.checkFragments()
	case phReveal:
		s.revealT = math.Min(1, s.revealT+0.005)
		if s.revealT >= 1 {
			s.phase = phEnd
			s.dlg.Start(s.endingLines()...)
		}
	case phEnd:
		if !s.dlg.Active() && confirmPressed() {
			g.GoTo(NewTitleScene())
		}
	case phLost:
		if confirmPressed() {
			g.GoTo(NewPainting2Scene(true))
		}
	}
	return nil
}

func (s *Painting2Scene) endingLines() []Line {
	lines := []Line{
		{"Elena", "Sotto il ragazzo che conta le monete c'è un altro volto. Tempie grigie. Un anello con il sigillo: «C.V.»", false},
		{"Elena", "Corrado Valli.", false},
		{"Ada", "È lui che porta i quadri, paga i restauratori e firma i certificati. Ma non dipinge, Elena. Le mani del Collegio sono altre.", true},
		{"Ada", "La prossima copia partirà per Ginevra, per un'asta. Se arriva là, l'originale sparisce per sempre.", true},
	}
	switch story.ch2Choice {
	case 0:
		lines = append(lines, Line{"Elena", "Corrado crede che io non sappia niente. È il mio unico vantaggio.", false})
	case 1:
		lines = append(lines, Line{"Elena", "E adesso Corrado sa che io so. Come lo sapeva di te, Ada.", false})
	case 2:
		lines = append(lines, Line{"Elena", "Il collezionista svizzero... Ginevra. Ecco chi compra.", false})
	}
	return lines
}

func (s *Painting2Scene) updateCamera() {
	fx, fy := s.x, s.y
	if s.phase == phReveal || s.phase == phEnd {
		fx, fy = 300, 380
	}
	tx := clamp(fx-ScreenW/2, 0, cravW-ScreenW)
	ty := clamp(fy-ScreenH*0.5, 0, cravH-ScreenH)
	s.camX += (tx - s.camX) * 0.08
	s.camY += (ty - s.camY) * 0.08
}

func (s *Painting2Scene) updateMood() {
	m := Mood{Pad: 0.9, Arp: 0.2 + 0.6*s.restore, Bright: s.restore * 0.7, Tension: 0.12 + s.exposure*0.9}
	if s.rayState() == 1 {
		m.Tension += 0.2
	}
	switch s.phase {
	case phEnd, phReveal:
		m = Mood{Pad: 1, Arp: 0.6, Bright: 0.8}
	case phLost:
		m = Mood{Pad: 0.6, Tension: 0.3}
	}
	SetMood(m)
}

func (s *Painting2Scene) onLampButton(x, y int) bool {
	return math.Hypot(float64(x)-flashBX, float64(y)-flashBY) < flashBR+6
}

func (s *Painting2Scene) updatePlayer() {
	ox, oy := s.x, s.y
	defer func() {
		var pal []color.NRGBA
		for _, f := range s.frags {
			if f.taken {
				pal = append(pal, color.NRGBA{f.r, f.g, f.b, 255})
			}
		}
		s.el.Step(s.x-ox, s.y-oy, pal)
		s.el.X, s.el.Y = s.x, s.y
	}()
	const speed = 1.7
	dx, dy := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dx--
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dx++
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dy--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dy++
	}
	if dx != 0 || dy != 0 {
		s.hasTarget = false
	}
	if x, y, ok := pointer(); ok && !s.onLampButton(x, y) &&
		(ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.AppendTouchIDs(nil)) > 0) {
		s.tx, s.ty, s.hasTarget = float64(x)+s.camX, float64(y)+s.camY, true
	}
	if s.hasTarget {
		vx, vy := s.tx-s.x, s.ty-s.y
		if d := math.Hypot(vx, vy); d < 2 {
			s.hasTarget = false
		} else {
			dx, dy = vx/d, vy/d
		}
	}
	if l := math.Hypot(dx, dy); l > 0 {
		s.x += dx / l * speed
		s.y += dy / l * speed
	}
	s.x = clamp(s.x, 15, cravW-15)
	s.y = clamp(s.y, 40, cravH-12)

	for i, c := range s.checkpoints {
		if i != s.cp && math.Hypot(c[0]-s.x, c[1]-s.y) < 30 {
			s.cp = i
			s.say("cp", "Un angolo d'ombra profonda: se vieni chiamata, ripartirai da qui.")
		}
	}
}

func (s *Painting2Scene) updateLampAndLight() {
	toggle := inpututil.IsKeyJustPressed(ebiten.KeyL) || inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyF) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if x, y, ok := pointerJustPressed(); ok && s.onLampButton(x, y) {
		toggle = true
	}
	if toggle {
		s.lampOn = !s.lampOn
	}

	if s.rayState() == 1 {
		s.say("ray", "La mano di Cristo si illumina: sta per indicare. Esci dalla sua altezza!")
	}

	if s.graceT > 0 {
		s.graceT--
	} else if s.lit(s.x, s.y) {
		rate := 0.022
		if s.lampOn {
			rate = 0.04
		}
		s.exposure += rate
	} else {
		s.exposure = math.Max(0, s.exposure-0.015)
	}
	if s.exposure >= 1 {
		s.lives--
		s.exposure = 0
		s.caughtT = 70
		s.graceT = 120
		s.shake = 8
		s.el.Hit()
		s.hasTarget = false
		PlaySFX("called")
		cp := s.checkpoints[s.cp]
		s.x, s.y = cp[0], cp[1]
		if s.lives <= 0 {
			s.phase = phLost
		} else {
			s.say("", "Sei stata chiamata. Torni nell'ombra.")
		}
	}

	lx, ly := s.el.LampPos()
	lr := 64.0
	for _, f := range s.frags {
		vis := 0.0
		if s.lampOn {
			d := math.Hypot(f.x-lx, f.y-ly)
			vis = clamp(1-(d-lr*0.5)/(lr*0.6), 0, 1)
		}
		f.seen = math.Max(f.seen*0.985, vis)
	}
}

func (s *Painting2Scene) checkFragments() {
	for _, f := range s.frags {
		if f.taken || math.Hypot(f.x-s.x, f.y-(s.y-10)) > 21 {
			continue
		}
		f.taken = true
		s.taken++
		PlaySFX("chime")
		var lines []Line
		if f.kind == pZinco {
			lines = []Line{{"Ada", zincoClue, true}}
		} else {
			lines = []Line{{"Ada", ch2Clues()[s.clueIdx], true}}
			if s.clueIdx == 0 && story.ch2Choice == 1 {
				lines = append(lines, Line{"Elena", "Troppo tardi, Ada. Gliel'ho già detto.", false})
			}
			s.clueIdx++
		}
		if s.taken == len(s.frags) {
			s.phase = phReveal
			PlaySFX("reveal")
			lines = append(lines, Line{"", "La luce si ferma. Le figure trattengono il respiro. Sul tavolo, la pittura comincia a diventare trasparente...", false})
		}
		s.dlg.Start(lines...)
		return
	}
}

func (s *Painting2Scene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	cx, cy := s.camX, s.camY
	s.off.Clear()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-cx, -cy)
	s.off.DrawImage(s.bg, op)

	frozen := s.phase == phReveal || s.phase == phEnd
	// fascio di luce dinamico
	bp := beamPolygon(s.sweep())
	var beam vector.Path
	beam.MoveTo(float32(bp[0][0]-cx), float32(bp[0][1]-cy))
	for _, p := range bp[1:] {
		beam.LineTo(float32(p[0]-cx), float32(p[1]-cy))
	}
	beam.Close()
	fillPath(s.off, &beam, rgba(255, 220, 160, 0.13))

	sx := (s.rng.Float64()*2 - 1) * s.shake
	sy := (s.rng.Float64()*2 - 1) * s.shake
	lx, ly := s.el.LampPos()
	lr := 0.0
	if s.lampOn {
		lr = 64 + 4*math.Sin(ft/20)
	}
	applyGradeLamp(screen, s.off, s.restore, ft, sx, sy, lx-cx, ly-cy, lr)
	ox, oy := -cx+sx, -cy+sy

	if !frozen {
		s.drawGazes(screen, ox, oy)
		s.drawRay(screen, ox, oy, ft)
	}
	for _, c := range s.checkpoints {
		vector.StrokeCircle(screen, float32(c[0]+ox), float32(c[1]+oy-6), 16, 1, rgba(150, 130, 200, 0.18+0.08*math.Sin(ft/30)), true)
	}

	s.el.DrawTrail(screen, ox, oy)
	if s.lampOn {
		s.el.DrawLight(screen, ft, ox, oy, 1)
	}

	// pigmenti, visibili solo sotto la lampada
	for _, f := range s.frags {
		if f.taken {
			continue
		}
		fx, fy := f.x+ox, f.y+oy
		if f.seen < 0.02 {
			if math.Hypot(f.x-s.x, f.y-s.y) < 200 && (s.t+int(f.x))%150 < 12 {
				vector.DrawFilledCircle(screen, float32(fx), float32(fy), 2, rgba(255, 255, 255, 0.35), true)
			}
			continue
		}
		p := 0.5 + 0.5*math.Sin(ft/12+f.x)
		for k := 4; k >= 1; k-- {
			vector.DrawFilledCircle(screen, float32(fx), float32(fy), float32(4+float64(k)*3+p*3), rgba(f.r, f.g, f.b, 0.14*f.seen), true)
		}
		vector.DrawFilledCircle(screen, float32(fx), float32(fy), 5, rgba(f.r, f.g, f.b, f.seen), true)
		vector.StrokeCircle(screen, float32(fx), float32(fy), 6, 1, rgba(255, 245, 220, 0.6*f.seen), true)
	}

	if s.revealT > 0 {
		s.drawPentimento(screen, ox, oy)
	}
	s.el.Draw(screen, ft, ox, oy)

	if s.caughtT > 0 {
		a := float64(s.caughtT) / 70
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(255, 235, 200, a*0.5), false)
		drawText(screen, "Chiamata.", faceTitle, ScreenW/2, 140, rgba(60, 30, 20, a), 0)
	}

	if s.phase == phPlay || s.phase == phEnter {
		s.drawHUD(screen, ft)
	}
	switch {
	case s.phase == phEnd && !s.dlg.Active():
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.72), false)
		drawText(screen, "PENTIMENTO", faceTitle, ScreenW/2, 90, rgba(214, 178, 104, 1), 0)
		drawText(screen, "Capitolo 2 — La chiamata", faceItalic, ScreenW/2, 165, rgba(230, 222, 205, 1), 0)
		drawText(screen, "continua a Ginevra...", faceText, ScreenW/2, 205, rgba(200, 190, 170, 1), 0)
		if (s.t/30)%2 == 0 {
			drawText(screen, "Clicca per tornare al titolo", faceSmall, ScreenW/2, 300, rgba(200, 190, 170, 0.8), 0)
		}
	case s.phase == phLost:
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(255, 240, 210, 0.85), false)
		drawText(screen, "Ti hanno scelta.", faceTitle, ScreenW/2, 110, rgba(70, 40, 25, 0.95), 0)
		drawText(screen, "La luce ti ha trovata tre volte. La tela ti respinge fuori.", faceItalic, ScreenW/2, 190, rgba(80, 50, 35, 0.95), 0)
		if (s.t/30)%2 == 0 {
			drawText(screen, "Clicca per tornare nell'ombra", faceText, ScreenW/2, 250, rgba(80, 50, 35, 0.95), 0)
		}
	}
	s.dlg.Draw(screen)
}

func (s *Painting2Scene) drawGazes(dst *ebiten.Image, ox, oy float64) {
	for _, g := range s.gazes {
		a := g.angle(s.t)
		h := g.half * math.Pi / 180
		var p vector.Path
		p.MoveTo(float32(g.x+ox), float32(g.y+oy))
		for i := 0; i <= 8; i++ {
			aa := a - h + 2*h*float64(i)/8
			p.LineTo(float32(g.x+ox+math.Cos(aa)*g.reach), float32(g.y+oy+math.Sin(aa)*g.reach))
		}
		p.Close()
		c := rgba(255, 230, 180, 0.07)
		if g.sees(s.t, s.x, s.y-14) {
			c = rgba(255, 120, 90, 0.16)
		}
		fillPath(dst, &p, c)
		for _, e := range []float64{-h, h} {
			vector.StrokeLine(dst, float32(g.x+ox), float32(g.y+oy), float32(g.x+ox+math.Cos(a+e)*g.reach), float32(g.y+oy+math.Sin(a+e)*g.reach), 1, rgba(255, 230, 180, 0.18), true)
		}
		// occhi che brillano
		vector.DrawFilledCircle(dst, float32(g.x+ox+math.Cos(a)*5), float32(g.y+oy+math.Sin(a)*3-2), 1.8, rgba(255, 245, 220, 0.9), true)
	}
}

func (s *Painting2Scene) drawRay(dst *ebiten.Image, ox, oy, ft float64) {
	hx, hy := cravHand[0]+ox, cravHand[1]+oy
	switch s.rayState() {
	case 1:
		p := float64(s.t%rayPeriod-rayIdle) / rayWarn
		vector.DrawFilledCircle(dst, float32(hx), float32(hy), float32(6+10*p), rgba(255, 230, 170, 0.25+0.4*p*math.Abs(math.Sin(ft/4))), true)
		vector.StrokeLine(dst, float32(hx), float32(hy-24), 0, float32(hy-24), 1, rgba(255, 230, 170, 0.15*p), false)
		vector.StrokeLine(dst, float32(hx), float32(hy+24), 0, float32(hy+24), 1, rgba(255, 230, 170, 0.15*p), false)
	case 2:
		vector.DrawFilledRect(dst, 0, float32(hy-24), float32(hx), 48, rgba(255, 238, 200, 0.32), false)
		vector.DrawFilledCircle(dst, float32(hx), float32(hy), 14, rgba(255, 245, 220, 0.7), true)
	}
}

func (s *Painting2Scene) drawPentimento(dst *ebiten.Image, ox, oy float64) {
	a := s.revealT
	c := rgba(250, 240, 210, a*0.85)
	f := cravFigures[0]
	hx, hy := f.x+ox, f.headY+oy
	// il volto nascosto: un uomo maturo, tempie grigie
	vector.DrawFilledCircle(dst, float32(hx), float32(hy-4), 22, rgba(250, 240, 210, a*0.22), true)
	vector.StrokeCircle(dst, float32(hx), float32(hy-4), 22, 2, c, true)
	vector.StrokeLine(dst, float32(hx-10), float32(hy-8), float32(hx-3), float32(hy-8), 1.5, c, true)
	vector.StrokeLine(dst, float32(hx+3), float32(hy-8), float32(hx+10), float32(hy-8), 1.5, c, true)
	vector.StrokeLine(dst, float32(hx), float32(hy-6), float32(hx-2), float32(hy+4), 1.2, c, true)
	vector.StrokeLine(dst, float32(hx-6), float32(hy+10), float32(hx+6), float32(hy+10), 1.2, c, true)
	vector.StrokeLine(dst, float32(hx-20), float32(hy-14), float32(hx-14), float32(hy-22), 2, c, true)
	vector.StrokeLine(dst, float32(hx+20), float32(hy-14), float32(hx+14), float32(hy-22), 2, c, true)
	// anello
	vector.StrokeCircle(dst, float32(hx+30), float32(hy+50), 6, 1.5, rgba(255, 220, 120, a), true)
	drawText(dst, "C.V.", faceBold, hx+30, hy+60, rgba(255, 225, 140, a), 0)
	drawText(dst, "copia n. 4", faceTitle, 360+ox, 255+oy, rgba(250, 240, 210, a*0.3), 0)
}

func (s *Painting2Scene) drawHUD(dst *ebiten.Image, ft float64) {
	for i, f := range s.frags {
		x := float32(20 + i*18)
		if f.taken {
			vector.DrawFilledCircle(dst, x, 18, 6, rgba(f.r, f.g, f.b, 1), true)
		}
		vector.StrokeCircle(dst, x, 18, 6, 1, rgba(230, 220, 200, 0.8), true)
	}
	for i := 0; i < maxLives; i++ {
		x := float32(20 + i*16)
		c := rgba(184, 72, 46, 1)
		if i >= s.lives {
			c = rgba(60, 50, 50, 0.6)
		}
		vector.DrawFilledCircle(dst, x, 42, 4.5, c, true)
		var p vector.Path
		p.MoveTo(x-4, 41)
		p.LineTo(x, 33)
		p.LineTo(x+4, 41)
		p.Close()
		fillPath(dst, &p, c)
	}
	// occhio: quanto sei esposta
	ex, ey := float32(ScreenW-40), float32(24)
	open := float32(2 + 8*s.exposure)
	var eye vector.Path
	eye.MoveTo(ex-16, ey)
	eye.QuadTo(ex, ey-open-4, ex+16, ey)
	eye.QuadTo(ex, ey+open+4, ex-16, ey)
	eye.Close()
	ec := lerpColor(rgba(200, 190, 170, 0.8), rgba(255, 90, 60, 1), s.exposure)
	fillPath(dst, &eye, rgba(20, 14, 10, 0.8))
	vector.StrokeLine(dst, ex-16, ey, ex+16, ey, 0.5, ec, true)
	vector.DrawFilledCircle(dst, ex, ey, open*0.6, ec, true)
	if s.lit(s.x, s.y) {
		drawText(dst, "SEI IN LUCE", faceBold, float64(ex)-26, 14, rgba(255, 120, 90, 0.6+0.4*math.Sin(ft/4)), 1)
	}

	// pulsante lampada
	a := 0.45
	if s.lampOn {
		a = 0.85
	}
	vector.DrawFilledCircle(dst, flashBX, flashBY, flashBR, rgba(60, 30, 110, a*0.6), true)
	vector.StrokeCircle(dst, flashBX, flashBY, flashBR, 2, rgba(210, 180, 255, a), true)
	label := "UV off"
	if s.lampOn {
		label = "UV on"
	}
	drawText(dst, label, faceSmall, flashBX, flashBY-7, rgba(240, 230, 255, a+0.15), 0)

	if s.hintT > 0 && !s.dlg.Active() {
		al := math.Min(1, float64(s.hintT)/30)
		lines := wrap(s.hint, faceText, ScreenW-120)
		h := float32(14 + 19*len(lines))
		vector.DrawFilledRect(dst, 50, ScreenH-28-h, ScreenW-100, h, rgba(10, 8, 14, 0.7*al), false)
		for i, l := range lines {
			drawText(dst, l, faceText, ScreenW/2, float64(ScreenH-28-h+7+float32(i)*19), rgba(236, 228, 214, al), 0)
		}
	}
}
