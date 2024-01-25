package main

import (
	"embed"

	"github.com/zegl/cosmic_unicorn"

	_ "embed"
	"image/color"
	"image/png"
	"log"
	"machine"
)

func main() {
	cu := cosmic_unicorn.CosmicUnicorn{}
	cu.Init()

	var fileIdx int
	files := []string{"mario_box.png", "polar.png", "zegl.png"}

	var pressedA = false

	handleA := func() {
		fileIdx = (fileIdx + 1) % len(files)
		drawImage(files[fileIdx], cu.SetColor)
	}

	cosmic_unicorn.SWITCH_A.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	cosmic_unicorn.SWITCH_A.SetInterrupt(machine.PinFalling, func(p machine.Pin) {
		pressedA = true
	})

	// Draw default image
	drawImage(files[fileIdx], cu.SetColor)

	for {
		if pressedA {
			pressedA = false
			handleA()
		}

		cu.Draw()
	}
}

//go:embed assets/*.png
var content embed.FS

type PixelSetter func(x, y int, c color.Color)

func drawImage(name string, ps PixelSetter) {
	file, err := content.Open("assets/" + name)
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