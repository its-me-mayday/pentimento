package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Omaggio procedurale a "La notte stellata" (Van Gogh, 1889 — pubblico dominio):
// niente immagini, solo pennellate che seguono un campo di vortici.

const horizonY = 245

type swirl struct{ x, y, r, s float64 }

var swirls = []swirl{
	{300, 115, 85, 1.0},
	{190, 85, 55, -0.8},
	{430, 150, 50, 0.9},
	{90, 120, 45, 0.6},
}

type star struct{ x, y, r float64 }

var stars = []star{
	{40, 40, 9}, {150, 30, 8}, {245, 50, 7}, {380, 40, 9}, {470, 70, 8},
	{520, 175, 10}, {360, 190, 7}, {130, 190, 8}, {610, 150, 7},
}

const moonX, moonY = 565, 62

// flow restituisce la direzione delle pennellate nel punto (x, y) al tempo t.
func flow(x, y, t float64) (float64, float64) {
	vx := 1.0
	vy := 0.25 * math.Sin(x/45+y/70+t*0.4)
	for _, s := range swirls {
		dx, dy := x-s.x, y-s.y
		d := math.Hypot(dx, dy)
		w := math.Exp(-(d * d) / (2 * s.r * s.r))
		if d > 0.001 {
			vx += -dy / d * w * 2.6 * s.s
			vy += dx / d * w * 2.6 * s.s
		}
	}
	l := math.Hypot(vx, vy)
	return vx / l, vy / l
}

var (
	skyDeep  = color.NRGBA{12, 26, 78, 255}
	skyMid   = color.NRGBA{34, 70, 150, 255}
	skyLight = color.NRGBA{88, 132, 196, 255}
	paleBlue = color.NRGBA{150, 190, 220, 255}
	chrome   = color.NRGBA{240, 205, 70, 255}
	cream    = color.NRGBA{246, 236, 190, 255}
)

func generateStarryNight() *ebiten.Image {
	img := ebiten.NewImage(ScreenW, ScreenH)
	rng := rand.New(rand.NewSource(1889))

	// fondo del cielo
	for y := 0; y < ScreenH; y += 2 {
		t := clamp(float64(y)/horizonY, 0, 1)
		vector.DrawFilledRect(img, 0, float32(y), ScreenW, 2, lerpColor(skyDeep, skyMid, t), false)
	}

	// pennellate del cielo
	for i := 0; i < 11000; i++ {
		x := rng.Float64() * ScreenW
		y := rng.Float64() * (horizonY + 10)
		c := skyPalette(rng, x, y)
		stroke(img, x, y, 5+rng.Float64()*7, 1.6+rng.Float64()*1.6, c)
	}

	// stelle: aloni concentrici
	for _, s := range stars {
		for k := 5; k >= 1; k-- {
			r := s.r * (1 + float64(k)*0.55)
			ring(img, rng, s.x, s.y, r, lerpColor(chrome, paleBlue, float64(k)/6), 0.75)
		}
		vector.DrawFilledCircle(img, float32(s.x), float32(s.y), float32(s.r*0.7), cream, true)
	}

	// luna
	for k := 7; k >= 1; k-- {
		ring(img, rng, moonX, moonY, 18+float64(k)*6, lerpColor(chrome, color.NRGBA{230, 150, 40, 255}, float64(k)/8), 0.85)
	}
	vector.DrawFilledCircle(img, moonX, moonY, 18, chrome, true)
	vector.DrawFilledCircle(img, moonX+7, moonY-4, 15, color.NRGBA{200, 130, 40, 255}, true)

	drawHills(img, rng)
	drawVillage(img, rng)
	drawCypress(img, rng)
	return img
}

func skyPalette(rng *rand.Rand, x, y float64) color.NRGBA {
	// vicino ai vortici e alla luna compaiono pennellate chiare e gialle
	near := 0.0
	for _, s := range swirls {
		d := math.Hypot(x-s.x, y-s.y)
		near = math.Max(near, math.Exp(-(d*d)/(2*s.r*s.r*0.5)))
	}
	dm := math.Hypot(x-moonX, y-moonY)
	near = math.Max(near, math.Exp(-(dm*dm)/(2*60*60)))
	r := rng.Float64()
	switch {
	case r < near*0.35:
		return lerpColor(chrome, cream, rng.Float64())
	case r < near*0.8+0.15:
		return lerpColor(skyLight, paleBlue, rng.Float64())
	case r < 0.75:
		return lerpColor(skyMid, skyLight, rng.Float64())
	default:
		return lerpColor(skyDeep, skyMid, rng.Float64())
	}
}

func stroke(img *ebiten.Image, x, y, length, width float64, c color.NRGBA) {
	dx, dy := flow(x, y, 0)
	h := length / 2
	vector.StrokeLine(img, float32(x-dx*h), float32(y-dy*h), float32(x+dx*h), float32(y+dy*h), float32(width), c, true)
}

