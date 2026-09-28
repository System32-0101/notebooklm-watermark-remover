# NotebookLM Watermark Remover

Una pequeña CLI para Windows que corta los últimos segundos de un vídeo (el
tramo donde vive la marca de agua propia de [NotebookLM](https://notebooklm.google/))
y estampa en su lugar tu propia marca de agua en la esquina inferior derecha.

No hace falta ningún editor de vídeo: metes tus vídeos exportados en una
carpeta, ejecutas el `.exe`, escribes el texto de tu marca y recibes los
archivos ya procesados.

## Funcionalidades

- **Autocontenido**: cada release incluye `ffmpeg`/`ffprobe`, no hay que
  instalar nada más.
- **Configurable en cada ejecución**: texto de la marca, una fuente
  `.ttf`/`.otf` personalizada (opcional) y cuántos segundos cortar, todo se
  pregunta de forma interactiva.
- **Caja redondeada y totalmente opaca** que se ajusta al ancho del texto
  que escribas, anclada a la esquina inferior derecha con un pequeño margen.
- **Procesado por lotes**: se pueden meter tantos vídeos como se quiera, se
  procesan todos en una sola ejecución.

## Uso

1. Descarga la última release (o compílalo tú mismo, ver más abajo) y coloca
   la carpeta donde te resulte cómodo.
2. Copia tus vídeos dentro de `raw_video/` (se crea sola en la primera
   ejecución).
3. Haz doble clic en `watermark.exe`.
4. Responde a las preguntas:
   - **Texto de la marca de agua**: el texto que se estampará en el vídeo.
   - **Ruta a una fuente personalizada**: Enter para usar la fuente por
     defecto incluida, o la ruta a tu propio `.ttf`/`.otf`.
   - **Segundos a cortar del final**: Enter para el valor por defecto (3s),
     o un número distinto.
5. Los vídeos procesados aparecen en `output/`, con el mismo nombre que el
   original.

Formatos de entrada soportados: `.mp4`, `.mov`, `.mkv`, `.avi`, `.webm`,
`.m4v`.

## Compilar desde el código fuente

Hace falta [Go 1.27+](https://go.dev/dl/).

```bash
git clone https://github.com/System32-0101/notebooklm-watermark-remover.git
cd notebooklm-watermark-remover
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/watermark.exe .
```

Ese comando solo genera `watermark.exe`. Para tener una carpeta `build/`
totalmente autocontenida como en las releases, copia también `ffmpeg.exe` y
`ffprobe.exe` dentro (ver [ffmpeg.org/download.html](https://ffmpeg.org/download.html),
cualquier build reciente para Windows sirve). No están incluidos en este
repositorio; el motivo está explicado en
[Por qué ffmpeg no está en el repo](#por-qué-ffmpeg-no-está-en-el-repo).

## Estructura del proyecto

```
main.go     punto de entrada: orquestación, flujo de preguntas, manejo de errores
prompt.go   preguntas interactivas de la CLI (texto, fuente, segundos a cortar)
fsutil.go   utilidades de filesystem (carpeta del exe, búsqueda de vídeos, herramientas)
ffmpeg.go   orquestación de ffprobe/ffmpeg: análisis, corte y composición
badge.go    renderizado de la marca de agua (rectángulo redondeado, texto, escalado)
assets/     fuente por defecto embebida
```

Cada archivo tiene una única responsabilidad a propósito: así se puede
revisar por separado la construcción del comando de ffmpeg (`ffmpeg.go`) y
el renderizado de la imagen (`badge.go`), y `main.go` queda legible como
pura orquestación.

## Personalización

Las proporciones de la marca de agua (tamaño de fuente, padding, radio de
las esquinas, margen respecto a la esquina) están definidas como constantes
al principio de `processVideo`, en [`ffmpeg.go`](ffmpeg.go), y se escalan a
la resolución del vídeo con `scaleClamp` en [`badge.go`](badge.go). Cambia
esos valores para ajustar el aspecto sin tocar la lógica de renderizado.

## Por qué ffmpeg no está en el repo

`ffmpeg.exe`/`ffprobe.exe` pesan unos 100 MB cada uno y su licencia es
(L)GPL según el build: incluirlos en git haría cada clon mucho más pesado y
añade obligaciones de licencia (atribución, ofrecer el código fuente
correspondiente) que no pintan nada en un repo de aplicación. Distribúyelos
junto a `watermark.exe` en tu propio release/zip, descargándolos de un build
oficial.

## Atribución

Este proyecto es open source, pero eso no significa sin crédito: si haces un
fork o redistribuyes el código, mantén la atribución al autor original y el
enlace al repositorio. La propia CLI muestra este aviso al terminar cada
ejecución; no lo quites en copias derivadas.

## Licencia

[MIT](LICENSE), ver el archivo LICENSE. Esto cubre solo el código de este
proyecto; ffmpeg/ffprobe mantienen su propia licencia si los redistribuyes
junto a él.

---

Creado por **Systemm32** ([DMRstudio.dev](https://dmrstudio.dev)).
Repo original: <https://github.com/System32-0101/notebooklm-watermark-remover>
