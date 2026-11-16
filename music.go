package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Colonna sonora procedurale: niente file audio, tutto sintetizzato in tempo reale.
// Progressione in re minore (i–VI–iv–V), pad che "respirano", arpeggi che si
// accendono con il restauro, battito cardiaco quando la Macchia è vicina.

const sampleRate = 44100

// Mood descrive lo stato emotivo che la musica deve seguire.
type Mood struct {
	Pad     float64 // pad e basso
	Arp     float64 // arpeggio pizzicato
	Bright  float64 // celesta e filtro più aperto (colori restaurati)
	Tension float64 // battito + archi dissonanti (pericolo)
}

var (
	music       *Synth
	musicPlayer *audio.Player
)

func initMusic() {
	ctx := audio.NewContext(sampleRate)
	music = NewSynth()
	p, err := ctx.NewPlayerF32(music)
	if err != nil {
		return // il gioco funziona anche senza audio
	}
	musicPlayer = p
	musicPlayer.SetVolume(0.8)
	musicPlayer.Play()
}

// SetMood imposta l'umore da raggiungere (con transizione morbida).
func SetMood(m Mood) {
	if music != nil {
		music.mu.Lock()
		music.target = m
		music.mu.Unlock()
	}
}

// PlaySFX: "chime" (pigmento), "hit" (Macchia), "reveal" (pentimento), "uv" (segno trovato).
func PlaySFX(name string) {
	if music != nil {
		music.mu.Lock()
		music.queue = append(music.queue, name)
		music.mu.Unlock()
	}
}

func ToggleMute() bool {
	if musicPlayer == nil {
		return true
	}
	if musicPlayer.Volume() > 0 {
		musicPlayer.SetVolume(0)
		return true
	}
	musicPlayer.SetVolume(0.8)
	return false
}

type shot struct {
	kind  int // 0 campana, 1 tonfo, 2 sweep
	freq  float64
	start int // campioni di attesa
	t     int
	amp   float64
	ph    float64
	ph2   float64
}

type Synth struct {
	mu     sync.Mutex
	target Mood
	queue  []string

	cur    Mood
	n      int64
	rng    *rand.Rand
	padPh  [8]float64
	lpL    float64
	bassP  float64
	arpP   float64
	arpEnv float64
	arpN   int64
	heart  float64
	trem   [2]float64
	shots  []shot

	dl1, dl2 []float64
	i1, i2   int
}

func NewSynth() *Synth {
	return &Synth{
		rng: rand.New(rand.NewSource(66)),
		dl1: make([]float64, int(0.31*sampleRate)),
		dl2: make([]float64, int(0.43*sampleRate)),
		cur: Mood{Pad: 1},
	}
}

var chords = [][4]int{
	{50, 57, 62, 65}, // Re minore
	{46, 53, 58, 62}, // Si bemolle
	{43, 50, 55, 58}, // Sol minore
	{45, 52, 57, 61}, // La (dominante)
}

var arpDecay = math.Exp(-5.0 / sampleRate)

var arpPattern = []int{0, 1, 2, 3, 2, 1, 3, 2}

var midiFreq = func() (t [128]float64) {
	for i := range t {
		t[i] = 440 * math.Pow(2, (float64(i)-69)/12)
	}
	return
}()

func mtof(n float64) float64 { return midiFreq[int(n)&127] }

// softClip: approssimazione veloce di tanh.
func softClip(x float64) float64 {
	if x > 3 {
		return 1
	} else if x < -3 {
		return -1
	}
	return x * (27 + x*x) / (27 + 9*x*x)
}

func tri(ph float64) float64 { return 4*math.Abs(ph-math.Floor(ph)-0.5) - 1 }

const (
	bpm       = 66.0
	beatLen   = sampleRate * 60 / bpm
	eighthLen = beatLen / 2
	chordLen  = beatLen * 8
)

func (s *Synth) Read(p []byte) (int, error) {
	s.mu.Lock()
	target := s.target
	q := s.queue
	s.queue = nil
	s.mu.Unlock()
	for _, name := range q {
		s.trigger(name)
	}

	frames := len(p) / 8
	for i := 0; i < frames; i++ {
		if s.n%64 == 0 {
			k := 64.0 / (sampleRate * 0.8) // ~0.8 s di transizione
			s.cur.Pad += (target.Pad - s.cur.Pad) * k
			s.cur.Arp += (target.Arp - s.cur.Arp) * k
			s.cur.Bright += (target.Bright - s.cur.Bright) * k
			s.cur.Tension += (target.Tension - s.cur.Tension) * k * 2
		}
		l, r := s.sample()
		binary.LittleEndian.PutUint32(p[i*8:], math.Float32bits(float32(l)))
		binary.LittleEndian.PutUint32(p[i*8+4:], math.Float32bits(float32(r)))
		s.n++
	}
	return frames * 8, nil
}

