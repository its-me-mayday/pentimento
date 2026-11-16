package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Capitolo 1 — dentro "La notte stellata".
// La lampada UV è la meccanica centrale: rivela i pigmenti, ridà colore e respinge
// la Macchia, ma si scarica. Lanterne e stelle la ricaricano. Ogni pigmento cambia il mondo:
// il blu fa cavalcare i vortici (si passa il muro e si cammina sul cielo), il giallo dà il lampo UV.

const (
	groundTop  = 345.0 // sopra questa linea comincia il cielo
	wallX0     = 760.0 // muro di vortice tra le due metà del villaggio
	wallX1     = 850.0
	maxLives   = 3
	flashCost  = 0.3
	drainRate  = 1.0 / 3000 // ~50 s di carica piena
	chargeRate = 0.006
)

type pigment int

const (
	pBlu pigment = iota
	pGiallo
	pTitanio
	pVerde
	pNero
)

type fragment struct {
	x, y    float64
	kind    pigment
	name    string
	r, g, b uint8
	taken   bool
	seen    float64 // quanto è visibile (0 = nascosto nel grigio)
}

type lantern struct {
	x, y float64
	sky  bool // le stelle funzionano da lanterne solo per chi sa camminare sul cielo
}

type particle struct {
	x, y, px, py float64
	age          int
}

type stain struct {
	x, y   float64
	active bool
	stun   int
	wx, wy float64 // punto verso cui vaga
	wt     int
}

type paintPhase int

const (
	phEnter paintPhase = iota
	phPlay
	phReveal
	phEnd
	phLost
)

var worldBG, worldMini *ebiten.Image

type PaintingScene struct {
	t     int
	phase paintPhase
	dlg   Dialogue
	off   *ebiten.Image
	rng   *rand.Rand
	parts []particle

	el        *Elena
	x, y      float64
	tx, ty    float64
	hasTarget bool
	camX      float64
	camY      float64

	frags    []*fragment
	taken    int
	clueIdx  int
	lanterns []lantern
	hasBlu   bool
	hasFlash bool

	battery  float64
	charging bool
	flashT   int
	lives    int
	inv      int

	mac       stain
	stainFade float64

	restore float64
	shake   float64
	revealT float64

	hint     string
	hintT    int
	hintSeen map[string]bool
}

var orderedClues = []string{
	"Elena. Se senti la mia voce, la lampada funziona. Dentro la tela si vede solo ciò che il falsario ha coperto: fatti luce.",
	"Questo quadro è arrivato in laboratorio di notte, senza documenti. Mi hanno pagata in contanti per «ripulirlo» in una settimana.",
	"Non sono l'unica ad averlo capito. Da settimane una macchina grigia resta parcheggiata sotto il laboratorio.",
	"Se ti diranno che è stato un incidente, non crederci. Guarda sotto il cielo. Ho lasciato lì il resto.",
}

const titanioClue = "Guarda il bianco delle stelle: è bianco di titanio. Nel 1889 non era ancora in commercio. Questa tela non è un originale."

