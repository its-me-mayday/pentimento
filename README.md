# Pentimento

*Sotto ogni quadro c'è un altro quadro.*

Un thriller sull'arte scritto in Go con [Ebitengine](https://ebitengine.org). Sei Elena, una restauratrice romana: indaghi sulla morte della tua maestra ed entri nei dipinti per scoprire ciò che il falsario ha coperto.

**Capitolo 1 — La notte stellata:**
1. **Laboratorio**: cerca con la lampada UV il segno nascosto da Ada.
2. **Dentro il quadro** (tre schermate di larghezza, con minimappa): i pigmenti perduti sono invisibili nel grigio e si vedono solo sotto la luce della lampada, che però si scarica. Lanterne e stelle la ricaricano.
   - **Blu oltremare**: i vortici ti portano invece di respingerti. Attraversi il muro di vortice e cammini sul cielo, trascinata dalle correnti.
   - **Giallo di cromo**: sblocca il **lampo UV**, che stordisce la Macchia.
   - **La Macchia** ti insegue: la luce la rallenta e le lanterne sono zone sicure. Tre colpi e la tela si richiude.
3. **Il pentimento**: con tutti e 5 i pigmenti il quadro rivela la sua firma nascosta.

**Capitolo 2 — La chiamata** (*La vocazione di san Matteo*, Caravaggio):
1. **Laboratorio**: Corrado Valli porta il quadro. Esaminalo con **UV**, **luce radente** e **lente** (tasti 1-3), ma non farti vedere con la UV quando Corrado si volta. Poi scegli cosa dirgli.
2. **Dentro il quadro** il gioco è stealth: resta nell'ombra. Il fascio della finestra, gli sguardi dei gabellieri e la mano di Cristo ti «chiamano». La lampada (L/Spazio) rivela i pigmenti ma ti rende più visibile.

Dal titolo si può scegliere il capitolo; alla fine del Capitolo 1 si passa al 2.

Tutto è procedurale: la grafica (pennellate generate da codice) e la musica (sintetizzata in tempo reale). Non ci sono asset esterni. La colonna sonora segue il gioco: gli arpeggi si accendono quando restauri i colori e il battito accelera quando la Macchia si avvicina. La storia completa è in [`docs/STORY.md`](docs/STORY.md).

## Comandi

```bash
go mod tidy      # la prima volta
make run         # finestra desktop (su Linux servono le librerie X11/GL, vedi docs Ebitengine)
make serve       # build WebAssembly + server su http://localhost:8080
```

Controlli: **WASD/frecce** oppure tieni premuto (mouse/dito) dove vuoi andare · **Spazio / F / tasto destro** (o il pulsante LAMPO su touch) per il lampo UV · **Invio/Spazio/click** per i dialoghi · **M** musica on/off.

Per i test: `go build -tags debug` abilita il tasto **N** (pigmento o prova successiva) e, nel Capitolo 2, **R** (salta al pentimento).

## Deploy su GitHub Pages

1. Crea il repo `pentimento` e fai il push su `main`.
2. *Settings → Pages → Build and deployment → Source:* **GitHub Actions**.
3. A ogni push, la Action compila il `.wasm` e pubblica su `https://its-me-mayday.github.io/pentimento/`.

## Struttura

| File | Contenuto |
|---|---|
| `main.go` | game loop, scene, dissolvenze |
| `scene_title.go` / `scene_lab.go` / `scene_painting.go` | le tre scene |
| `starry.go` | generatore procedurale della "Notte stellata" e campo di vortici |
| `shader.go` | shader Kage: seppia ⇄ colore, vignettatura, trama della tela |
| `character.go` | Elena: animazione, lampada UV, impronte di colore |
| `music.go` | sintetizzatore e colonna sonora adattiva |
| `dialogue.go` | dialoghi con effetto macchina da scrivere |
| `web/index.html` | pagina di caricamento per il browser |