func (s *Synth) sample() (float64, float64) {
	m := s.cur
	n := float64(s.n)
	ci := int(n/chordLen) % len(chords)
	ch := chords[ci]
	tc := math.Mod(n, chordLen) / chordLen

	// pad: due oscillatori triangolari leggermente scordati per nota, inviluppo che respira
	env := math.Sqrt(math.Sin(math.Pi * tc))
	pad := 0.0
	for j, note := range ch {
		f := mtof(float64(note))
		s.padPh[j*2] += f / sampleRate
		s.padPh[j*2+1] += f * 1.004 / sampleRate
		pad += tri(s.padPh[j*2]) + tri(s.padPh[j*2+1])
	}
	pad *= env * 0.045 * m.Pad
	a := 0.025 + 0.08*m.Bright
	s.lpL += (pad - s.lpL) * a
	pad = s.lpL

	// basso
	s.bassP += mtof(float64(ch[0]-12)) / sampleRate
	bass := (math.Sin(2*math.Pi*s.bassP) + 0.25*math.Sin(4*math.Pi*s.bassP)) * 0.13 * m.Pad * (0.6 + 0.4*env)

	// arpeggio pizzicato a crome
	k := int64(n / eighthLen)
	if k != s.arpN {
		s.arpN = k
		s.arpP = 0
		s.arpEnv = 1
		// celesta: campanelle acute quando i colori tornano
		if m.Bright > 0.15 && s.rng.Float64() < m.Bright*0.35 {
			s.shots = append(s.shots, shot{kind: 0, freq: mtof(float64(ch[s.rng.Intn(4)] + 24)), amp: 0.05 * m.Bright})
		}
	}
	s.arpEnv *= arpDecay
	note := ch[arpPattern[k%int64(len(arpPattern))]] + 12
	f := mtof(float64(note))
	s.arpP += f / sampleRate
	arp := (math.Sin(2*math.Pi*s.arpP) + 0.35*math.Sin(4*math.Pi*s.arpP)) * s.arpEnv * 0.09 * m.Arp

	// tensione: battito "lub-dub" e archi in seconda minore con tremolo
	tension := 0.0
	if m.Tension > 0.01 {
		period := (1.25 - 0.6*m.Tension) * sampleRate
		tb := math.Mod(n, period)
		for _, off := range []float64{0, 0.22 * sampleRate} {
			t := (tb - off) / sampleRate
			if t >= 0 && t < 0.25 {
				freq := 45 + 50*math.Exp(-t*25)
				tension += math.Sin(2*math.Pi*freq*t) * math.Exp(-t*14) * 0.45
			}
		}
		trem := 0.5 + 0.5*math.Sin(2*math.Pi*7*n/sampleRate)
		s.trem[0] += mtof(76) / sampleRate
		s.trem[1] += mtof(77) / sampleRate
		tension += (tri(s.trem[0]) + tri(s.trem[1])) * 0.035 * trem
		tension *= m.Tension
	}

	// effetti one-shot
	fx := 0.0
	alive := s.shots[:0]
	for _, sh := range s.shots {
		if sh.start > 0 {
			sh.start--
			alive = append(alive, sh)
			continue
		}
		t := float64(sh.t) / sampleRate
		switch sh.kind {
		case 0: // campana: parziale inarmonica
			sh.ph += sh.freq / sampleRate
			sh.ph2 += sh.freq * 2.76 / sampleRate
			fx += (math.Sin(2*math.Pi*sh.ph) + 0.3*math.Sin(2*math.Pi*sh.ph2)) * math.Exp(-t*2.2) * sh.amp
			if t > 2.5 {
				continue
			}
		case 1: // tonfo d'inchiostro
			freq := 35 + 80*math.Exp(-t*12)
			sh.ph += freq / sampleRate
			fx += (math.Sin(2*math.Pi*sh.ph)*0.8 + (s.rng.Float64()*2-1)*0.3*math.Exp(-t*30)) * math.Exp(-t*6) * sh.amp
			if t > 0.8 {
				continue
			}
		case 2: // sweep luminoso
			freq := sh.freq * (1 + t*1.5)
			if sh.freq > 400 { // sweep breve (lampo)
				freq = sh.freq * (1 + t*6)
				if t > 0.6 {
					continue
				}
			}
			sh.ph += freq / sampleRate
			fx += tri(sh.ph) * math.Sin(math.Pi*math.Min(t/3, 1)) * sh.amp
			if t > 3 {
				continue
			}
		}
		sh.t++
		alive = append(alive, sh)
	}
	s.shots = alive

	dry := pad + bass + arp + tension + fx

	// riverbero: due linee di ritardo incrociate
	d1, d2 := s.dl1[s.i1], s.dl2[s.i2]
	s.dl1[s.i1] = dry + d2*0.42
	s.dl2[s.i2] = dry + d1*0.42
	s.i1 = (s.i1 + 1) % len(s.dl1)
	s.i2 = (s.i2 + 1) % len(s.dl2)

	l := softClip((dry + d1*0.35) * 1.2)
	r := softClip((dry + d2*0.35) * 1.2)
	return l, r
}

func (s *Synth) trigger(name string) {
	ch := chords[int(float64(s.n)/chordLen)%len(chords)]
	switch name {
	case "chime":
		for i, nt := range []int{ch[0] + 24, ch[1] + 24, ch[2] + 24, ch[3] + 24} {
			s.shots = append(s.shots, shot{kind: 0, freq: mtof(float64(nt)), amp: 0.11, start: i * sampleRate / 9})
		}
	case "uv":
		s.shots = append(s.shots, shot{kind: 2, freq: mtof(62), amp: 0.06})
		s.shots = append(s.shots, shot{kind: 0, freq: mtof(81), amp: 0.08, start: sampleRate / 2})
	case "flash":
		s.shots = append(s.shots, shot{kind: 2, freq: mtof(69), amp: 0.09})
		s.shots = append(s.shots, shot{kind: 0, freq: mtof(93), amp: 0.07})
		s.shots = append(s.shots, shot{kind: 0, freq: mtof(98), amp: 0.05, start: sampleRate / 12})
	case "hit":
		s.shots = append(s.shots, shot{kind: 1, amp: 0.7})
	case "reveal":
		s.shots = append(s.shots, shot{kind: 2, freq: mtof(50), amp: 0.08})
		for i, nt := range []int{74, 77, 81, 86, 89} {
			s.shots = append(s.shots, shot{kind: 0, freq: mtof(float64(nt)), amp: 0.09, start: sampleRate/2 + i*sampleRate/6})
		}
	}
}