func NewPaintingScene(retry bool) *PaintingScene {
	if worldBG == nil {
		worldBG = WorldImage()
		small := ebiten.NewImage(160, 45)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(160/worldCanvas.W, 45/worldCanvas.H)
		op.Filter = ebiten.FilterLinear
		small.DrawImage(worldBG, op)
		worldMini = ebiten.NewImage(160, 45)
		applyGrade(worldMini, small, 0, 0, 0, 0) // minimappa in seppia, come il quadro
	}
	s := &PaintingScene{
		off:      ebiten.NewImage(ScreenW, ScreenH),
		rng:      rand.New(rand.NewSource(7)),
		x:        170,
		y:        440,
		battery:  1,
		lives:    maxLives,
		hintSeen: map[string]bool{},
	}
	s.el = NewElena(s.x, s.y)
	s.camX = 0
	s.camY = clamp(s.y-ScreenH*0.42, 0, worldCanvas.H-ScreenH)

	defs := []struct {
		kind    pigment
		name    string
		r, g, b uint8
		x, y    float64
	}{
		{pBlu, "Blu oltremare", 60, 100, 240, 395, 470},
		{pGiallo, "Giallo di cromo", 245, 200, 40, 1120, 445},
		{pTitanio, "Bianco di titanio", 250, 250, 245, 1338, 218},
		{pVerde, "Verde smeraldo", 40, 190, 120, 300, 70},
		{pNero, "Nero d'avorio", 95, 85, 80, 1860, 500},
	}
	for _, d := range defs {
		s.frags = append(s.frags, &fragment{x: d.x, y: d.y, kind: d.kind, name: d.name, r: d.r, g: d.g, b: d.b})
	}
	for _, p := range [][2]float64{{130, 455}, {600, 505}, {985, 470}, {1470, 485}, {1735, 445}} {
		s.lanterns = append(s.lanterns, lantern{x: p[0], y: p[1]})
	}
	for _, st := range worldCanvas.Stars {
		s.lanterns = append(s.lanterns, lantern{x: st.x, y: st.y, sky: true})
	}
	for i := 0; i < 1100; i++ {
		s.parts = append(s.parts, s.newParticle())
	}

	if retry {
		s.dlg.Start(Line{"", "La tela si riapre. I pigmenti sono tornati dove li avevi trovati.", false})
	} else {
		s.dlg.Start(
			Line{"", "Odore di olio di lino. Vento caldo. Il cielo si muove, pennellata dopo pennellata.", false},
			Line{"Elena", "Sono... dentro il quadro? I colori sono spenti. Come se qualcuno li avesse rubati.", false},
			Line{"", "I pigmenti perduti sono nascosti nel grigio: si vedono solo sotto la luce della lampada di Ada.", false},
			Line{"", "La lampada si scarica: ricaricala alle lanterne. Muoviti con WASD / frecce, o tieni premuto dove vuoi andare.", false},
		)
	}
	return s
}

func (s *PaintingScene) newParticle() particle {
	x, y := s.rng.Float64()*worldCanvas.W, s.rng.Float64()*worldCanvas.Horizon
	return particle{x, y, x, y, s.rng.Intn(120)}
}

func (s *PaintingScene) lampRadius() float64 {
	r := 26 + 46*s.battery
	if s.el.hurt > 0 {
		r *= 0.5
	}
	return r
}

func (s *PaintingScene) say(key, msg string) {
	if key != "" {
		if s.hintSeen[key] {
			return
		}
		s.hintSeen[key] = true
	}
	s.hint, s.hintT = msg, 240
}

func (s *PaintingScene) Update(g *Game) error {
	s.t++
	ft := float64(s.t)
	target := float64(s.taken) / float64(len(s.frags))
	s.restore += (target - s.restore) * 0.03
	s.shake *= 0.9
	if s.hintT > 0 {
		s.hintT--
	}
	if s.flashT > 0 {
		s.flashT--
	}
	s.updateMood()
	for i := range s.parts {
		p := &s.parts[i]
		p.px, p.py = p.x, p.y
		dx, dy := worldCanvas.Flow(p.x, p.y, ft/60)
		p.x += dx * 0.9
		p.y += dy * 0.9
		p.age++
		if p.age > 160 || p.x < 0 || p.x > worldCanvas.W || p.y < 0 || p.y > worldCanvas.Horizon {
			*p = s.newParticle()
		}
	}
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
					s.x, s.y = f.x+30, f.y+10
					s.hasBlu = s.hasBlu || f.kind != pBlu
					break
				}
			}
		}
		s.updatePlayer()
		s.updateLamp()
		s.updateStain()
		s.checkFragments()
	case phReveal:
		s.stainFade = math.Min(1, s.stainFade+0.02)
		s.revealT = math.Min(1, s.revealT+0.006)
		if s.revealT >= 1 {
			s.phase = phEnd
			s.dlg.Start(
				Line{"Elena", "«Copia n. 3»... e una firma sola: «C.». Non è di Ada.", false},
				Line{"Ada", "Esistono almeno tre copie di questo quadro, Elena. L'originale è uscito da un museo per un «prestito» e non è mai tornato.", true},
				Line{"Ada", "Chi le dipinge si fa chiamare il Collegio. Restaurano, sostituiscono, vendono. E nessuno se ne accorge, perché i certificati li firmiamo noi.", true},
				Line{"Ada", "Il prossimo quadro passerà dal laboratorio fra sette giorni. Sarà lì che li troverai. Perdonami se ti lascio questo peso.", true},
				Line{"Elena", "Il Collegio... Va bene, Ada. Vediamo chi ha dipinto sopra la tua morte.", false},
			)
		}
	case phEnd:
		if !s.dlg.Active() && confirmPressed() {
			g.GoTo(NewTitleScene())
		}
	case phLost:
		if confirmPressed() {
			g.GoTo(NewPaintingScene(true))
		}
	}
	return nil
}

