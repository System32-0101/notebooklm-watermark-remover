//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// badgeSupersample es el factor de sobremuestreo por eje al rasterizar el
// fondo redondeado: cada píxel de salida se comprueba en
// badgeSupersample*badgeSupersample puntos y se promedia, y eso es lo que
// convierte las esquinas en curvas suaves en vez de escalones.
const badgeSupersample = 4

// badgeBG es el color de relleno de la caja de la marca de agua. Es
// totalmente opaco (alpha 255) a propósito: se probó primero con un fondo
// semitransparente y quedaba deslavado sobre imágenes claras.
var badgeBG = color.RGBA{R: 16, G: 16, B: 16, A: 255}

// renderBadge dibuja la marca de agua como una imagen RGBA independiente:
// una caja opaca de esquinas redondeadas, del tamaño justo para el texto,
// con el texto centrado dentro.
//
// Renderizarla como imagen propia (en vez de usar el filtro drawtext de
// ffmpeg, cuya opción "box" solo dibuja rectángulos de esquina recta) es lo
// que permite tener esquinas redondeadas. Esta imagen se compone después
// sobre el vídeo con el filtro overlay de ffmpeg (ver processVideo en
// ffmpeg.go).
func renderBadge(text string, parsedFont *opentype.Font, fontSizePx, padX, padY, radius int) (image.Image, error) {
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    float64(fontSizePx),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("cargando fuente: %w", err)
	}
	defer face.Close()

	metrics := face.Metrics()
	ascent := metrics.Ascent.Round()
	descent := metrics.Descent.Round()
	textHeight := ascent + descent
	textWidth := font.MeasureString(face, text).Round()

	// La caja crece o se encoge según el texto: no hay un ancho mínimo fijo,
	// así que tanto marcas cortas como largas quedan con una caja ajustada.
	boxW := textWidth + padX*2
	boxH := textHeight + padY*2

	img := image.NewRGBA(image.Rect(0, 0, boxW, boxH))
	drawRoundedRectAA(img, boxW, boxH, radius, badgeBG)

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: face,
		// Dot es el origen del glifo (inicio de la línea base). Y se
		// desplaza por el ascent de la fuente porque Drawer posiciona el
		// texto por su línea base, no por su esquina superior izquierda.
		Dot: fixed.Point26_6{X: fixed.I(padX), Y: fixed.I(padY + ascent)},
	}
	drawer.DrawString(text)

	return img, nil
}

// drawRoundedRectAA rellena img con un rectángulo redondeado opaco,
// sobremuestreando cada píxel para que las esquinas curvas queden
// suavizadas en vez de dentadas.
func drawRoundedRectAA(img *image.RGBA, w, h, radius int, bg color.RGBA) {
	ss := badgeSupersample
	ssW, ssH, ssR := w*ss, h*ss, radius*ss
	samples := ss * ss

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			hits := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					if insideRoundedRect(x*ss+sx, y*ss+sy, ssW, ssH, ssR) {
						hits++
					}
				}
			}
			if hits == 0 {
				continue // se deja transparente para que se vea el vídeo
			}
			// Mezcla bg hacia transparente según la proporción de aciertos.
			// image.RGBA usa alpha premultiplicado, así que escalar todos
			// los canales (incluido alpha) por el mismo factor es correcto.
			coverage := float64(hits) / float64(samples)
			img.Set(x, y, color.RGBA{
				R: uint8(float64(bg.R) * coverage),
				G: uint8(float64(bg.G) * coverage),
				B: uint8(float64(bg.B) * coverage),
				A: uint8(float64(bg.A) * coverage),
			})
		}
	}
}

// insideRoundedRect indica si el punto (x, y) cae dentro de un rectángulo
// w×h con esquinas redondeadas a radio r. Solo las cuatro esquinas necesitan
// comprobar la distancia; cualquier otro punto está dentro del cuerpo
// del rectángulo sin más.
func insideRoundedRect(x, y, w, h, r int) bool {
	if r*2 > w {
		r = w / 2
	}
	if r*2 > h {
		r = h / 2
	}
	cx, cy := 0, 0
	switch {
	case x < r && y < r:
		cx, cy = r, r
	case x >= w-r && y < r:
		cx, cy = w-r-1, r
	case x < r && y >= h-r:
		cx, cy = r, h-r-1
	case x >= w-r && y >= h-r:
		cx, cy = w-r-1, h-r-1
	default:
		return true
	}
	dx, dy := float64(x-cx), float64(y-cy)
	return dx*dx+dy*dy <= float64(r*r)
}

// scaleClamp escala un valor de diseño de referencia (medido en un lienzo de
// 1920px de alto) según ratio (normalmente height/1920), redondeando al
// píxel más cercano y sin bajar nunca de min. Así la marca de agua se ve
// igual de proporcionada tanto en un vídeo de 720p como en uno de 4K.
func scaleClamp(base, ratio float64, min int) int {
	v := int(math.Round(base * ratio))
	if v < min {
		v = min
	}
	return v
}
