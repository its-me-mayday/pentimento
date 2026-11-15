package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Dentro "La notte stellata": recupera i pigmenti, sfuggi alla Macchia,
// scopri il pentimento nascosto.

type fragment struct {
	x, y    float64
	name    string
	r, g, b uint8
	clue    string
	taken   bool
}

type particle struct {
	x, y, px, py float64
	age          int
}

type paintPhase int

const (
	phEnter paintPhase = iota
	phPlay
	phReveal
	phEnd
)

type PaintingScene struct {
	t     int
	phase paintPhase
	dlg   Dialogue
	off   *ebiten.Image
	rng   *rand.Rand
	parts []particle

	x, y      float64 // Elena
	tx, ty    float64 // destinazione click/tocco
	hasTarget bool
	frags     []*fragment
	taken     int
	restore   float64 // 0..1 mostrato
	mx, my    float64 // la Macchia
	stainOn   bool
	stainFade float64
	shake     float64
	revealT   float64
}

func NewPaintingScene() *PaintingScene {
	s := &PaintingScene{
		off: ebiten.NewImage(ScreenW, ScreenH),
		rng: rand.New(rand.NewSource(7)),
		x:   300, y: 320,
		mx: 600, my: 20,
	}
	s.frags = []*fragment{
		{250, 300, "Blu oltremare", 40, 70, 220,
			"Elena. Se senti la mia voce, la lampada funziona. Non avere paura: dentro la tela si sente solo ciò che il pittore ha coperto.", false},
		{470, 290, "Giallo di cromo", 245, 200, 40,
			"Questo quadro è arrivato in laboratorio di notte, senza documenti. Mi hanno pagata in contanti per «ripulirlo» in una settimana.", false},
		{545, 175, "Bianco di titanio", 250, 250, 245,
			"Guarda il bianco delle stelle: è bianco di titanio. Nel 1889 non era ancora in commercio. Questa tela non è un originale.", false},
		{320, 70, "Verde smeraldo", 30, 170, 110,
			"Non sono l'unica ad averlo capito. Da settimane una macchina grigia resta parcheggiata sotto il laboratorio.", false},
		{110, 175, "Nero d'avorio", 60, 55, 50,
			"Se ti diranno che è stato un incidente, non crederci. Guarda sotto il cielo. Ho lasciato lì il resto.", false},
	}
	for i := 0; i < 420; i++ {
		s.parts = append(s.parts, s.newParticle())
	}
	s.dlg.Start(
		Line{"", "Odore di olio di lino. Vento caldo. Il cielo si muove, pennellata dopo pennellata.", false},
		Line{"Elena", "Sono... dentro il quadro? I colori sono spenti. Come se qualcuno li avesse rubati.", false},
		Line{"", "Recupera i 5 pigmenti perduti. Muoviti con WASD / frecce, oppure clicca o tocca dove vuoi andare.", false},
	)
	return s
}

func (s *PaintingScene) newParticle() particle {
	x, y := s.rng.Float64()*ScreenW, s.rng.Float64()*horizonY
	return particle{x, y, x, y, s.rng.Intn(120)}
}

func (s *PaintingScene) Update(g *Game) error {
	s.t++
	ft := float64(s.t)
	target := float64(s.taken) / float64(len(s.frags))
	s.restore += (target - s.restore) * 0.03
	s.shake *= 0.9
	for i := range s.parts {
		p := &s.parts[i]
		p.px, p.py = p.x, p.y
		dx, dy := flow(p.x, p.y, ft/60)
		p.x += dx * 0.9
		p.y += dy * 0.9
		p.age++
		if p.age > 160 || p.x < 0 || p.x > ScreenW || p.y < 0 || p.y > horizonY {
			*p = s.newParticle()
		}
	}

	if s.dlg.Update() {
		return nil
	}

	switch s.phase {
	case phEnter:
		s.phase = phPlay
	case phPlay:
		s.updatePlayer()
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
	}
	return nil
}

func (s *PaintingScene) updatePlayer() {
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
	// click/tocco (anche tenendo premuto)
	if x, y, ok := pointer(); ok && (ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.AppendTouchIDs(nil)) > 0) {
		s.tx, s.ty, s.hasTarget = float64(x), float64(y), true
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
	s.x = clamp(s.x, 12, ScreenW-12)
	s.y = clamp(s.y, 12, ScreenH-12)
}

func (s *PaintingScene) updateStain() {
	if !s.stainOn {
		return
	}
	speed := 0.45 + 0.12*float64(s.taken)
	vx, vy := s.x-s.mx, s.y-s.my
	d := math.Hypot(vx, vy)
	if d > 0 {
		wob := math.Sin(float64(s.t)/30) * 0.6
		s.mx += (vx/d + -vy/d*wob) * speed
		s.my += (vy/d + vx/d*wob) * speed
	}
	if d < 24 {
		// respinta: la Macchia "cancella" Elena per un istante
		s.x -= vx / d * 60
		s.y -= vy / d * 60
		s.x = clamp(s.x, 12, ScreenW-12)
		s.y = clamp(s.y, 12, ScreenH-12)
		s.mx += vx / d * -20
		s.my += vy / d * -20
		s.shake = 10
		s.restore = math.Max(0, s.restore-0.15)
		s.hasTarget = false
	}
}

