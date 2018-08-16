package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	qrcode "github.com/xrlin/qart"
	"image"
)

func main() {
	maskImage := flag.String("m", "", "mask image path")
	outFile := flag.String("o", "", "out PNG file prefix, empty for stdout")
	pointWidth := flag.Int("s", 256, "image point width (pixel)")
	textArt := flag.Bool("t", false, "print as text-art on stdout")
	negative := flag.Bool("i", false, "invert black and white")
	startX := flag.Int("startX", 0, "mask image start point")
	startY := flag.Int("startY", 0, "mask image start point")
	width := flag.Int("width", 0, "sub image width")
	embed := flag.Bool("embed", false, "when set to true, embed the code in the mask image.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `cmd -- QR Code encoder in Go
https://github.com/xrlin/qart

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Usage:
  1. Arguments except for flags are joined by " " and used to generate QR code.
     Default output is STDOUT, pipe to imagemagick command "display" to display
     on any X server.

       cmd hello word | display

  2. Save to file if "display" not available:

       cmd "homepage: https://github.com/xrlin/qart" > out.png

`)
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		checkError(fmt.Errorf("Error: no content given"))
	}

	content := strings.Join(flag.Args(), " ")

	var err error
	var q *qrcode.HalftoneQRCode
	q, err = qrcode.NewHalftoneCode(content, qrcode.Highest)

	var maskRect image.Rectangle
	q.Embed = *embed
	if *startY >= 0 && *startX >= 0 && *width > 0 {
		maskRect = image.Rect(*startX, *startY, *startX + *width, *startY + *width)
	}
	q.MaskRectangle = maskRect

	checkError(err)

	q.MaskImagePath = *maskImage

	if *textArt {
		art := q.ToString(*negative)
		fmt.Println(art)
		return
	}

	if *negative {
		q.ForegroundColor, q.BackgroundColor = q.BackgroundColor, q.ForegroundColor
	}

	var png []byte
	png, err = q.PNG(*pointWidth)
	checkError(err)

	if *outFile == "" {
		os.Stdout.Write(png)
	} else {
		var fh *os.File
		fh, err = os.Create(*outFile + ".png")
		checkError(err)
		defer fh.Close()
		fh.Write(png)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