func ring(img *ebiten.Image, rng *rand.Rand, cx, cy, r float64, c color.NRGBA, a float64) {
	n := int(r * 1.3)
	for i := 0; i < n; i++ {
		ang := float64(i)/float64(n)*2*math.Pi + rng.Float64()*0.1
		x, y := cx+math.Cos(ang)*r, cy+math.Sin(ang)*r
		tx, ty := -math.Sin(ang), math.Cos(ang)
		cc := c
		cc.A = uint8(255 * a)
		vector.StrokeLine(img, float32(x-tx*3), float32(y-ty*3), float32(x+tx*3), float32(y+ty*3), 2, cc, true)
	}
}

func drawHills(img *ebiten.Image, rng *rand.Rand) {
	layers := []struct {
		base, amp, freq float64
		c1, c2          color.NRGBA
	}{
		{232, 14, 90, color.NRGBA{40, 60, 120, 255}, color.NRGBA{60, 90, 150, 255}},
		{258, 10, 60, color.NRGBA{20, 40, 80, 255}, color.NRGBA{40, 70, 100, 255}},
		{300, 8, 40, color.NRGBA{18, 36, 50, 255}, color.NRGBA{30, 60, 60, 255}},
	}
	for _, l := range layers {
		var p vector.Path
		p.MoveTo(0, ScreenH)
		for x := 0.0; x <= ScreenW; x += 8 {
			p.LineTo(float32(x), float32(l.base+math.Sin(x/l.freq)*l.amp))
		}
		p.LineTo(ScreenW, ScreenH)
		p.Close()
		fillPath(img, &p, l.c1)
		for i := 0; i < 900; i++ {
			x := rng.Float64() * ScreenW
			top := l.base + math.Sin(x/l.freq)*l.amp
			y := top + rng.Float64()*(ScreenH-top)
			slope := math.Cos(x/l.freq) * l.amp / l.freq
			n := math.Hypot(1, slope)
			dx, dy := 1/n, slope/n
			c := lerpColor(l.c1, l.c2, rng.Float64())
			vector.StrokeLine(img, float32(x-dx*4), float32(y-dy*4), float32(x+dx*4), float32(y+dy*4), 2, c, true)
		}
	}
}

func drawVillage(img *ebiten.Image, rng *rand.Rand) {
	wall := color.NRGBA{40, 50, 70, 255}
	roof := color.NRGBA{25, 30, 50, 255}
	window := color.NRGBA{250, 210, 90, 255}
	for i := 0; i < 16; i++ {
		x := 200 + float64(i)*22 + rng.Float64()*8
		w := 14 + rng.Float64()*8
		h := 9 + rng.Float64()*7
		y := 272 + rng.Float64()*14
		vector.DrawFilledRect(img, float32(x), float32(y), float32(w), float32(h), wall, true)
		var p vector.Path
		p.MoveTo(float32(x-2), float32(y))
		p.LineTo(float32(x+w/2), float32(y-7))
		p.LineTo(float32(x+w+2), float32(y))
		p.Close()
		fillPath(img, &p, roof)
		if rng.Float64() < 0.7 {
			vector.DrawFilledRect(img, float32(x+w/2-2), float32(y+3), 3, 3, window, false)
		}
	}
	// campanile
	vector.DrawFilledRect(img, 372, 248, 10, 34, wall, true)
	var p vector.Path
	p.MoveTo(370, 249)
	p.LineTo(377, 212)
	p.LineTo(384, 249)
	p.Close()
	fillPath(img, &p, roof)
}

func drawCypress(img *ebiten.Image, rng *rand.Rand) {
	dark := color.NRGBA{14, 22, 16, 255}
	var p vector.Path
	p.MoveTo(55, ScreenH)
	for y := ScreenH; y >= 30; y -= 10 {
		t := float64(ScreenH-y) / float64(ScreenH-30)
		half := 34 * math.Sin(math.Pi*(0.15+0.85*(1-t))) * (1 - t*0.7)
		p.LineTo(float32(95-half+math.Sin(float64(y)/18)*6), float32(y))
	}
	for y := 30; y <= ScreenH; y += 10 {
		t := float64(ScreenH-y) / float64(ScreenH-30)
		half := 34 * math.Sin(math.Pi*(0.15+0.85*(1-t))) * (1 - t*0.7)
		p.LineTo(float32(95+half+math.Cos(float64(y)/22)*6), float32(y))
	}
	p.Close()
	fillPath(img, &p, dark)
	// fiamme verticali
	for i := 0; i < 700; i++ {
		y := 40 + rng.Float64()*(ScreenH-40)
		t := (ScreenH - y) / (ScreenH - 30)
		half := 30 * math.Sin(math.Pi*(0.15+0.85*(1-t))) * (1 - t*0.7)
		x := 95 + (rng.Float64()*2-1)*half
		c := lerpColor(color.NRGBA{20, 40, 26, 255}, color.NRGBA{60, 80, 40, 255}, rng.Float64())
		sway := math.Sin(y/20) * 3
		vector.StrokeLine(img, float32(x), float32(y), float32(x+sway), float32(y-10), 2, c, true)
	}
}
