//go:build windows

// notebooklm-watermark-remover corta los últimos N segundos de un vídeo
// (el tramo donde vive la marca de agua propia de NotebookLM) y estampa en
// su lugar una marca de agua personalizada en la esquina inferior derecha.
//
// Es una CLI autocontenida: se meten vídeos en raw_video/, se ejecuta el
// exe y se responden tres preguntas (texto, fuente opcional, segundos a
// cortar). Los archivos procesados quedan en output/ con el mismo nombre.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/image/font/opentype"
)

// Handles para cambiar la codepage de la consola a UTF-8 al arrancar.
// Por defecto la consola de Windows usa una codepage antigua que rompe las
// tildes y la ñ escritas por el usuario, y eso acabaría quemado en la marca de agua.
var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP       = kernel32.NewProc("SetConsoleCP")
)

func init() {
	const utf8CodePage = 65001
	procSetConsoleOutputCP.Call(uintptr(utf8CodePage))
	procSetConsoleCP.Call(uintptr(utf8CodePage))
}

// repoURL y author identifican el proyecto original. Se muestran siempre en
// el footer de la CLI: aunque el código sea open source, la atribución al
// autor y al repo no debe quitarse en copias o forks.
const (
	repoURL = "https://github.com/System32-0101/notebooklm-watermark-remover"
	author  = "Systemm32 (DMRstudio.dev)"
)

// printFooter muestra la atribución del proyecto. Se llama antes de cada
// salida del programa para que siempre quede visible en la consola.
func printFooter() {
	fmt.Println()
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Repo original: %s\n", repoURL)
	fmt.Printf("Creado por %s\n", author)
	fmt.Println("------------------------------------------------------------")
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	exeDir, err := exeDirectory()
	fatalIf(err)

	rawDir := filepath.Join(exeDir, "raw_video")
	outDir := filepath.Join(exeDir, "output")
	fatalIf(os.MkdirAll(rawDir, 0o755))
	fatalIf(os.MkdirAll(outDir, 0o755))

	fmt.Println("=== NotebookLM Watermark Remover ===")
	fmt.Printf("Carpeta de entrada: %s\n", rawDir)
	fmt.Printf("Carpeta de salida:  %s\n\n", outDir)

	files, err := listVideos(rawDir)
	fatalIf(err)

	if len(files) == 0 {
		fmt.Println(`No hay vídeos en raw_video\. Copia ahí los vídeos que quieras marcar y vuelve a ejecutar este programa.`)
		printFooter()
		waitEnter(reader)
		return
	}

	fmt.Printf("Vídeos encontrados: %d\n\n", len(files))

	text := promptRequired(reader, "Texto de la marca de agua: ")
	fontPath := promptFont(reader, exeDir)
	trim := promptTrim(reader)

	ffmpegPath, err := resolveTool(exeDir, "ffmpeg.exe")
	fatalIf(err)
	ffprobePath, err := resolveTool(exeDir, "ffprobe.exe")
	fatalIf(err)

	fontBytes, err := os.ReadFile(fontPath)
	fatalIf(err)
	parsedFont, err := opentype.Parse(fontBytes)
	fatalIf(err)

	fmt.Println()
	okCount := 0
	for _, f := range files {
		name := filepath.Base(f)
		fmt.Printf("-> %s ... ", name)
		outPath := filepath.Join(outDir, name)
		if err := processVideo(ffmpegPath, ffprobePath, f, outPath, text, parsedFont, trim); err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}
		fmt.Println("OK")
		okCount++
	}

	fmt.Printf("\nListo: %d/%d vídeos procesados. Revisa la carpeta output\\.\n", okCount, len(files))
	printFooter()
	waitEnter(reader)
}

// fatalIf imprime el error, espera a que el usuario lo lea y sale.
// Se usa para fallos de arranque de los que el programa no puede recuperarse:
// si se abrió con doble clic, la consola se cerraría sola y ocultaría el error.
func fatalIf(err error) {
	if err != nil {
		fmt.Printf("Error fatal: %v\n", err)
		printFooter()
		fmt.Println("Pulsa Enter para salir...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}
}
