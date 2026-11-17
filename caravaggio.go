package main

import (
	"image/color"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Omaggio procedurale a "La vocazione di san Matteo" (Caravaggio, 1599–1600, pubblico dominio).
// Una taverna buia, un tavolo di gabellieri a sinistra, due figure in piedi a destra
// e il fascio di luce che entra dall'alto e "chiama".

const (
	cravW = 1280.0
	cravH = 720.0
)

// Punti chiave della composizione (coordinate del mondo del Capitolo 2).
var (
	cravWindow  = [4]float64{860, 70, 1140, 220} // finestra in alto a destra
	cravTableY  = 450.0                          // piano del tavolo
	cravTableX0 = 140.0
	cravTableX1 = 650.0
	cravFloorY  = 590.0
	cravHand    = [2]float64{835, 345} // mano che indica
)

// Figura seduta o in piedi.
type cravFigure struct {
	x, headY  float64
	robe      color.NRGBA
	hat       bool
	feather   bool
	beard     bool
	standing  bool
	faceRight bool
}

var cravFigures = []cravFigure{
	{x: 195, headY: 400, robe: color.NRGBA{70, 62, 40, 255}},                         // il giovane che conta le monete
	{x: 300, headY: 355, robe: color.NRGBA{60, 46, 34, 255}, beard: true},            // il vecchio con gli occhiali
	{x: 415, headY: 345, robe: color.NRGBA{40, 32, 26, 255}, hat: true, beard: true}, // Matteo
	{x: 535, headY: 352, robe: color.NRGBA{150, 34, 26, 255}, hat: true, feather: true, faceRight: true},
	{x: 650, headY: 380, robe: color.NRGBA{120, 96, 60, 255}, hat: true, feather: true, faceRight: true}, // il ragazzo sullo sgabello
	{x: 930, headY: 315, robe: color.NRGBA{90, 66, 40, 255}, beard: true, standing: true},                // Pietro
	{x: 1020, headY: 300, robe: color.NRGBA{110, 40, 34, 255}, beard: true, standing: true},              // Cristo
}

var (
	cravOnce  sync.Once
	cravReady = make(chan *ebiten.Image, 1)
)

func PrewarmCaravaggio() {
	cravOnce.Do(func() { go func() { cravReady <- generateCaravaggio() }() })
}

func CaravaggioImage() *ebiten.Image {
	PrewarmCaravaggio()
	img := <-cravReady
	cravReady <- img
	return img
}

// beamPolygon restituisce i 4 vertici del fascio di luce con un'oscillazione (radianti).
func beamPolygon(sweep float64) [4][2]float64 {
	a := math.Pi + 0.42 + sweep         // verso sinistra e in basso
	dx, dy := math.Cos(a), -math.Sin(a) // y dello schermo verso il basso
	// bordo d'ingresso: lato sinistro della finestra
	p0 := [2]float64{cravWindow[0] + 30, cravWindow[1] + 10}
	p1 := [2]float64{cravWindow[0] + 70, cravWindow[3]}
	const L = 800.0 // il fascio si ferma sul muro di fondo
	// il fascio si allarga un poco
	nx, ny := -dy, dx
	p2 := [2]float64{p1[0] + dx*L - nx*60, p1[1] + dy*L - ny*60}
	p3 := [2]float64{p0[0] + dx*L + nx*20, p0[1] + dy*L + ny*20}
	return [4][2]float64{p0, p1, p2, p3}
}

func generateCaravaggio() *ebiten.Image {
	img := ebiten.NewImage(int(cravW), int(cravH))
	rng := rand.New(rand.NewSource(1600))
	umber := color.NRGBA{22, 16, 11, 255}
	img.Fill(umber)

	// fondo: pennellate scure in tutte le direzioni
	for i := 0; i < 9000; i++ {
		x, y := rng.Float64()*cravW, rng.Float64()*cravH
		a := rng.Float64() * math.Pi
		l := 6 + rng.Float64()*10
		c := lerpColor(color.NRGBA{18, 13, 9, 255}, color.NRGBA{48, 36, 24, 255}, rng.Float64())
		vector.StrokeLine(img, float32(x), float32(y), float32(x+math.Cos(a)*l), float32(y+math.Sin(a)*l), 3, c, true)
		yield(i)
	}

	// la luce "di base" sul muro: il fascio a riposo dipinto nel quadro
	bp := beamPolygon(0)
	var beam vector.Path
	beam.MoveTo(float32(bp[0][0]), float32(bp[0][1]))
	for _, p := range bp[1:] {
		beam.LineTo(float32(p[0]), float32(p[1]))
	}
	beam.Close()
	fillPath(img, &beam, color.NRGBA{120, 92, 56, 70})
	for i := 0; i < 3500; i++ {
		x, y := rng.Float64()*cravW, rng.Float64()*cravFloorY
		if !pointInQuad(x, y, bp) {
			continue
		}
		c := lerpColor(color.NRGBA{120, 92, 58, 255}, color.NRGBA{175, 140, 90, 255}, rng.Float64())
		c.A = 140
		vector.StrokeLine(img, float32(x-5), float32(y+2), float32(x+5), float32(y-2), 3, c, true)
		yield(i)
	}

	// finestra
	w := cravWindow
	vector.DrawFilledRect(img, float32(w[0]), float32(w[1]), float32(w[2]-w[0]), float32(w[3]-w[1]), color.NRGBA{58, 50, 38, 255}, false)
	for i := 1; i < 4; i++ {
		x := float32(w[0] + (w[2]-w[0])*float64(i)/4)
		vector.StrokeLine(img, x, float32(w[1]), x, float32(w[3]), 3, color.NRGBA{20, 15, 10, 255}, false)
	}
	vector.StrokeLine(img, float32(w[0]), float32((w[1]+w[3])/2), float32(w[2]), float32((w[1]+w[3])/2), 3, color.NRGBA{20, 15, 10, 255}, false)
	vector.StrokeRect(img, float32(w[0]), float32(w[1]), float32(w[2]-w[0]), float32(w[3]-w[1]), 5, color.NRGBA{30, 22, 15, 255}, false)

	// pavimento
	vector.DrawFilledRect(img, 0, float32(cravFloorY), cravW, float32(cravH-cravFloorY), color.NRGBA{30, 22, 15, 255}, false)
	for i := 0; i < 1800; i++ {
		x, y := rng.Float64()*cravW, cravFloorY+rng.Float64()*(cravH-cravFloorY)
		c := lerpColor(color.NRGBA{26, 19, 13, 255}, color.NRGBA{52, 40, 27, 255}, rng.Float64())
		vector.StrokeLine(img, float32(x-6), float32(y), float32(x+6), float32(y), 2, c, true)
	}

	// figure dietro il tavolo, poi il tavolo, poi le figure in piedi
	for _, f := range cravFigures {
		if !f.standing {
			drawCravFigure(img, f)
		}
	}
	drawCravTable(img, rng)
	for _, f := range cravFigures {
		if f.standing {
			drawCravFigure(img, f)
		}
	}
	// braccio che indica (Cristo)
	christ := cravFigures[6]
	vector.StrokeLine(img, float32(christ.x-14), float32(christ.headY+42), float32(cravHand[0]), float32(cravHand[1]), 9, color.NRGBA{110, 40, 34, 255}, true)
	vector.DrawFilledCircle(img, float32(cravHand[0]), float32(cravHand[1]), 6, color.NRGBA{214, 166, 124, 255}, true)
	vector.StrokeLine(img, float32(cravHand[0]), float32(cravHand[1]), float32(cravHand[0]-12), float32(cravHand[1]-1), 3, color.NRGBA{214, 166, 124, 255}, true)
	// aureola sottile
	vector.StrokeCircle(img, float32(christ.x), float32(christ.headY-6), 22, 1.2, color.NRGBA{200, 170, 100, 160}, true)
	return img
}

func drawCravFigure(img *ebiten.Image, f cravFigure) {
	skin := color.NRGBA{214, 166, 124, 255}
	shade := color.NRGBA{120, 84, 60, 255}
	hx, hy := f.x, f.headY
	// corpo
	bottom := cravTableY + 40
	width := 34.0
	if f.standing {
		bottom = cravFloorY + 30
		width = 40
	}
	var p vector.Path
	p.MoveTo(float32(hx-width*0.55), float32(hy+16))
	p.LineTo(float32(hx+width*0.55), float32(hy+16))
	p.LineTo(float32(hx+width), float32(bottom))
	p.LineTo(float32(hx-width), float32(bottom))
	p.Close()
	fillPath(img, &p, f.robe)
	// piega di luce sulla veste (luce da destra)
	hl := lerpColor(f.robe, color.NRGBA{255, 230, 190, 255}, 0.25)
	vector.StrokeLine(img, float32(hx+width*0.3), float32(hy+22), float32(hx+width*0.7), float32(bottom-10), 4, hl, true)
	// colletto bianco
	vector.DrawFilledRect(img, float32(hx-9), float32(hy+12), 18, 5, color.NRGBA{225, 215, 195, 255}, true)
	// testa: lato in luce e lato in ombra
	vector.DrawFilledCircle(img, float32(hx), float32(hy), 13, shade, true)
	off := 3.0
	if !f.faceRight {
		off = 2
	}
	vector.DrawFilledCircle(img, float32(hx+off), float32(hy), 11, skin, true)
	if f.beard {
		vector.DrawFilledCircle(img, float32(hx), float32(hy+9), 8, color.NRGBA{70, 52, 38, 255}, true)
	}
	if f.hat {
		vector.DrawFilledRect(img, float32(hx-16), float32(hy-15), 32, 7, color.NRGBA{18, 14, 10, 255}, true)
		vector.DrawFilledCircle(img, float32(hx), float32(hy-15), 11, color.NRGBA{18, 14, 10, 255}, true)
		if f.feather {
			vector.StrokeLine(img, float32(hx+4), float32(hy-20), float32(hx+22), float32(hy-34), 4, color.NRGBA{230, 225, 210, 255}, true)
		}
	}
	if f.standing {
		vector.DrawFilledCircle(img, float32(hx-4), float32(hy-8), 10, color.NRGBA{50, 36, 26, 255}, true) // capelli
	}
}

func drawCravTable(img *ebiten.Image, rng *rand.Rand) {
	x0, x1, y := cravTableX0, cravTableX1, cravTableY
	// tovaglia verde scuro e bordo
	vector.DrawFilledRect(img, float32(x0), float32(y), float32(x1-x0), 46, color.NRGBA{44, 52, 34, 255}, true)
	vector.DrawFilledRect(img, float32(x0), float32(y), float32(x1-x0), 8, color.NRGBA{80, 86, 56, 255}, true)
	// gambe
	for _, lx := range []float64{x0 + 20, x1 - 30} {
		vector.DrawFilledRect(img, float32(lx), float32(y+46), 10, float32(cravFloorY-y-46+10), color.NRGBA{34, 24, 16, 255}, true)
	}
	// monete, libro, calamaio
	for i := 0; i < 14; i++ {
		cx := 170 + rng.Float64()*120
		vector.DrawFilledCircle(img, float32(cx), float32(y+3+rng.Float64()*4), 3, color.NRGBA{205, 170, 80, 255}, true)
	}
	vector.DrawFilledRect(img, 340, float32(y-8), 46, 10, color.NRGBA{200, 190, 160, 255}, true)
	vector.DrawFilledRect(img, 470, float32(y-14), 10, 14, color.NRGBA{30, 30, 34, 255}, true)
	vector.StrokeLine(img, 476, float32(y-14), 486, float32(y-34), 1.5, color.NRGBA{230, 225, 210, 255}, true)
}

func pointInQuad(x, y float64, q [4][2]float64) bool {
	// quadrilatero convesso: stesso segno per tutti i lati
	sign := 0.0
	for i := 0; i < 4; i++ {
		a, b := q[i], q[(i+1)%4]
		cr := (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
		if cr == 0 {
			continue
		}
		if sign == 0 {
			sign = cr
		} else if (cr > 0) != (sign > 0) {
			return false
		}
	}
	return true
}
