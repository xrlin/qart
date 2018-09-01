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
	outFile := flag.String("o", "", "out PNG/GIF file prefix")
	pointWidth := flag.Int("pw", 3, "image point width (module)")
	textArt := flag.Bool("t", false, "print as pure text-art on stdout")
	startX := flag.Int("startX", 0, "mask image start point")
	startY := flag.Int("startY", 0, "mask image start point")
	width := flag.Int("width", 0, "sub image width")
	embed := flag.Bool("embed", false, "when set to true, over the code in the source image.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Qart -- Generate charming QR Code encoder in Go
https://github.com/xrlin/qart

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Usage:
  1. Generate normal qr code to stdout
   qart http://excample.com
  2. Generate code with image to file.
   qart -m test.png http://example.com

Tips:
  1. Arguments except for flags are joined by " " and used to generate QR code.
     Default output is STDOUT. You can set the option to save to file.
  2. To generate QR code with mask image(jpg/png/gif) must specify the output file.
`)
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		checkError(fmt.Errorf("error: no content given"))
	}

	content := strings.Join(flag.Args(), " ")

	var err error
	var q *qrcode.HalftoneQRCode
	q, err = qrcode.NewHalftoneCode(content, qrcode.Highest)

	var maskRect image.Rectangle
	if *startY >= 0 && *startX >= 0 && *width > 0 {
		maskRect = image.Rect(*startX, *startY, *startX + *width, *startY + *width)
	}

	checkError(err)

	q.AddOption(qrcode.Option{Embed: *embed, MaskImagePath: *maskImage, MaskRectangle: maskRect})

	//var png []byte
	imgBytes, err := q.ImageData(*pointWidth)
	checkError(err)

	if *outFile != "" {
		f, _ := os.OpenFile(*outFile, os.O_RDWR|os.O_CREATE, os.ModePerm)
		f.Write(imgBytes)
		f.Close()
	}

	if *outFile == "" || *textArt {
		fmt.Println(q.ToString())
	}

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
