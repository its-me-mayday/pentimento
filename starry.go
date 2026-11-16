package main

import (
	"image/color"
	"math"
	"math/rand"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Omaggio procedurale a "La notte stellata" (Van Gogh, 1889 — pubblico dominio):
// niente immagini, solo pennellate che seguono un campo di vortici.

type swirl struct{ x, y, r, s float64 }

type star struct{ x, y, r float64 }

// Canvas descrive un quadro generabile: la miniatura del laboratorio o il mondo di gioco.
type Canvas struct {
	W, H         float64
	Horizon      float64
	Swirls       []swirl
	Stars        []star
	MoonX, MoonY float64
	CypressX     float64
	Villages     [][2]float64 // [x iniziale, numero case]
	Church       float64
	Seed         int64
}

// smallCanvas: la tela nel laboratorio e nel titolo (una schermata).
var smallCanvas = &Canvas{
	W: ScreenW, H: ScreenH, Horizon: 245,
	Swirls: []swirl{{300, 115, 85, 1.0}, {190, 85, 55, -0.8}, {430, 150, 50, 0.9}, {90, 120, 45, 0.6}},
	Stars: []star{
		{40, 40, 9}, {150, 30, 8}, {245, 50, 7}, {380, 40, 9}, {470, 70, 8},
		{520, 175, 10}, {360, 190, 7}, {130, 190, 8}, {610, 150, 7},
	},
	MoonX: 565, MoonY: 62, CypressX: 95,
	Villages: [][2]float64{{200, 16}}, Church: 372, Seed: 1889,
}

// worldCanvas: il quadro in cui entra Elena, tre schermate di larghezza.
var worldCanvas = &Canvas{
	W: 1920, H: 540, Horizon: 370,
	Swirls: []swirl{
		{330, 140, 90, 1}, {640, 100, 70, -0.9}, {980, 160, 115, 1}, {1320, 110, 80, -0.8},
		{1610, 170, 90, 1}, {170, 250, 50, 0.6}, {1150, 270, 60, -0.7}, {1840, 230, 55, 0.8},
	},
	Stars: []star{
		{60, 50, 9}, {200, 110, 8}, {470, 60, 9}, {520, 250, 8}, {760, 230, 10}, {820, 60, 8},
		{1110, 60, 9}, {1240, 230, 8}, {1330, 175, 10}, {1470, 70, 8}, {1500, 280, 7}, {1690, 60, 9},
		{1880, 140, 8}, {60, 300, 7},
	},
	MoonX: 1790, MoonY: 70, CypressX: 300,
	Villages: [][2]float64{{420, 13}, {1000, 16}}, Church: 1180, Seed: 1890,
}

// Flow restituisce la direzione delle pennellate nel punto (x, y) al tempo t.
func (c *Canvas) Flow(x, y, t float64) (float64, float64) {
	vx := 1.0
	vy := 0.25 * math.Sin(x/45+y/70+t*0.4)
	for _, s := range c.Swirls {
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

var (
	worldOnce  sync.Once
	worldReady = make(chan *ebiten.Image, 1)
)

// PrewarmWorld genera il quadro grande in sottofondo (cedendo il passo al game loop),
// così l'ingresso nel quadro non blocca il gioco.
func PrewarmWorld() {
	worldOnce.Do(func() {
		go func() { worldReady <- worldCanvas.Generate() }()
	})
}

// WorldImage attende (se serve) il quadro grande e lo restituisce.
func WorldImage() *ebiten.Image {
	PrewarmWorld()
	img := <-worldReady
	worldReady <- img
	return img
}

func yield(i int) {
	if i%300 == 0 {
		runtime.Gosched()
	}
}

func (c *Canvas) Generate() *ebiten.Image {
	img := ebiten.NewImage(int(c.W), int(c.H))
	rng := rand.New(rand.NewSource(c.Seed))

	for y := 0.0; y < c.H; y += 2 {
		t := clamp(y/c.Horizon, 0, 1)
		vector.DrawFilledRect(img, 0, float32(y), float32(c.W), 2, lerpColor(skyDeep, skyMid, t), false)
	}

	n := int(c.W * (c.Horizon + 10) / 14.3) // stessa densità della miniatura
	for i := 0; i < n; i++ {
		x := rng.Float64() * c.W
		y := rng.Float64() * (c.Horizon + 10)
		c.stroke(img, x, y, 5+rng.Float64()*7, 1.6+rng.Float64()*1.6, c.skyPalette(rng, x, y))
		yield(i)
	}

	for _, s := range c.Stars {
		for k := 5; k >= 1; k-- {
			r := s.r * (1 + float64(k)*0.55)
			ring(img, rng, s.x, s.y, r, lerpColor(chrome, paleBlue, float64(k)/6), 0.75)
		}
		vector.DrawFilledCircle(img, float32(s.x), float32(s.y), float32(s.r*0.7), cream, true)
	}

	for k := 7; k >= 1; k-- {
		ring(img, rng, c.MoonX, c.MoonY, 18+float64(k)*6, lerpColor(chrome, color.NRGBA{230, 150, 40, 255}, float64(k)/8), 0.85)
	}
	vector.DrawFilledCircle(img, float32(c.MoonX), float32(c.MoonY), 18, chrome, true)
	vector.DrawFilledCircle(img, float32(c.MoonX+7), float32(c.MoonY-4), 15, color.NRGBA{200, 130, 40, 255}, true)

	c.drawHills(img, rng)
	c.drawVillage(img, rng)
	c.drawCypress(img, rng)
	return img
}

func (c *Canvas) skyPalette(rng *rand.Rand, x, y float64) color.NRGBA {
	near := 0.0
	for _, s := range c.Swirls {
		d := math.Hypot(x-s.x, y-s.y)
		near = math.Max(near, math.Exp(-(d*d)/(2*s.r*s.r*0.5)))
	}
	dm := math.Hypot(x-c.MoonX, y-c.MoonY)
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

func (c *Canvas) stroke(img *ebiten.Image, x, y, length, width float64, col color.NRGBA) {
	dx, dy := c.Flow(x, y, 0)
	h := length / 2
	vector.StrokeLine(img, float32(x-dx*h), float32(y-dy*h), float32(x+dx*h), float32(y+dy*h), float32(width), col, true)
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

func (c *Canvas) drawHills(img *ebiten.Image, rng *rand.Rand) {
	layers := []struct {
		base, amp, freq float64
		c1, c2          color.NRGBA
	}{
		{c.Horizon - 13, 14, 90, color.NRGBA{40, 60, 120, 255}, color.NRGBA{60, 90, 150, 255}},
		{c.Horizon + 13, 10, 60, color.NRGBA{20, 40, 80, 255}, color.NRGBA{40, 70, 100, 255}},
		{c.Horizon + 55, 8, 40, color.NRGBA{18, 36, 50, 255}, color.NRGBA{30, 60, 60, 255}},
	}
	for _, l := range layers {
		var p vector.Path
		p.MoveTo(0, float32(c.H))
		for x := 0.0; x <= c.W; x += 8 {
			p.LineTo(float32(x), float32(l.base+math.Sin(x/l.freq)*l.amp))
		}
		p.LineTo(float32(c.W), float32(c.H))
		p.Close()
		fillPath(img, &p, l.c1)
		n := int(c.W * (c.H - l.base) / 85)
		for i := 0; i < n; i++ {
			x := rng.Float64() * c.W
			top := l.base + math.Sin(x/l.freq)*l.amp
			y := top + rng.Float64()*(c.H-top)
			slope := math.Cos(x/l.freq) * l.amp / l.freq
			nn := math.Hypot(1, slope)
			dx, dy := 1/nn, slope/nn
			col := lerpColor(l.c1, l.c2, rng.Float64())
			vector.StrokeLine(img, float32(x-dx*4), float32(y-dy*4), float32(x+dx*4), float32(y+dy*4), 2, col, true)
			yield(i)
		}
	}
}

func (c *Canvas) drawVillage(img *ebiten.Image, rng *rand.Rand) {
	wall := color.NRGBA{40, 50, 70, 255}
	roof := color.NRGBA{25, 30, 50, 255}
	window := color.NRGBA{250, 210, 90, 255}
	for _, v := range c.Villages {
		for i := 0; i < int(v[1]); i++ {
			x := v[0] + float64(i)*22 + rng.Float64()*8
			w := 14 + rng.Float64()*8
			h := 9 + rng.Float64()*7
			y := c.Horizon + 27 + rng.Float64()*14
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
	}
	cx, cy := float32(c.Church), float32(c.Horizon+3)
	vector.DrawFilledRect(img, cx, cy, 10, 34, wall, true)
	var p vector.Path
	p.MoveTo(cx-2, cy+1)
	p.LineTo(cx+5, cy-36)
	p.LineTo(cx+12, cy+1)
	p.Close()
	fillPath(img, &p, roof)
}

func (c *Canvas) cypressHalf(y float64) float64 {
	t := (c.H - y) / (c.H - 30)
	return 34 * math.Sin(math.Pi*(0.15+0.85*(1-t))) * (1 - t*0.7)
}

func (c *Canvas) drawCypress(img *ebiten.Image, rng *rand.Rand) {
	dark := color.NRGBA{14, 22, 16, 255}
	cx := c.CypressX
	var p vector.Path
	p.MoveTo(float32(cx-40), float32(c.H))
	for y := c.H; y >= 30; y -= 10 {
		p.LineTo(float32(cx-c.cypressHalf(y)+math.Sin(y/18)*6), float32(y))
	}
	for y := 30.0; y <= c.H; y += 10 {
		p.LineTo(float32(cx+c.cypressHalf(y)+math.Cos(y/22)*6), float32(y))
	}
	p.Close()
	fillPath(img, &p, dark)
	n := int(700 * c.H / 360)
	for i := 0; i < n; i++ {
		y := 40 + rng.Float64()*(c.H-40)
		half := c.cypressHalf(y) * 0.88
		x := cx + (rng.Float64()*2-1)*half
		col := lerpColor(color.NRGBA{20, 40, 26, 255}, color.NRGBA{60, 80, 40, 255}, rng.Float64())
		sway := math.Sin(y/20) * 3
		vector.StrokeLine(img, float32(x), float32(y), float32(x+sway), float32(y-10), 2, col, true)
		yield(i)
	}
}
