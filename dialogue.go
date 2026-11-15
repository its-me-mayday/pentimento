package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Line è una battuta di dialogo.
type Line struct {
	Speaker string
	Text    string
	Echo    bool // voce di Ada: corsivo e colore freddo
}

// Dialogue mostra battute con effetto macchina da scrivere.
type Dialogue struct {
	lines   []Line
	idx     int
	shown   float64
	wrapped []string
	total   int
	tick    int
}

func (d *Dialogue) Start(lines ...Line) {
	d.lines = lines
	d.idx = 0
	d.prepare()
}

func (d *Dialogue) Active() bool { return d.idx < len(d.lines) }

func (d *Dialogue) face() text.Face {
	if d.lines[d.idx].Echo {
		return faceItalic
	}
	return faceText
}

func (d *Dialogue) prepare() {
	d.shown = 0
	if !d.Active() {
		return
	}
	d.wrapped = wrap(d.lines[d.idx].Text, d.face(), ScreenW-90)
	d.total = 0
	for _, l := range d.wrapped {
		d.total += len([]rune(l))
	}
}

// Update avanza il dialogo; restituisce true se ha consumato l'input.
func (d *Dialogue) Update() bool {
	if !d.Active() {
		return false
	}
	d.tick++
	d.shown += 1.2
	if confirmPressed() {
		if int(d.shown) < d.total {
			d.shown = float64(d.total)
		} else {
			d.idx++
			d.prepare()
		}
	}
	return true
}

func (d *Dialogue) Draw(dst *ebiten.Image) {
	if !d.Active() {
		return
	}
	l := d.lines[d.idx]
	const x, y, w, h = 24, 262, ScreenW - 48, 86
	vector.DrawFilledRect(dst, x, y, w, h, rgba(10, 8, 14, 0.86), false)
	border := rgba(196, 160, 90, 0.9)
	if l.Echo {
		border = rgba(140, 200, 255, 0.9)
	}
	vector.StrokeRect(dst, x+0.5, y+0.5, w-1, h-1, 1, border, false)

	if l.Speaker != "" {
		drawText(dst, l.Speaker, faceBold, x+18, y+10, border, -1)
	}
	tc := rgba(236, 228, 214, 1)
	if l.Echo {
		tc = rgba(200, 225, 255, 1)
	}
	left := int(d.shown)
	ly := float64(y + 32)
	for _, line := range d.wrapped {
		r := []rune(line)
		n := min(left, len(r))
		if n > 0 {
			drawText(dst, string(r[:n]), d.face(), x+18, ly, tc, -1)
		}
		left -= len(r)
		ly += 19
	}
	if int(d.shown) >= d.total && (d.tick/20)%2 == 0 {
		vector.DrawFilledCircle(dst, x+w-16, y+h-14, 3, border, true)
	}
}
