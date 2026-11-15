# Pentimento

*Sotto ogni quadro c'è un altro quadro.*

Un thriller sull'arte scritto in Go con [Ebitengine](https://ebitengine.org). Sei Elena, una restauratrice romana: indaghi sulla morte della tua maestra ed entri nei dipinti per scoprire ciò che il falsario ha coperto.

**Vertical slice attuale (Capitolo 1, La notte stellata):**
1. **Titolo**
2. **Laboratorio**: cerca con la lampada UV il segno nascosto da Ada.
3. **Dentro il quadro**: recupera 5 pigmenti, sfuggi alla Macchia e rivela il pentimento.

Tutta la grafica è procedurale (pennellate generate da codice): non ci sono asset esterni. La storia completa è in [`docs/STORY.md`](docs/STORY.md).

## Comandi

```bash
go mod tidy      # la prima volta
make run         # finestra desktop (su Linux servono le librerie X11/GL, vedi docs Ebitengine)
make serve       # build WebAssembly + server su http://localhost:8080
```

Controlli: **WASD/frecce** oppure **click/tocco** per muoversi; **Invio/Spazio/click** per avanzare i dialoghi.

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
| `dialogue.go` | dialoghi con effetto macchina da scrivere |
| `web/index.html` | pagina di caricamento per il browser |