func (s *PaintingScene) updateCamera() {
	tx := clamp(s.x-ScreenW/2, 0, worldCanvas.W-ScreenW)
	ty := clamp(s.y-ScreenH*0.42, 0, worldCanvas.H-ScreenH)
	s.camX += (tx - s.camX) * 0.1
	s.camY += (ty - s.camY) * 0.1
}

func (s *PaintingScene) palette() []color.NRGBA {
	var p []color.NRGBA
	for _, f := range s.frags {
		if f.taken {
			p = append(p, color.NRGBA{f.r, f.g, f.b, 255})
		}
	}
	return p
}

func (s *PaintingScene) updateMood() {
	m := Mood{Pad: 1, Arp: 0.25 + 0.75*s.restore, Bright: s.restore}
	if s.mac.active && s.phase == phPlay {
		d := math.Hypot(s.x-s.mac.x, s.y-s.mac.y)
		m.Tension = clamp(1-d/320, 0.12, 1)
		if s.mac.stun > 0 {
			m.Tension = 0.1
		}
	}
	switch s.phase {
	case phEnd:
		m = Mood{Pad: 0.9, Arp: 0.7, Bright: 1}
	case phLost:
		m = Mood{Pad: 0.6, Tension: 0.3}
	}
	SetMood(m)
}

// flashButton: area del pulsante LAMPO per il touch.
const flashBX, flashBY, flashBR = ScreenW - 48, ScreenH - 48, 30

func (s *PaintingScene) onFlashButton(x, y int) bool {
	return s.hasFlash && math.Hypot(float64(x)-flashBX, float64(y)-flashBY) < flashBR+6
}

func (s *PaintingScene) updatePlayer() {
	ox, oy := s.x, s.y
	defer func() {
		s.el.Step(s.x-ox, s.y-oy, s.palette())
		s.el.X, s.el.Y = s.x, s.y
		if s.inv > 0 {
			s.inv--
		}
	}()
	const speed = 1.8
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
	if x, y, ok := pointer(); ok && !s.onFlashButton(x, y) &&
		(ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.AppendTouchIDs(nil)) > 0) {
		s.tx, s.ty, s.hasTarget = float64(x)+s.camX, float64(y)+s.camY, true
	}
	if s.hasTarget {
		vx, vy := s.tx-s.x, s.ty-s.y
		d := math.Hypot(vx, vy)
		if d < 2 {
			s.hasTarget = false
		} else {
			dx, dy = vx/d, vy/d
		}
	}
	if l := math.Hypot(dx, dy); l > 0 {
		s.x += dx / l * speed
		s.y += dy / l * speed
	}

	// correnti: nel cielo i vortici trascinano Elena
	if s.y < groundTop && s.hasBlu {
		fx, fy := worldCanvas.Flow(s.x, s.y, float64(s.t)/60)
		s.x += fx * 0.6
		s.y += fy * 0.6
	}
	// il muro di vortice
	if s.y >= groundTop-10 && s.x > wallX0 && s.x < wallX1 {
		if !s.hasBlu {
			s.x -= 3.0
			s.shake = math.Max(s.shake, 1.5)
			s.say("wall", "Il vortice ti respinge. Ti serve un colore capace di scorrere con lui...")
		} else {
			s.x += 0.6
		}
	}
	// il cielo è irraggiungibile senza il blu
	if !s.hasBlu && s.y < groundTop {
		s.y = groundTop
		if dy < 0 {
			s.say("sky", "Il cielo è troppo lontano... per ora.")
		}
	}
	s.x = clamp(s.x, 12, worldCanvas.W-12)
	s.y = clamp(s.y, 30, worldCanvas.H-10)
}

