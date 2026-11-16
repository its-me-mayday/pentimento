package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Elena Sarti, restauratrice: camice, foulard ocra, lente da restauro sulla fronte
// e la lampada UV di Ada in mano. È l'unica cosa a colori pieni dentro il quadro.

type dab struct {
	x, y, ang float64
	c         color.NRGBA
	life      int
}

type Elena struct {
	X, Y  float64
	Face  float64 // 1 destra, -1 sinistra
	walk  float64 // fase della camminata
	speed float64 // 0..1 quanto sta camminando
	hurt  int
	dist  float64 // distanza dall'ultima impronta
	trail []dab
}

func NewElena(x, y float64) *Elena { return &Elena{X: x, Y: y, Face: 1} }

// LampPos: posizione della lampada UV (per lo shader e i pigmenti).
func (e *Elena) LampPos() (float64, float64) { return e.X + e.Face*9, e.Y - 17 }

// Step aggiorna animazione e impronte dopo uno spostamento (mx, my).
func (e *Elena) Step(mx, my float64, palette []color.NRGBA) {
	d := math.Hypot(mx, my)
	if mx > 0.05 {
		e.Face = 1
	} else if mx < -0.05 {
		e.Face = -1
	}
	e.speed += (math.Min(d/1.7, 1) - e.speed) * 0.2
	e.walk += d * 0.22
	if e.hurt > 0 {
		e.hurt--
	}

	// impronte di colore: dove passa Elena il quadro ricorda i pigmenti recuperati
	e.dist += d
	if e.dist > 9 {
		e.dist = 0
		c := color.NRGBA{235, 225, 200, 255}
		if len(palette) > 0 {
			c = palette[int(e.walk*7)%len(palette)]
		}
		side := 3.0
		if int(e.walk/math.Pi)%2 == 0 {
			side = -3
		}
		e.trail = append(e.trail, dab{x: e.X + side*0.5, y: e.Y + 1, ang: math.Atan2(my, mx), c: c, life: 150})
	}
	alive := e.trail[:0]
	for _, t := range e.trail {
		t.life--
		if t.life > 0 {
			alive = append(alive, t)
		}
	}
	e.trail = alive
}

func (e *Elena) Hit() { e.hurt = 45 }

var (
	coatCol   = color.NRGBA{232, 224, 204, 255}
	coatShade = color.NRGBA{188, 176, 156, 255}
	pantsCol  = color.NRGBA{42, 38, 52, 255}
	bootCol   = color.NRGBA{58, 36, 26, 255}
	skinCol   = color.NRGBA{233, 196, 160, 255}
	hairCol   = color.NRGBA{43, 29, 22, 255}
	scarfCol  = color.NRGBA{184, 72, 46, 255}
	lampBody  = color.NRGBA{70, 74, 86, 255}
	uvCol     = color.NRGBA{170, 120, 255, 255}
)

// DrawTrail disegna le impronte (sotto al personaggio).
func (e *Elena) DrawTrail(dst *ebiten.Image, ox, oy float64) {
	for _, t := range e.trail {
		a := float64(t.life) / 150 * 0.8
		c := t.c
		c.A = uint8(255 * a)
		dx, dy := math.Cos(t.ang)*3, math.Sin(t.ang)*3
		x, y := t.x+ox, t.y+oy
		vector.StrokeLine(dst, float32(x-dx), float32(y-dy), float32(x+dx), float32(y+dy), 2.5, c, true)
	}
}

// DrawLight disegna l'alone viola della lampada; power 0..1 è la carica.
func (e *Elena) DrawLight(dst *ebiten.Image, t, ox, oy, power float64) {
	lx, ly := e.LampPos()
	lx, ly = lx+ox, ly+oy
	flick := (0.45 + 0.55*power) * (1 + 0.06*math.Sin(t/3.1) + 0.04*math.Sin(t/1.7))
	if power < 0.15 {
		flick *= 0.6 + 0.4*math.Abs(math.Sin(t/2.3)*math.Sin(t/5.1)) // sfarfalla quando è scarica
	}
	for k := 6; k >= 1; k-- {
		r := float32(float64(k) * 9 * flick)
		vector.DrawFilledCircle(dst, float32(lx+e.Face*float64(k)*3), float32(ly+float64(k)), r, rgba(150, 90, 255, 0.045), true)
	}
}

