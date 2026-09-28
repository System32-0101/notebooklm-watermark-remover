//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// videoExts son las extensiones de entrada que se buscan en raw_video/.
var videoExts = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".avi": true, ".webm": true, ".m4v": true,
}

// exeDirectory devuelve la carpeta donde vive el ejecutable, resolviendo
// symlinks para que raw_video/ y output/ se creen junto al binario real
// incluso si se lanzó desde un acceso directo.
func exeDirectory() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return filepath.Dir(resolved), nil
}

// resolveTool busca name (p.ej. "ffmpeg.exe") primero junto al ejecutable
// (la copia incluida en build/) y solo recurre al PATH si no está ahí. Así,
// una distribución autocontenida no depende de si el usuario tiene o no
// ffmpeg instalado en el sistema.
func resolveTool(exeDir, name string) (string, error) {
	local := filepath.Join(exeDir, name)
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return local, nil
	}
	found, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("no se encontró %s (ni junto al .exe ni en el PATH)", name)
	}
	return found, nil
}

// listVideos devuelve los vídeos que están directamente dentro de dir (sin
// recursividad) cuya extensión está en videoExts.
func listVideos(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if videoExts[strings.ToLower(filepath.Ext(e.Name()))] {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}
