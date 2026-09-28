//go:build windows

package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Embebida para que el ejecutable no dependa de archivos de fuente externos.
// Así, si se comparte solo el exe (más ffmpeg/ffprobe), sigue funcionando.
//
//go:embed assets/RedditMono-Bold.ttf
var defaultFontBytes []byte

const defaultTrimSeconds = 3.0

// waitEnter espera a que el usuario pulse Enter, para que si el programa se
// abrió con doble clic la consola no se cierre sola antes de leer el resumen.
func waitEnter(reader *bufio.Reader) {
	fmt.Println("Pulsa Enter para salir...")
	reader.ReadString('\n')
}

// promptRequired repite la pregunta hasta recibir una respuesta no vacía.
func promptRequired(reader *bufio.Reader, label string) string {
	for {
		fmt.Print(label)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
		fmt.Println("  (no puede estar vacío)")
	}
}

// promptFont pregunta por un archivo .ttf/.otf personalizado, opcional. Si la
// respuesta está vacía o la ruta no es válida, usa la fuente por defecto
// embebida, que se vuelca a un archivo de caché junto al exe la primera vez
// que hace falta (el renderizador de fuentes necesita una ruta real, no
// bytes en memoria).
func promptFont(reader *bufio.Reader, exeDir string) string {
	fmt.Print("Ruta a una fuente personalizada .ttf/.otf (Enter = usar la fuente por defecto): ")
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.Trim(line, `"`))

	if line != "" {
		if info, err := os.Stat(line); err == nil && !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(line))
			if ext == ".ttf" || ext == ".otf" {
				return line
			}
		}
		fmt.Println("  Fuente no válida o no encontrada, se usará la fuente por defecto.")
	}

	defaultFontPath := filepath.Join(exeDir, ".default-font.ttf")
	if _, err := os.Stat(defaultFontPath); err != nil {
		_ = os.WriteFile(defaultFontPath, defaultFontBytes, 0o644)
	}
	return defaultFontPath
}

// promptTrim pregunta cuántos segundos cortar del final de cada vídeo,
// aceptando "." o "," como separador decimal. Si la respuesta está vacía o
// no es válida, usa defaultTrimSeconds.
func promptTrim(reader *bufio.Reader) float64 {
	fmt.Printf("Segundos a cortar del final (Enter = %.0f): ", defaultTrimSeconds)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultTrimSeconds
	}
	v, err := strconv.ParseFloat(strings.Replace(line, ",", ".", 1), 64)
	if err != nil || v < 0 {
		fmt.Println("  Valor no válido, se usará el valor por defecto.")
		return defaultTrimSeconds
	}
	return v
}