func (s *PaintingScene) updateLamp() {
	// ricarica vicino a lanterne e stelle
	s.charging = false
	lx, ly := s.el.LampPos()
	for _, l := range s.lanterns {
		if l.sky && !s.hasBlu {
			continue
		}
		if math.Hypot(l.x-s.x, l.y-(s.y-12)) < 38 {
			s.charging = true
			break
		}
	}
	if s.charging {
		s.battery = math.Min(1, s.battery+chargeRate)
	} else {
		s.battery = math.Max(0, s.battery-drainRate)
	}
	if s.battery < 0.15 && !s.charging {
		s.say("low", "La lampada si sta spegnendo: torna a una lanterna!")
	}
	if s.battery > 0.5 {
		delete(s.hintSeen, "low")
	}

	// lampo UV
	want := inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyF) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if x, y, ok := pointerJustPressed(); ok && s.onFlashButton(x, y) {
		want = true
	}
	if want && s.hasFlash {
		if s.battery < flashCost {
			s.say("", "Non c'è abbastanza carica per un lampo.")
		} else {
			s.battery -= flashCost
			s.flashT = 30
			PlaySFX("flash")
			if s.mac.active && math.Hypot(s.mac.x-lx, s.mac.y-ly) < 170 {
				s.mac.stun = 200
				d := math.Max(1, math.Hypot(s.mac.x-s.x, s.mac.y-s.y))
				s.mac.x += (s.mac.x - s.x) / d * 90
				s.mac.y += (s.mac.y - s.y) / d * 90
			}
		}
	}

	// i pigmenti si rivelano sotto la luce (o con il lampo)
	lr := s.lampRadius()
	for _, f := range s.frags {
		d := math.Hypot(f.x-lx, f.y-ly)
		vis := clamp(1-(d-lr*0.5)/(lr*0.6), 0, 1)
		if s.flashT > 0 && d < 260 {
			vis = 1
		}
		f.seen = math.Max(f.seen*0.985, vis)
	}
}

func (s *PaintingScene) updateStain() {
	m := &s.mac
	if !m.active {
		return
	}
	if m.stun > 0 {
		m.stun--
		return
	}
	lx, ly := s.el.LampPos()
	d := math.Hypot(s.x-m.x, s.y-m.y)
	speed := 0.7 + 0.1*float64(s.taken)
	tx, ty := s.x, s.y
	if d > 320 {
		// vaga, ma sempre più o meno verso Elena
		m.wt--
		if m.wt <= 0 {
			m.wx = s.x + (s.rng.Float64()*2-1)*300
			m.wy = clamp(s.y+(s.rng.Float64()*2-1)*150, 40, worldCanvas.H-20)
			m.wt = 180
		}
		tx, ty, speed = m.wx, m.wy, 0.45
	}
	if math.Hypot(m.x-lx, m.y-ly) < s.lampRadius() {
		speed *= 0.45 // la luce UV la rallenta
	}
	vx, vy := tx-m.x, ty-m.y
	if l := math.Hypot(vx, vy); l > 0.5 {
		wob := math.Sin(float64(s.t)/30) * 0.5
		m.x += (vx/l - vy/l*wob) * speed
		m.y += (vy/l + vx/l*wob) * speed
	}
	// le lanterne sono zone sicure
	for _, l := range s.lanterns {
		if dl := math.Hypot(m.x-l.x, m.y-l.y); dl < 70 && dl > 0 {
			m.x += (m.x - l.x) / dl * 2
			m.y += (m.y - l.y) / dl * 2
		}
	}
	if d < 20 && s.inv == 0 {
		s.lives--
		s.inv = 100
		s.el.Hit()
		PlaySFX("hit")
		s.shake = 10
		s.hasTarget = false
		if d > 0 {
			s.x = clamp(s.x+(s.x-m.x)/d*50, 12, worldCanvas.W-12)
			s.y = clamp(s.y+(s.y-m.y)/d*50, 30, worldCanvas.H-10)
		}
		s.restore = math.Max(0, s.restore-0.15)
		if s.lives <= 0 {
			s.phase = phLost
			PlaySFX("hit")
		} else if s.lives == 1 {
			s.say("", "Un'altra macchia e la tela si richiude!")
		}
	}
}

