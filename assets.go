package main

import (
	"bytes"
	"image"
	"image/color"
	"log"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	faceText   *text.GoTextFace
	faceSmall  *text.GoTextFace
	faceBold   *text.GoTextFace
	faceItalic *text.GoTextFace
	faceTitle  *text.GoTextFace

	whiteSub *ebiten.Image

	// paintingBG è "La notte stellata" generata proceduralmente a pennellate.
	paintingBG *ebiten.Image

	gradeShader *ebiten.Shader
)

func initAssets() {
	reg := mustFace(goregular.TTF)
	bold := mustFace(gobold.TTF)
	ita := mustFace(goitalic.TTF)
	faceText = &text.GoTextFace{Source: reg, Size: 15}
	faceSmall = &text.GoTextFace{Source: reg, Size: 12}
	faceBold = &text.GoTextFace{Source: bold, Size: 14}
	faceItalic = &text.GoTextFace{Source: ita, Size: 15}
	faceTitle = &text.GoTextFace{Source: bold, Size: 58}

	w := ebiten.NewImage(3, 3)
	w.Fill(color.White)
	whiteSub = w.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	sh, err := ebiten.NewShader([]byte(gradeShaderSrc))
	if err != nil {
		log.Fatal(err)
	}
	gradeShader = sh

	paintingBG = smallCanvas.Generate()
	initMusic()
}

func mustFace(ttf []byte) *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
	if err != nil {
		log.Fatal(err)
	}
	return s
}

// rgba costruisce un colore da componenti 0–255 e alpha 0–1.
func rgba(r, g, b uint8, a float64) color.NRGBA {
	return color.NRGBA{r, g, b, uint8(clamp(a, 0, 1) * 255)}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		uint8(lerp(float64(a.R), float64(b.R), t)),
		uint8(lerp(float64(a.G), float64(b.G), t)),
		uint8(lerp(float64(a.B), float64(b.B), t)),
		uint8(lerp(float64(a.A), float64(b.A), t)),
	}
}

// fillPath riempie un vector.Path con un colore pieno.
func fillPath(dst *ebiten.Image, p *vector.Path, c color.NRGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR = float32(c.R) / 255
		vs[i].ColorG = float32(c.G) / 255
		vs[i].ColorB = float32(c.B) / 255
		vs[i].ColorA = float32(c.A) / 255
	}
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{
		FillRule:  ebiten.FillRuleNonZero,
		AntiAlias: true,
	})
}

// drawText disegna testo; align: -1 sinistra, 0 centro, 1 destra.
func drawText(dst *ebiten.Image, s string, face text.Face, x, y float64, c color.NRGBA, align int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = face.Metrics().HAscent + face.Metrics().HDescent + 4
	switch align {
	case 0:
		op.PrimaryAlign = text.AlignCenter
	case 1:
		op.PrimaryAlign = text.AlignEnd
	}
	text.Draw(dst, s, face, op)
}

// wrap spezza s in righe che stanno in maxW pixel.
func wrap(s string, face text.Face, maxW float64) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if tw, _ := text.Measure(try, face, 0); tw > maxW && cur != "" {
			lines = append(lines, cur)
			cur = w
		} else {
			cur = try
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// applyGrade disegna src su dst con il colour grading del gioco:
// amount 0 = seppia spenta, 1 = colori pieni.
func applyGrade(dst, src *ebiten.Image, amount, t float64, dx, dy float64) {
	applyGradeLamp(dst, src, amount, t, dx, dy, 0, 0, 0)
}

// applyGradeLamp: come applyGrade, ma la lampada UV in (lx, ly) ridà colore nel raggio lr.
func applyGradeLamp(dst, src *ebiten.Image, amount, t, dx, dy, lx, ly, lr float64) {
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(dx, dy)
	op.Images[0] = src
	op.Uniforms = map[string]any{
		"Amount": float32(amount),
		"Time":   float32(t),
		"Lamp":   []float32{float32(lx), float32(ly), float32(lr)},
	}
	b := src.Bounds()
	dst.DrawRectShader(b.Dx(), b.Dy(), gradeShader, op)
}
