package qart

import (
	"bytes"
	"errors"
	"github.com/disintegration/imaging"
	"github.com/xrlin/qart/bitset"
	"github.com/xrlin/qart/reedsolomon"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"image/gif"
	"image/draw"
	"io"
	"reflect"
	"io/ioutil"
)

// HalftoneQRCode specify the basic qrcode and another options to generate a qrcode with image
type HalftoneQRCode struct {
	// Original content encoded.
	Content string

	// QR Code type.
	Level         RecoveryLevel

	// QR Code version number
	VersionNumber int

	encoder *dataEncoder
	version qrCodeVersion

	data   *bitset.Bitset
	symbol *HalftoneSymbol

	mask int

	option *Option
}

// Option struct contains the option to build code
type Option struct {
	// User settable drawing options.
	ForegroundColor color.Color
	BackgroundColor color.Color

	MaskImagePath string
	MaskImageFile io.Reader

	MaskRectangle image.Rectangle

	Embed bool
}

// OptionKey act as the key of Option struct
type OptionKey string

const (
	// Field name of ForegroundColor in Option
	ForegroundColorOpt OptionKey = "ForegroundColor"
	// Field name of BackgroundColor in Option
	BackgroundColorOpt OptionKey = "BackgroundColor"
	// Field name of MaskImagePath in Option
	MaskImagePathOpt   OptionKey = "MaskImagePath"
	// Field name of MaskImageFile in Option
	MaskImageFileOpt   OptionKey = "MaskImageFile"
	// Field name of MaskRectangle in Option
	MaskRectangleOpt   OptionKey = "MaskRectangle"
	// Field name of Embed in Option
	EmbedOpt           OptionKey = "Embed"
)

// AddOption add Option to a HalftoneQRCode.
// Attention: only the none-zero field will be merged into HalftoneQRCode's option.
// If you want to delete an option, use RemoveOption method.
func (q *HalftoneQRCode) AddOption(cfg Option) *HalftoneQRCode {
	preValue := reflect.ValueOf(q.option).Elem()
	values := reflect.ValueOf(cfg)
	for i := 0; i < values.NumField(); i++ {
		if values.Field(i).Interface() != reflect.Zero(values.Field(i).Type()).Interface() {
			preValue.Field(i).Set(values.Field(i))
		}
	}
	return q
}

// Option method returns the pointer point to option.
func (q HalftoneQRCode) Option() *Option {
	return q.option
}

// RemoveOption method set the option field's value to its zero value.
func (q *HalftoneQRCode) RemoveOption(opt OptionKey) *HalftoneQRCode {
	preValue := reflect.ValueOf(q.option).Elem()
	field := preValue.FieldByName(string(opt))
	field.Set(reflect.Zero(field.Type()))
	return q
}

// getMaskImageFile method try to get the image file used to generate code.
// you should check return value before using.
func (q *HalftoneQRCode) getMaskImageFile() (f io.Reader, err error) {
	if q.option.MaskImageFile == nil && q.option.MaskImagePath == "" {
		return nil, nil
	}
	if q.option.MaskImageFile != nil {
		b, _ := ioutil.ReadAll(q.option.MaskImageFile)
		q.AddOption(Option{MaskImageFile: bytes.NewBuffer(b)})
		f = bytes.NewBuffer(b)
		return
	}
	f, err = os.Open(q.option.MaskImagePath)
	return
}

// ImageData generate code and return the bytes represents the cod image(png/gif).
// pointWidth parameter set the width of a qr code module.
func (q *HalftoneQRCode) ImageData(pointWidth int) (ret []byte, err error) {
	fileObj, err := q.getMaskImageFile()
	if err != nil {
		return
	}

	var format string
	if fileObj != nil {
		_, format, _ = image.DecodeConfig(fileObj)
	}
	var buf bytes.Buffer
	if format == "gif" {
		var gifCode *gif.GIF
		gifCode, err = q.CodeGif(pointWidth)
		if err != nil {
			return
		}
		err = gif.EncodeAll(&buf, gifCode)
		ret = buf.Bytes()
		return
	}
	imgCode, err := q.CodeImage(pointWidth)
	if err != nil {
		return
	}
	encoder := png.Encoder{CompressionLevel: png.BestCompression}

	err = encoder.Encode(&buf, imgCode)
	ret = buf.Bytes()
	return
}