func (s *PaintingScene) checkFragments() {
	for _, f := range s.frags {
		if f.taken || math.Hypot(f.x-s.x, f.y-(s.y-10)) > 21 {
			continue
		}
		f.taken = true
		s.taken++
		PlaySFX("chime")
		clue := titanioClue
		if f.kind != pTitanio {
			clue = orderedClues[s.clueIdx]
			s.clueIdx++
		}
		lines := []Line{{"Ada", clue, true}}
		switch f.kind {
		case pBlu:
			s.hasBlu = true
			s.mac = stain{x: wallX0 - 40, y: 420, active: true}
			lines = append(lines,
				Line{"", "Il blu oltremare scorre nella lampada. Ora i vortici ti portano invece di respingerti: puoi attraversare il muro a est e camminare sul cielo.", false},
				Line{"", "Ma qualcosa si muove tra le pennellate. Una macchia d'inchiostro, densa, affamata. La luce UV la rallenta; vicino alle lanterne non osa avvicinarsi.", false},
			)
		case pGiallo:
			s.hasFlash = true
			lines = append(lines, Line{"", "Il giallo di cromo accende la lampada. LAMPO UV: premi SPAZIO (o tocca il pulsante) per stordire la Macchia. Costa un terzo della carica.", false})
		}
		if s.taken == len(s.frags) {
			s.phase = phReveal
			PlaySFX("reveal")
			lines = append(lines, Line{"", "La Macchia si scioglie. Sotto il cielo, la pittura si fa trasparente...", false})
		}
		s.dlg.Start(lines...)
		return
	}
}

func (s *PaintingScene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	cx, cy := s.camX, s.camY
	s.off.Clear()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-cx, -cy)
	s.off.DrawImage(worldBG, op)
	for _, p := range s.parts {
		if p.x < cx-10 || p.x > cx+ScreenW+10 || p.y < cy-10 || p.y > cy+ScreenH+10 {
			continue
		}
		a := 0.55 * math.Sin(math.Pi*float64(p.age)/160)
		x, y := p.x-cx, p.y-cy
		vector.StrokeLine(s.off, float32(p.px-cx), float32(p.py-cy), float32(x+(p.x-p.px)*3), float32(y+(p.y-p.py)*3), 1.6, rgba(220, 230, 255, a), true)
	}
	s.drawStain(s.off, cx, cy, ft)

	sx := (s.rng.Float64()*2 - 1) * s.shake
	sy := (s.rng.Float64()*2 - 1) * s.shake
	lx, ly := s.el.LampPos()
	lr := s.lampRadius() * (1 + 0.05*math.Sin(ft/20))
	if s.flashT > 0 {
		lr += float64(s.flashT) * 8
	}
	applyGradeLamp(screen, s.off, s.restore, ft, sx, sy, lx-cx, ly-cy, lr)
	ox, oy := -cx+sx, -cy+sy

	s.drawWall(screen, ox, oy, ft)
	s.drawLanterns(screen, ox, oy, ft)
	s.el.DrawTrail(screen, ox, oy)
	s.el.DrawLight(screen, ft, ox, oy, s.battery)

	// pentimento rivelato
	if s.revealT > 0 {
		a := s.revealT
		drawText(screen, "copia n. 3", faceTitle, ScreenW/2, 115, rgba(250, 240, 210, a*0.35), 0)
		drawText(screen, "— C.", faceItalic, ScreenW/2+150, 185, rgba(250, 240, 210, a*0.6), 0)
	}

	// pigmenti: visibili solo sotto la luce
	for _, f := range s.frags {
		if f.taken {
			continue
		}
		fx, fy := f.x+ox, f.y+oy
		if f.seen < 0.02 {
			// un luccichio ogni tanto, se sei vicina
			if math.Hypot(f.x-s.x, f.y-s.y) < 230 && (s.t+int(f.x))%150 < 12 {
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

	s.el.Draw(screen, ft, ox, oy)

	if s.flashT > 0 {
		a := float64(s.flashT) / 30
		ex, ey := float32(lx+ox), float32(ly+oy)
		vector.StrokeCircle(screen, ex, ey, float32((30-float64(s.flashT))*7), 3, rgba(220, 190, 255, a), true)
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(200, 170, 255, a*0.25), false)
	}

	if s.phase == phPlay || s.phase == phEnter {
		s.drawHUD(screen, ft)
	}

	switch {
	case s.phase == phEnd && !s.dlg.Active():
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.7), false)
		drawText(screen, "PENTIMENTO", faceTitle, ScreenW/2, 90, rgba(214, 178, 104, 1), 0)
		drawText(screen, "Capitolo 1 — La notte stellata", faceItalic, ScreenW/2, 165, rgba(230, 222, 205, 1), 0)
		drawText(screen, "continua...", faceText, ScreenW/2, 205, rgba(200, 190, 170, 1), 0)
		if (s.t/30)%2 == 0 {
			drawText(screen, "Clicca per tornare al titolo", faceSmall, ScreenW/2, 300, rgba(200, 190, 170, 0.8), 0)
		}
	case s.phase == phLost:
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(4, 4, 10, 0.8), false)
		drawText(screen, "La tela si richiude...", faceTitle, ScreenW/2, 110, rgba(200, 190, 230, 0.9), 0)
		drawText(screen, "L'inchiostro ha coperto la lampada. Ada aspetta ancora sotto il cielo.", faceItalic, ScreenW/2, 190, rgba(200, 190, 210, 0.9), 0)
		if (s.t/30)%2 == 0 {
			drawText(screen, "Clicca per riaprire il quadro", faceText, ScreenW/2, 250, rgba(230, 220, 240, 0.9), 0)
		}
	}

	s.dlg.Draw(screen)
}

