package main

import (
	"embed"
	"fmt"
	"math"
	"time"

	cosmic "github.com/zegl/go-cosmic-unicorn"

	_ "embed"
	"image/color"
	"image/png"
	"log"
	"machine"
)

type Effect interface {
	Init(ps PixelSetter)
	Draw(ps PixelSetter)
}

func main() {
	cu := cosmic.CosmicUnicorn{}
	cu.Init()

	var effect Effect

	var fileIdx int
	files := []string{"polar.png", "zegl.png", "mario_box.png", "gopher.png", "bread.png", "pepper.png", "pizza.png", "popcorn.png", "soda.png", "taco.png", "watermelon.png"}

	// Press A to go to the next image
	var pressedA = false
	handleA := func() {
		fileIdx = (fileIdx + 1) % len(files)
		effect = &Image{name: files[fileIdx]}
		effect.Init(cu.SetColor)
		// drawImage(files[fileIdx], cu.SetColor)
	}
	cosmic.SWITCH_A.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic.SWITCH_A.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedA = true
	})

	// Press B to draw gradient
	var pressedB = false
	handleB := func() {
		// drawGradient(cu.SetColor)
		effect = &Gradient{}
		effect.Init(cu.SetColor)
	}
	cosmic.SWITCH_B.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic.SWITCH_B.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedB = true
	})

	// Press C to draw circle
	var pressedC = false
	handleC := func() {
		effect = &Circle{}
		effect.Init(cu.SetColor)
	}
	cosmic.SWITCH_C.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic.SWITCH_C.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedC = true
	})

	// Press D to draw fire
	var pressedD = false
	handleD := func() {
		effect = &Fire{}
		effect.Init(cu.SetColor)
	}
	cosmic.SWITCH_D.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic.SWITCH_D.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedD = true
	})

	// Increase brightness
	var pressedBrightnessUp = false
	handleBrightnessUp := func() {
		cu.ChangeBrightness(10)
		// drawImage(files[fileIdx], cu.SetColor)
	}
	cosmic.SWITCH_BRIGHTNESS_UP.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic.SWITCH_BRIGHTNESS_UP.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedBrightnessUp = true
	})

	// Decrease brightness
	var pressedBrightnessDown = false
	handleBrightnessDown := func() {
		cu.ChangeBrightness(-10)
		// drawImage(files[fileIdx], cu.SetColor)
	}
	cosmic.SWITCH_BRIGHTNESS_DOWN.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic.SWITCH_BRIGHTNESS_DOWN.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedBrightnessDown = true
	})

	// Draw default image
	// drawImage(files[fileIdx], cu.SetColor)
	effect = &Fire{}
	effect.Init(cu.SetColor)
	effect.Draw(cu.SetColor)

	var t uint8
	t0 := time.Now()

	for {
		t++

		if pressedA {
			pressedA = false
			handleA()
		}

		if pressedB {
			pressedB = false
			handleB()
		}

		if pressedC {
			pressedC = false
			handleC()
		}

		if pressedD {
			pressedD = false
			handleD()
		}

		if pressedBrightnessUp {
			pressedBrightnessUp = false
			handleBrightnessUp()
		}

		if pressedBrightnessDown {
			pressedBrightnessDown = false
			handleBrightnessDown()
		}

		if t%4 == 0 {
			effect.Draw(cu.SetColor)
		}

		// Draw to screen
		cu.Draw()

		if t == 100 {
			dur := time.Since(t0)
			fmt.Printf("Rendered 100 frames in %s / %f FPS\n", dur, 100/dur.Seconds())
			t0 = time.Now()
			t = 0
		}
	}
}

//go:embed assets/*.png
var content embed.FS

type PixelSetter func(x, y int, c color.Color)

type Image struct {
	name string
}

func (g *Image) Init(ps PixelSetter) {
	file, err := content.Open("assets/" + g.name)
	if err != nil {
		log.Println(err)
		return
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Println(err)
		return
	}

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			px := img.At(x, y)
			ps(x, y, px)
		}
	}
}

func (g *Image) Draw(ps PixelSetter) {

}

type Gradient struct{}

func (g *Gradient) Init(ps PixelSetter) {
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			ps(x, y, color.RGBA{
				R: uint8(float32(x) * 255 / 32),
				G: uint8(float32(y) * 255 / 32),
			})
		}
	}
}
func (g *Gradient) Draw(ps PixelSetter) {

}

type Circle struct{}

func (g *Circle) Init(ps PixelSetter) {
	var t float64 = 0
	center := 15
	var radius float64 = 15

	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			c := float64(y*32 + x)
			dx := abs(center - x)
			dy := abs(center - y)

			dist := math.Sqrt(math.Pow(float64(dy), 2) + math.Pow(float64(dx), 2))

			if dist < radius {
				ps(x, y, color.RGBA{
					R: uint8(100 + 50*math.Cos(c)),
					G: uint8(100 + 50*math.Sin(c)),
					B: uint8(100 + 50*math.Sin(c+t)),
				})

			} else {
				ps(x, y, color.RGBA{})
			}
		}
	}
}
func (g *Circle) Draw(ps PixelSetter) {

}

func abs(i int) int {
	if i < 0 {
		return i * -1
	}
	return i
}