func (q *HalftoneQRCode) readAsGif(f io.Reader) (maskGif *gif.GIF, err error) {
	maskGif, err = gif.DecodeAll(f)
	return
}

func (q *HalftoneQRCode) readAsImage(f io.Reader) (maskImage image.Image, err error) {
	maskImage, _, err = image.Decode(f)
	return
}

// CodeImage generate the code as a normal image.
// pointWidth parameter set the width of a qr code module.
func (q *HalftoneQRCode) CodeImage(pointWidth int) (ret image.Image, err error) {
	fileObj, err := q.getMaskImageFile()
	if err != nil {
		return
	}

	var srcImg image.Image
	if fileObj != nil {
		srcImg, err = q.readAsImage(fileObj)
		if err != nil {
			return
		}
	}

	ret, err = q.drawCodeWithImage(pointWidth, srcImg)
	return
}

// CodeGif generates the code as a gif.
// pointWidth parameter set the width of a qr code module.
func (q *HalftoneQRCode) CodeGif(pointWidth int) (ret *gif.GIF, err error) {
	fileObj, err := q.getMaskImageFile()
	if err != nil {
		return
	}

	maskGif, err := q.readAsGif(fileObj)
	if err != nil {
		return
	}

	for idx, img := range maskGif.Image {
		img1, err := q.drawCodeWithImage(pointWidth, img)
		if err != nil {
			return ret, err
		}
		palettedImage := image.NewPaletted(img1.Bounds(), img.Palette)
		draw.Draw(palettedImage, palettedImage.Rect, img1, img1.Bounds().Min, draw.Over)
		maskGif.Image[idx] = palettedImage
		maskGif.Config.Height = img1.Bounds().Size().Y
		maskGif.Config.Width = img1.Bounds().Size().X
	}
	ret = maskGif
	return
}

// getMaskAreaImage analyzes the options and construct the proper mask image combined with qr code
func (q *HalftoneQRCode) getMaskAreaImage(sourceImage image.Image, bounds image.Rectangle) (maskImage image.Image, err error) {
	if sourceImage == nil {
		return nil, nil
	}
	maskImage = imaging.Clone(sourceImage)
	if q.option.Embed && !q.option.MaskRectangle.In(maskImage.Bounds()) {
		err = errors.New("when in embed mode, mask image must not smaller than the cmd")
		return
	}
	if !q.option.MaskRectangle.Empty() {
		if !q.option.MaskRectangle.In(maskImage.Bounds()) {
			err = errors.New("sub mask image area must in the mask image")
			return
		}
		maskImage = imaging.Crop(maskImage, q.option.MaskRectangle)
	}
	maskImage = imaging.Resize(maskImage, bounds.Max.X, bounds.Max.Y, imaging.Lanczos)
	return
}