func (s *PaintingScene) drawStain(dst *ebiten.Image, cx, cy, ft float64) {
	if !s.mac.active {
		return
	}
	a := 0.9 * (1 - s.stainFade)
	size := 1.0
	if s.mac.stun > 0 {
		size = 0.6 + 0.1*math.Sin(ft/2)
		a *= 0.7
	}
	mx, my := s.mac.x-cx, s.mac.y-cy
	for i := 0; i < 8; i++ {
		ang := float64(i)/8*2*math.Pi + ft/40
		r := (10 + 5*math.Sin(ft/13+float64(i)*1.7)) * size
		vector.DrawFilledCircle(dst, float32(mx+math.Cos(ang)*r), float32(my+math.Sin(ang)*r), float32((11+3*math.Sin(ft/9+float64(i)))*size), rgba(6, 6, 12, a), true)
	}
	vector.DrawFilledCircle(dst, float32(mx), float32(my), float32(15*size), rgba(0, 0, 0, a), true)
	// gocce che colano
	for i := 0; i < 3; i++ {
		ph := math.Mod(ft/50+float64(i)*0.33, 1)
		vector.DrawFilledCircle(dst, float32(mx-8+float64(i)*8), float32(my+14+ph*12), float32(3*(1-ph)*size), rgba(0, 0, 0, a*(1-ph)), true)
	}
}

func (s *PaintingScene) drawWall(dst *ebiten.Image, ox, oy, ft float64) {
	top := groundTop - 20
	for i := 0; i < 70; i++ {
		fi := float64(i)
		y := top + math.Mod(fi*37+ft*1.5, worldCanvas.H-top)
		x := wallX0 + 15 + math.Mod(fi*53, wallX1-wallX0-30) + math.Sin(ft/15+fi)*12
		var c color.NRGBA
		var dx, dy float64
		if s.hasBlu {
			c = rgba(90, 140, 255, 0.25)
			dx, dy = 6, math.Sin(fi)*2
		} else {
			c = rgba(230, 220, 255, 0.45)
			ang := ft/8 + fi
			dx, dy = math.Cos(ang)*7, math.Sin(ang)*7
		}
		vector.StrokeLine(dst, float32(x+ox-dx), float32(y+oy-dy), float32(x+ox+dx), float32(y+oy+dy), 2, c, true)
	}
}

func (s *PaintingScene) drawLanterns(dst *ebiten.Image, ox, oy, ft float64) {
	for _, l := range s.lanterns {
		x, y := l.x+ox, l.y+oy
		if x < -40 || x > ScreenW+40 || y < -40 || y > ScreenH+40 {
			continue
		}
		pulse := 0.7 + 0.3*math.Sin(ft/17+l.x)
		if l.sky {
			if !s.hasBlu {
				continue
			}
			vector.StrokeCircle(dst, float32(x), float32(y), float32(20+4*pulse), 1.5, rgba(255, 220, 120, 0.35*pulse), true)
			continue
		}
		vector.StrokeLine(dst, float32(x), float32(y+4), float32(x), float32(y+26), 2, rgba(30, 26, 24, 1), true)
		for k := 5; k >= 1; k-- {
			vector.DrawFilledCircle(dst, float32(x), float32(y), float32(k)*7*float32(pulse), rgba(255, 190, 90, 0.05), true)
		}
		vector.DrawFilledRect(dst, float32(x-3), float32(y-4), 6, 8, rgba(255, 210, 120, 1), true)
		vector.StrokeRect(dst, float32(x-3), float32(y-4), 6, 8, 1, rgba(40, 30, 20, 1), true)
	}
	if s.charging {
		lx, ly := s.el.LampPos()
		r := 10 + math.Mod(ft, 30)
		vector.StrokeCircle(dst, float32(lx+ox), float32(ly+oy), float32(r), 1.5, rgba(255, 220, 140, 1-(r-10)/30), true)
	}
}

