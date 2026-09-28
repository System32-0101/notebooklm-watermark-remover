//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/image/font/opentype"
)

// probeResult refleja el subconjunto de la salida de `ffprobe -of json` que
// hace falta: las dimensiones de la primera pista de vídeo y la duración.
type probeResult struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// probeVideo lee la resolución y duración de un vídeo con ffprobe. Ambas
// hacen falta antes de renderizar la marca de agua: la altura decide qué
// tamaño debe tener, y la duración cuánto hay que cortar al final.
func probeVideo(ffprobePath, input string) (width, height int, duration float64, err error) {
	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "json",
		input,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe: %w", err)
	}
	var res probeResult
	if err := json.Unmarshal(out, &res); err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe json: %w", err)
	}
	if len(res.Streams) == 0 {
		return 0, 0, 0, fmt.Errorf("no se encontró pista de vídeo")
	}
	d, err := strconv.ParseFloat(res.Format.Duration, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("duración inválida: %w", err)
	}
	return res.Streams[0].Width, res.Streams[0].Height, d, nil
}

// processVideo corta trimSeconds del final de input y quema la marca de
// agua en la esquina inferior derecha, escribiendo el resultado en output.
// Recodifica tanto vídeo como audio (en vez de copiar los streams) porque
// el filtro overlay ya obliga a decodificar cada fotograma de todos modos.
func processVideo(ffmpegPath, ffprobePath, input, output, text string, parsedFont *opentype.Font, trimSeconds float64) error {
	_, height, duration, err := probeVideo(ffprobePath, input)
	if err != nil {
		return err
	}

	newDuration := duration - trimSeconds
	if newDuration <= 0.2 {
		return fmt.Errorf("el vídeo dura %.2fs, no se puede cortar %.2fs", duration, trimSeconds)
	}

	// Las medidas parten de la marca de agua de referencia del proyecto
	// Remotion AIPatch (fontSize 32, padding 12px/24px, borderRadius 8,
	// margen exterior 10 sobre un lienzo de 1920px de alto), escaladas a la
	// altura real de este vídeo y aumentadas ~1.4x para que se lea mejor,
	// manteniendo el mismo margen relativo respecto a la esquina.
	ratio := float64(height) / 1920.0
	fontSize := scaleClamp(46, ratio, 16)
	padX := scaleClamp(34, ratio, 8)
	padY := scaleClamp(18, ratio, 5)
	radius := scaleClamp(11, ratio, 3)
	margin := scaleClamp(10, ratio, 6)

	badge, err := renderBadge(text, parsedFont, fontSize, padX, padY, radius)
	if err != nil {
		return err
	}
	badgePath, cleanup, err := writeTempPNG(badge)
	if err != nil {
		return err
	}
	defer cleanup()

	// W/H se refieren a las dimensiones del vídeo principal (input 0), w/h a
	// las de la imagen superpuesta (input 1): es la convención del filtro
	// overlay de ffmpeg. Así se ancla la marca a la esquina inferior
	// derecha dejando `margin` px de separación.
	filter := fmt.Sprintf("[0:v][1:v]overlay=W-w-%d:H-h-%d[v]", margin, margin)

	args := []string{
		"-y",
		"-i", input,
		"-i", badgePath,
		"-t", fmt.Sprintf("%.3f", newDuration),
		"-filter_complex", filter,
		"-map", "[v]",
		"-map", "0:a?", // el "?" hace el mapeo de audio opcional: algunos vídeos no tienen
		"-c:v", "libx264", "-preset", "medium", "-crf", "18",
		"-c:a", "aac", "-b:a", "192k",
		"-movflags", "+faststart",
		output,
	}

	cmd := exec.Command(ffmpegPath, args...)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, lastLines(string(combined), 15))
	}
	return nil
}

// writeTempPNG guarda img en un archivo temporal y devuelve una función para
// borrarlo después. El filtro overlay de ffmpeg necesita una ruta de
// archivo para su segunda entrada, no bytes de imagen en memoria.
func writeTempPNG(img image.Image) (string, func(), error) {
	f, err := os.CreateTemp("", "watermark-badge-*.png")
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

// lastLines devuelve como mucho las últimas n líneas de s. Sirve para que el
// error de ffmpeg sea legible en vez de volcar todo su log detallado.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