func (q *HalftoneQRCode) drawCodeWithImage(pointWidth int, sourceImage image.Image) (image.Image, error) {
	// Minimum pixels (both width and height) required.
	realSize := q.symbol.size

	var size int
	// Variable size support.
	if pointWidth < 0 {
		pointWidth = 1
	}
	pointWidth = 3
	size = 3 * pointWidth * realSize

	// Size of each module drawn.
	widthPerModule := size / realSize
	maskBlockWidth := pointWidth
	offset := 0

	// Init image
	rect := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{size, size}}
	img := image.NewRGBA(rect)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			img.Set(i, j, q.option.BackgroundColor)
		}
	}

	maskAreaImage, err := q.getMaskAreaImage(sourceImage, rect)
	if err != nil {
		return nil, err
	}

	bitmap := q.symbol.bitmap()

	// Start draw each pixel
	for y, row := range bitmap {
		for x, v := range row {
			// calculate the start position of every module in image
			startX := x*widthPerModule + offset - 1
			startY := y*widthPerModule + offset - 1
			// every module separated into nine blocks
			// 1 2 3
			// 4 5 6
			// 7 8 9
			// If the module does not contains the special module, only the center block keep the foreground color, other
			// set the pixel color with the maskImage's.
			var countX, countY int
			for i := startX; i < startX+widthPerModule; i += maskBlockWidth {
				countY = 0
				for j := startY; j < startY+widthPerModule; j += maskBlockWidth {
					// draw each pixel of each separated block.
					pixelX := i
					pixelY := j
					for itx := 0; itx <= maskBlockWidth; itx++ {
						pixelX++
						pixelY = j
						for ity := 0; ity <= maskBlockWidth; ity++ {
							pixelY++
							if maskAreaImage != nil && (q.isDataModule(x, y) || !q.symbol.isUsed[y][x]) {
								if !(countX == 1 && countY == 1) {
									img.Set(pixelX, pixelY, maskAreaImage.At(pixelX, pixelY))
									continue
								}
							}
							if v {
								img.Set(pixelX, pixelY, q.option.ForegroundColor)
							} else if q.symbol.isUsed[y][x] {
								img.Set(pixelX, pixelY, q.option.BackgroundColor)
							}
						}
					}
					countY++
				}
				countX++
			}

		}
	}
	if q.option.Embed {
		return q.embedCode(sourceImage, img), nil
	}
	return img, nil
}

// embedCode overlay the code on image
func (q *HalftoneQRCode) embedCode(dst image.Image, src image.Image) image.Image {
	codeImage := imaging.Resize(src, q.option.MaskRectangle.Size().X, q.option.MaskRectangle.Size().Y, imaging.Lanczos)
	return imaging.Overlay(dst, codeImage, q.option.MaskRectangle.Min, 1)
}

func (q *HalftoneQRCode) isDataModule(x, y int) bool {
	return q.symbol.dataModule[y][x]
}

// NewHalftoneCode constructs a basic QRCode.
//
//	var q *cmd.HalftoneQRCode
//	q, err := cmd.NewHalftoneCode("my content", cmd.Medium)
//
// An error occurs if the content is too long.
func NewHalftoneCode(content string, level RecoveryLevel) (*HalftoneQRCode, error) {
	encoders := []dataEncoderType{dataEncoderType1To9, dataEncoderType10To26,
		dataEncoderType27To40}

	var encoder *dataEncoder
	var encoded *bitset.Bitset
	var chosenVersion *qrCodeVersion
	var err error

	for _, t := range encoders {
		encoder = newDataEncoder(t)
		encoded, err = encoder.encode([]byte(content))

		if err != nil {
			continue
		}

		chosenVersion = chooseQRCodeVersion(level, encoder, encoded.Len())

		if chosenVersion != nil {
			break
		}
	}

	if err != nil {
		return nil, err
	} else if chosenVersion == nil {
		return nil, errors.New("content too long to encode")
	}

	q := &HalftoneQRCode{
		Content: content,

		Level:         level,
		VersionNumber: chosenVersion.version,
		option: &Option{
			ForegroundColor: color.Black,
			BackgroundColor: color.White,
		},

		encoder: encoder,
		data:    encoded,
		version: *chosenVersion,
	}

	q.encode(chosenVersion.numTerminatorBitsRequired(encoded.Len()))

	return q, nil
}

func (q *HalftoneQRCode) encode(numTerminatorBits int) {
	q.addTerminatorBits(numTerminatorBits)
	q.addPadding()

	encoded := q.encodeBlocks()

	const numMasks = 8
	penalty := 0

	for mask := 0; mask < numMasks; mask++ {
		var s *HalftoneSymbol
		var err error

		s, err = buildHalftoneRegularSymbol(q.version, mask, encoded)

		if err != nil {
			log.Panic(err.Error())
		}

		numEmptyModules := s.numEmptyModules()
		if numEmptyModules != 0 {
			log.Panicf("bug: numEmptyModules is %d (expected 0) (version=%d)",
				numEmptyModules, q.VersionNumber)
		}

		p := s.penaltyScore()

		if q.symbol == nil || p < penalty {
			q.symbol = s
			q.mask = mask
			penalty = p
		}
	}
}