func (s *PaintingScene) drawHUD(dst *ebiten.Image, ft float64) {
	// pigmenti
	for i, f := range s.frags {
		x := float32(20 + i*18)
		if f.taken {
			vector.DrawFilledCircle(dst, x, 18, 6, rgba(f.r, f.g, f.b, 1), true)
		}
		vector.StrokeCircle(dst, x, 18, 6, 1, rgba(230, 220, 200, 0.8), true)
	}
	// integrità della tela: gocce di colore
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
	// batteria UV
	bx, by := float32(ScreenW-112), float32(12)
	drawText(dst, "UV", faceBold, float64(bx)-8, 6, rgba(200, 170, 255, 0.9), 1)
	vector.StrokeRect(dst, bx, by, 96, 10, 1, rgba(200, 170, 255, 0.8), false)
	bc := rgba(170, 120, 255, 0.9)
	if s.battery < 0.15 && (s.t/10)%2 == 0 {
		bc = rgba(255, 90, 90, 0.9)
	}
	vector.DrawFilledRect(dst, bx+2, by+2, float32(92*s.battery), 6, bc, false)
	if s.hasFlash {
		x := bx + 2 + 92*flashCost
		vector.StrokeLine(dst, x, by, x, by+10, 1, rgba(255, 255, 255, 0.5), false)
	}

	// minimappa
	mx, my := float32(ScreenW/2-80), float32(6)
	mop := &ebiten.DrawImageOptions{}
	mop.GeoM.Translate(float64(mx), float64(my))
	mop.ColorScale.Scale(0.55, 0.55, 0.55, 0.85)
	dst.DrawImage(worldMini, mop)
	vector.StrokeRect(dst, mx, my, 160, 45, 1, rgba(214, 178, 104, 0.6), false)
	sc := float32(160 / worldCanvas.W)
	vector.StrokeRect(dst, mx+float32(s.camX)*sc, my+float32(s.camY)*sc, ScreenW*sc, ScreenH*sc, 1, rgba(255, 255, 255, 0.3), false)
	for _, l := range s.lanterns {
		if !l.sky {
			vector.DrawFilledCircle(dst, mx+float32(l.x)*sc, my+float32(l.y)*sc, 1.5, rgba(255, 200, 100, 1), false)
		}
	}
	for _, f := range s.frags {
		if !f.taken && f.seen > 0.3 {
			vector.DrawFilledCircle(dst, mx+float32(f.x)*sc, my+float32(f.y)*sc, 2, rgba(f.r, f.g, f.b, 1), false)
		}
	}
	if s.mac.active {
		vector.DrawFilledCircle(dst, mx+float32(s.mac.x)*sc, my+float32(s.mac.y)*sc, 2.5, rgba(0, 0, 0, 1), false)
	}
	if (s.t/15)%3 != 0 {
		vector.DrawFilledCircle(dst, mx+float32(s.x)*sc, my+float32(s.y)*sc, 2, rgba(255, 255, 255, 1), false)
	}

	// pulsante LAMPO (per il touch)
	if s.hasFlash {
		ready := s.battery >= flashCost
		a := 0.35
		if ready {
			a = 0.75
		}
		vector.DrawFilledCircle(dst, flashBX, flashBY, flashBR, rgba(60, 30, 110, a*0.6), true)
		vector.StrokeCircle(dst, flashBX, flashBY, flashBR, 2, rgba(210, 180, 255, a), true)
		drawText(dst, "LAMPO", faceSmall, flashBX, flashBY-7, rgba(240, 230, 255, a+0.2), 0)
	}

	// suggerimenti
	if s.hintT > 0 && !s.dlg.Active() {
		a := math.Min(1, float64(s.hintT)/30)
		lines := wrap(s.hint, faceText, ScreenW-120)
		h := float32(14 + 19*len(lines))
		vector.DrawFilledRect(dst, 50, ScreenH-28-h, ScreenW-100, h, rgba(10, 8, 14, 0.7*a), false)
		for i, l := range lines {
			drawText(dst, l, faceText, ScreenW/2, float64(ScreenH-28-h+7+float32(i)*19), rgba(236, 228, 214, a), 0)
		}
	}
}
