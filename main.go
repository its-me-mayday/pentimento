// Pentimento — un thriller sull'arte.
// Vertical slice: titolo → laboratorio (indagine con lampada UV) → dentro "La notte stellata".
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenW = 640
	ScreenH = 360
)

// Scene è una schermata del gioco.
type Scene interface {
	Update(g *Game) error
	Draw(screen *ebiten.Image)
}

// Game gestisce scena corrente e dissolvenze tra scene.
type Game struct {
	scene     Scene
	next      Scene
	fade      float64 // 0 = nessun nero, 1 = schermo nero
	fadingOut bool
	tick      int
}

// GoTo avvia una dissolvenza verso la scena s.
func (g *Game) GoTo(s Scene) {
	if g.next != nil {
		return
	}
	g.next = s
	g.fadingOut = true
}

func (g *Game) Update() error {
	if g.tick == 0 {
		initAssets()
		g.scene = NewTitleScene()
	}
	g.tick++
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		ToggleMute()
	}

	const step = 1.0 / 40
	if g.fadingOut {
		g.fade += step
		if g.fade >= 1 {
			g.fade = 1
			g.scene = g.next
			g.next = nil
			g.fadingOut = false
		}
		return nil
	}
	if g.fade > 0 {
		g.fade -= step
		if g.fade < 0 {
			g.fade = 0
		}
	}
	return g.scene.Update(g)
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.scene == nil {
		return
	}
	g.scene.Draw(screen)
	if g.fade > 0 {
		vector.DrawFilledRect(screen, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, g.fade), false)
	}
}

func (g *Game) Layout(_, _ int) (int, int) { return ScreenW, ScreenH }

func main() {
	ebiten.SetWindowSize(ScreenW*2, ScreenH*2)
	ebiten.SetWindowTitle("Pentimento")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