// addTerminatorBits adds final terminator bits to the encoded data.
//
// The number of terminator bits required is determined when the QR Code version
// is chosen (which itself depends on the length of the data encoded). The
// terminator bits are thus added after the QR Code version
// is chosen, rather than at the data encoding stage.
func (q *HalftoneQRCode) addTerminatorBits(numTerminatorBits int) {
	q.data.AppendNumBools(numTerminatorBits, false)
}

// addPadding pads the encoded data upto the full length required.
func (q *HalftoneQRCode) addPadding() {
	numDataBits := q.version.numDataBits()

	if q.data.Len() == numDataBits {
		return
	}

	// Pad to the nearest codeword boundary.
	q.data.AppendNumBools(q.version.numBitsToPadToCodeword(q.data.Len()), false)

	// Pad codewords 0b11101100 and 0b00010001.
	padding := [2]*bitset.Bitset{
		bitset.New(true, true, true, false, true, true, false, false),
		bitset.New(false, false, false, true, false, false, false, true),
	}

	// Insert pad codewords alternately.
	i := 0
	for numDataBits-q.data.Len() >= 8 {
		q.data.Append(padding[i])

		i = 1 - i // Alternate between 0 and 1.
	}

	if q.data.Len() != numDataBits {
		log.Panicf("BUG: got len %d, expected %d", q.data.Len(), numDataBits)
	}
}

// encodeBlocks takes the completed (terminated & padded) encoded data, splits
// the data into blocks (as specified by the QR Code version), applies error
// correction to each block, then interleaves the blocks together.
//
// The QR Code's final data sequence is returned.
func (q *HalftoneQRCode) encodeBlocks() *bitset.Bitset {
	// Split into blocks.
	type dataBlock struct {
		data          *bitset.Bitset
		ecStartOffset int
	}

	block := make([]dataBlock, q.version.numBlocks())

	start := 0
	end := 0
	blockID := 0

	for _, b := range q.version.block {
		for j := 0; j < b.numBlocks; j++ {
			start = end
			end = start + b.numDataCodewords*8

			// Apply error correction to each block.
			numErrorCodewords := b.numCodewords - b.numDataCodewords
			block[blockID].data = reedsolomon.Encode(q.data.Substr(start, end), numErrorCodewords)
			block[blockID].ecStartOffset = end - start

			blockID++
		}
	}

	// Interleave the blocks.

	result := bitset.New()

	// Combine data blocks.
	working := true
	for i := 0; working; i += 8 {
		working = false

		for j, b := range block {
			if i >= block[j].ecStartOffset {
				continue
			}

			result.Append(b.data.Substr(i, i+8))

			working = true
		}
	}

	// Combine error correction blocks.
	working = true
	for i := 0; working; i += 8 {
		working = false

		for j, b := range block {
			offset := i + block[j].ecStartOffset
			if offset >= block[j].data.Len() {
				continue
			}

			result.Append(b.data.Substr(offset, offset+8))

			working = true
		}
	}

	// Append remainder bits.
	result.AppendNumBools(q.version.numRemainderBits, false)

	return result
}

// ToString produces a multi-line string that forms a QR-code image.
// This method return the pure qrcode without mask image.
func (q *HalftoneQRCode) ToString() string {
	bits := q.Bitmap()
	var buf bytes.Buffer
	for y := range bits {
		for x := range bits[y] {
			if bits[y][x] {
				buf.WriteString("  ")
			} else {
				buf.WriteString("██")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

// Bitmap returns the QR Code as a 2D array of 1-bit pixels.
//
// bitmap[y][x] is true if the pixel at (x, y) is set.
//
// The bitmap includes the required "quiet zone" around the QR Code to aid
// decoding.
func (q *HalftoneQRCode) Bitmap() [][]bool {
	return q.symbol.bitmap()
}