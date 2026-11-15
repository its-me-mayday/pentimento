package main

// gradeShaderSrc: desaturazione verso la seppia, vignettatura e grana della tela.
const gradeShaderSrc = `//kage:unit pixels
package main

var Amount float
var Time float

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0At(srcPos)
	g := dot(c.rgb, vec3(0.299, 0.587, 0.114))
	sepia := vec3(g*1.07, g*0.95, g*0.78)
	rgb := mix(sepia, c.rgb, clamp(Amount, 0.0, 1.0))

	uv := (srcPos - imageSrc0Origin()) / imageSrc0Size()
	d := distance(uv, vec2(0.5, 0.5))
	rgb *= 1.0 - smoothstep(0.42, 0.85, d)*0.65

	// trama della tela + grana leggera
	weave := sin(srcPos.x*2.1) * sin(srcPos.y*2.1) * 0.025
	grain := (hash(srcPos + vec2(Time, Time*1.7)) - 0.5) * 0.05
	rgb += weave + grain

	return vec4(rgb*c.a, c.a)
}
`
