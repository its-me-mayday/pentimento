package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// confirmPressed: spazio, invio, E, click o tocco.
func confirmPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyE) {
		return true
	}
	_, _, ok := pointerJustPressed()
	return ok
}

// pointer restituisce la posizione di tocco o mouse.
func pointer() (int, int, bool) {
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])
		return x, y, true
	}
	x, y := ebiten.CursorPosition()
	return x, y, x >= 0 && y >= 0 && x < ScreenW && y < ScreenH
}

// pointerJustPressed restituisce la posizione di un click o tocco appena avvenuto.
func pointerJustPressed() (int, int, bool) {
	if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])
		return x, y, true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return x, y, true
	}
	return 0, 0, false
}
