package main

import (
	"image/color"
	"math/rand"

	cosmic "github.com/zegl/go-cosmic-unicorn"
)

type Fire struct {
	width          int
	height         int
	heat           [][]float32
	fire_spawns    int
	damping_factor float32
}

func (f *Fire) Init(ps PixelSetter) {
	f.width = cosmic.WIDTH + 2
	f.height = cosmic.WIDTH + 4
	f.heat = make([][]float32, f.width)
	for w := 0; w < f.width; w++ {
		f.heat[w] = make([]float32, f.height)
	}
	f.fire_spawns = 5
	f.damping_factor = 0.97
}

/*def init():
  # a palette of five firey colours (white, yellow, orange, red, smoke)
  global palette
  palette = [
      graphics.create_pen(0, 0, 0),
      graphics.create_pen(20, 20, 20),
      graphics.create_pen(180, 30, 0),
      graphics.create_pen(220, 160, 0),
      graphics.create_pen(255, 255, 180)
  ]*/

func colorFromValue(value float32) color.Color {
	if value < 0.15 {
		return color.RGBA{0, 0, 0, 0}
	}
	if value < 0.25 {
		return color.RGBA{20, 20, 20, 0}
	}
	if value < 0.35 {
		return color.RGBA{180, 30, 0, 0}
	}
	if value < 0.45 {
		return color.RGBA{220, 160, 0, 0}
	}
	return color.RGBA{255, 255, 255, 0}
}

func (f *Fire) Draw(ps PixelSetter) {
	// clear the the rows off the bottom of the display
	for x := 0; x < f.width; x++ {
		f.heat[x][f.height-1] = 0
		f.heat[x][f.height-2] = 0
	}

	// add new fire spawns
	for c := 0; c < f.fire_spawns; c++ {
		x := rand.Int31n(int32(f.width)-4) + 2
		f.heat[x+0][f.height-1] = 1.0
		f.heat[x+1][f.height-1] = 1.0
		f.heat[x-1][f.height-1] = 1.0
		f.heat[x+0][f.height-2] = 1.0
		f.heat[x+1][f.height-2] = 1.0
		f.heat[x-1][f.height-2] = 1.0
	}

	// average and damp out each value to create rising flame effect
	for y := 0; y < f.height-2; y++ {
		for x := 1; x < f.width-1; x++ {
			// update this pixel by averaging the below pixels
			average := (f.heat[x][y] + f.heat[x][y+1] + f.heat[x][y+2] + f.heat[x-1][y+1] + f.heat[x+1][y+1]) / 5.0

			// damping factor to ensure flame tapers out towards the top of the displays
			average *= f.damping_factor

			// update the heat map with our newly averaged value
			f.heat[x][y] = average
		}
	}

	// render the heat values to the graphics buffer
	for y := 0; y < cosmic.HEIGHT; y++ {
		for x := 0; x < cosmic.WIDTH; x++ {
			ps(x, y, colorFromValue(f.heat[x+1][y]))
		}
	}
}