func (s *PaintingScene) checkFragments() {
	for _, f := range s.frags {
		if f.taken || math.Hypot(f.x-s.x, f.y-s.y) > 16 {
			continue
		}
		f.taken = true
		s.taken++
		s.stainOn = true
		lines := []Line{{"Ada", f.clue, true}}
		if s.taken == 1 {
			lines = append(lines, Line{"", "Qualcosa si muove tra le pennellate. Una macchia d'inchiostro, densa, affamata. Sta venendo verso di te.", false})
		}
		if s.taken == len(s.frags) {
			s.phase = phReveal
			lines = append(lines, Line{"", "La Macchia si scioglie. Sotto il cielo, la pittura si fa trasparente...", false})
		}
		s.dlg.Start(lines...)
		return
	}
}

func (s *PaintingScene) Draw(screen *ebiten.Image) {
	ft := float64(s.t)
	s.off.Clear()
	s.off.DrawImage(paintingBG, nil)
	for _, p := range s.parts {
		a := 0.55 * math.Sin(math.Pi*float64(p.age)/160)
		vector.StrokeLine(s.off, float32(p.px), float32(p.py), float32(p.x+(p.x-p.px)*3), float32(p.y+(p.y-p.py)*3), 1.6, rgba(220, 230, 255, a), true)
	}
	// la Macchia
	if s.stainOn {
		a := 0.9 * (1 - s.stainFade)
		for i := 0; i < 7; i++ {
			ang := float64(i)/7*2*math.Pi + ft/40
			r := 9 + 4*math.Sin(ft/13+float64(i)*1.7)
			vector.DrawFilledCircle(s.off, float32(s.mx+math.Cos(ang)*r), float32(s.my+math.Sin(ang)*r), float32(10+3*math.Sin(ft/9+float64(i))), rgba(6, 6, 12, a), true)
		}
		vector.DrawFilledCircle(s.off, float32(s.mx), float32(s.my), 14, rgba(0, 0, 0, a), true)
	}

	sx := (s.rng.Float64()*2 - 1) * s.shake
	sy := (s.rng.Float64()*2 - 1) * s.shake
	applyGrade(screen, s.off, s.restore, ft, sx, sy)

	// pentimento rivelato
	if s.revealT > 0 {
		a := s.revealT
		drawText(screen, "copia n. 3", faceTitle, ScreenW/2, 135, rgba(250, 240, 210, a*0.35), 0)
		drawText(screen, "— C.", faceItalic, ScreenW/2+150, 205, rgba(250, 240, 210, a*0.6), 0)
	}

	// pigmenti: brillano a colori anche nel mondo spento
	for _, f := range s.frags {
		if f.taken {
			continue
		}
		p := 0.5 + 0.5*math.Sin(ft/12+f.x)
		for k := 4; k >= 1; k-- {
			vector.DrawFilledCircle(screen, float32(f.x), float32(f.y), float32(4+float64(k)*3+p*3), rgba(f.r, f.g, f.b, 0.12), true)
		}
		vector.DrawFilledCircle(screen, float32(f.x), float32(f.y), 5, rgba(f.r, f.g, f.b, 1), true)
	}

	// Elena: silhouette con lanterna
	ex, ey := float32(s.x+sx), float32(s.y+sy)
	for k := 5; k >= 1; k-- {
		vector.DrawFilledCircle(screen, ex+5, ey-4, float32(k)*6, rgba(255, 200, 120, 0.05), true)
	}
	vector.DrawFilledRect(screen, ex-4, ey-6, 8, 14, rgba(28, 22, 30, 1), true)
	vector.DrawFilledCircle(screen, ex, ey-10, 4.5, rgba(28, 22, 30, 1), true)
	vector.DrawFilledCircle(screen, ex+6, ey-2, 2.5, rgba(255, 220, 150, 1), true)

	// HUD pigmenti
	if s.phase == phPlay || s.phase == phEnter {
		for i, f := range s.frags {
			x := float32(20 + i*18)
			if f.taken {
				vector.DrawFilledCircle(screen, x, 18, 6, rgba(f.r, f.g, f.b, 1), true)
			}
			vector.StrokeCircle(screen, x, 18, 6, 1, rgba(230, 220, 200, 0.8), true)
		}
		if s.taken > 0 && !s.dlg.Active() {
			last := ""
			for _, f := range s.frags {
				if f.taken {
					last = f.name
				}
			}
			drawText(screen, last+" recuperato", faceSmall, 112, 11, rgba(230, 220, 200, 0.8), -1)
		}
	}

	if s.phase == phEnd && !s.dlg.Active() {
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 0.7), false)
		drawText(screen, "PENTIMENTO", faceTitle, ScreenW/2, 90, rgba(214, 178, 104, 1), 0)
		drawText(screen, "Capitolo 1 — La notte stellata", faceItalic, ScreenW/2, 165, rgba(230, 222, 205, 1), 0)
		drawText(screen, "continua...", faceText, ScreenW/2, 205, rgba(200, 190, 170, 1), 0)
		if (s.t/30)%2 == 0 {
			drawText(screen, "Clicca per tornare al titolo", faceSmall, ScreenW/2, 300, rgba(200, 190, 170, 0.8), 0)
		}
	}

	s.dlg.Draw(screen)
}