func (e *Elena) Draw(dst *ebiten.Image, t float64, ox, oy float64) {
	if e.hurt > 0 && (e.hurt/3)%2 == 0 {
		return // lampeggia quando la Macchia la colpisce
	}
	f := e.Face
	x, y := e.X+ox, e.Y+oy
	sw := math.Sin(e.walk) * e.speed
	bob := math.Abs(math.Cos(e.walk))*1.3*e.speed + math.Sin(t/25)*0.4*(1-e.speed)
	by := y - bob
	px := func(dx float64) float32 { return float32(x + dx*f) }
	py := func(dy float64) float32 { return float32(by + dy) }

	// ombra
	ellipse(dst, x, y+1, 8, 2.5, rgba(0, 0, 0, 0.35))

	// gambe e stivali
	for _, s := range []float64{1, -1} {
		fx := s * sw * 3.5
		lift := math.Max(0, s*math.Cos(e.walk)) * 1.5 * e.speed
		vector.StrokeLine(dst, px(s*1.5), py(-10), float32(x+(fx+s*1.2)*f), float32(y-2-lift), 3, pantsCol, true)
		vector.DrawFilledRect(dst, float32(x+(fx+s*1.2)*f)-2, float32(y-3-lift), 4.5, 2.5, bootCol, true)
	}

	// braccio dietro
	vector.StrokeLine(dst, px(-2), py(-21), px(-3-sw*2), py(-13), 2.5, coatShade, true)

	// camice: trapezio con orlo che ondeggia
	hem := sw * 1.5
	var p vector.Path
	p.MoveTo(px(-4.5), py(-24))
	p.LineTo(px(4.5), py(-24))
	p.LineTo(px(6.5+hem), py(-8))
	p.LineTo(px(-7+hem), py(-8))
	p.Close()
	fillPath(dst, &p, coatCol)
	vector.StrokeLine(dst, px(-4.5), py(-23), px(-7+hem), py(-8.5), 1.5, coatShade, true)
	vector.StrokeLine(dst, px(1), py(-22), px(1.2+hem*0.5), py(-9), 0.8, coatShade, true) // abbottonatura
	vector.DrawFilledRect(dst, px(-0.5)-1.5, py(-15), 3, 0.8, coatShade, false)           // tasca

	// foulard ocra con la coda che svolazza all'indietro
	vector.DrawFilledRect(dst, float32(x-4), py(-25.5), 8, 2.5, scarfCol, true)
	tail := math.Sin(t/6+e.walk) * 1.5
	vector.StrokeLine(dst, px(-3.5), py(-24.5), px(-8-e.speed*2), py(-21+tail), 2, scarfCol, true)

	// braccio avanti con la lampada UV
	lx, ly := e.LampPos()
	lx, ly = lx+ox, ly+oy
	vector.StrokeLine(dst, px(2.5), py(-22), float32(lx-f*2), float32(ly+bob*0.5), 2.5, coatCol, true)
	vector.DrawFilledCircle(dst, float32(lx-f*2), float32(ly+bob*0.5), 1.6, skinCol, true)
	vector.DrawFilledRect(dst, float32(lx-f*1)-2, float32(ly-1.5+bob*0.5), 4, 3, lampBody, true)
	vector.DrawFilledCircle(dst, float32(lx+f*2), float32(ly+bob*0.5), 2.2, uvCol, true)

	// testa: capelli raccolti dietro, viso davanti
	hx, hy := x+f*0.5, by-30
	vector.DrawFilledCircle(dst, float32(hx-f*5), float32(hy-3.5), 2.8, hairCol, true) // chignon
	vector.DrawFilledCircle(dst, float32(hx-f*1), float32(hy-1), 5.4, hairCol, true)
	vector.DrawFilledCircle(dst, float32(hx+f*1.3), float32(hy+0.8), 4.1, skinCol, true)
	vector.StrokeLine(dst, float32(hx-f*6), float32(hy-6), float32(hx-f*3.5), float32(hy-1.5), 0.8, color.NRGBA{200, 160, 60, 255}, true) // matita nel chignon
	// lente da restauro sulla fronte
	vector.StrokeLine(dst, float32(hx-f*3), float32(hy-3.6), float32(hx+f*4), float32(hy-3.6), 1, color.NRGBA{90, 90, 100, 255}, true)
	vector.DrawFilledCircle(dst, float32(hx+f*3.8), float32(hy-3.4), 1.6, color.NRGBA{150, 200, 230, 255}, true)
	// occhio
	vector.DrawFilledCircle(dst, float32(hx+f*2.6), float32(hy-0.3), 0.8, hairCol, true)

	// quando viene colpita, schegge d'inchiostro
	if e.hurt > 0 {
		for i := 0; i < 6; i++ {
			a := float64(i)/6*2*math.Pi + t/5
			r := float64(45-e.hurt) * 0.6
			vector.DrawFilledCircle(dst, float32(x+math.Cos(a)*r), float32(by-15+math.Sin(a)*r), 1.5, rgba(10, 10, 20, 0.8), true)
		}
	}
}

func ellipse(dst *ebiten.Image, cx, cy, rx, ry float64, c color.NRGBA) {
	var p vector.Path
	for i := 0; i <= 16; i++ {
		a := float64(i) / 16 * 2 * math.Pi
		x, y := float32(cx+math.Cos(a)*rx), float32(cy+math.Sin(a)*ry)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	fillPath(dst, &p, c)
}
