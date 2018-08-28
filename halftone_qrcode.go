package qart

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/xrlin/qart/bitset"
	"github.com/xrlin/qart/reedsolomon"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

type HalftoneQRCode struct {
	// Original content encoded.
	Content string

	// QR Code type.
	Level         RecoveryLevel
	VersionNumber int

	// User settable drawing options.
	ForegroundColor color.Color
	BackgroundColor color.Color

	encoder *dataEncoder
	version qrCodeVersion

	data   *bitset.Bitset
	symbol *HalftoneSymbol

	mask          int
	MaskImagePath string
	MaskImage     image.Image

	MaskRectangle image.Rectangle

	Embed bool
}

// Image returns the QR Code as an image.Image.
//
// A positive size sets a fixed image width and height (e.g. 256 yields an
// 256x256px image).
//
// Depending on the amount of data encoded, fixed size images can have different
// amounts of padding (white space around the QR Code). As an alternative, a
// variable sized image can be generated instead:
//
// A negative size causes a variable sized image to be returned. The image
// returned is the minimum size required for the QR Code. Choose a larger
// negative number to increase the scale of the image. e.g. a size of -5 causes
// each module (QR Code "pixel") to be 5px in size.
func (q *HalftoneQRCode) Image(pointWidth int) image.Image {
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
			img.Set(i, j, q.BackgroundColor)
		}
	}

	srcMaskImage, maskImage, err := q.openMaskImage(img.Bounds())
	if err != nil {
		fmt.Println(err)
		return nil
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
						pixelX += 1
						pixelY = j
						for ity := 0; ity <= maskBlockWidth; ity++ {
							pixelY += 1
							if maskImage != nil && (q.isDataModule(x, y) || !q.symbol.isUsed[y][x]) {
								if !(countX == 1 && countY == 1) {
									img.Set(pixelX, pixelY, maskImage.At(pixelX, pixelY))
									continue
								}
							}
							if v {
								img.Set(pixelX, pixelY, q.ForegroundColor)
							} else if q.symbol.isUsed[y][x] {
								img.Set(pixelX, pixelY, q.BackgroundColor)
							}
						}
					}
					countY++
				}
				countX++
			}

		}
	}
	if q.Embed {
		return q.embedCode(srcMaskImage, img)
	}
	return img
}

// openMaskImage return the maskImage object if maskPath  is present.
// Besides if maskPath is blank, no error returned.
func (q *HalftoneQRCode) openMaskImage(bounds image.Rectangle) (sourceMaskImage image.Image, maskImage image.Image, err error) {
	if q.MaskImage == nil {
		if q.MaskImagePath == "" {
			return
		}
		var f *os.File
		f, err = os.Open(q.MaskImagePath)
		if err != nil {
			return
		}
		defer f.Close()
		sourceMaskImage, _, err = image.Decode(f)
	} else {
		sourceMaskImage = q.MaskImage
	}
	maskImage = imaging.Clone(sourceMaskImage)

	if err != nil {
		return
	}
	if q.Embed && !bounds.In(maskImage.Bounds()) {
		err = errors.New("when in embed mode, mask image must not smaller than the cmd")
		return
	}
	if !q.MaskRectangle.Empty() {
		if !q.MaskRectangle.In(maskImage.Bounds()) {
			err = errors.New("sub mask image area must in the mask image")
			return
		}
		maskImage = imaging.Crop(maskImage, q.MaskRectangle)
	}
	maskImage = imaging.Resize(maskImage, bounds.Max.X, bounds.Max.Y, imaging.Lanczos)
	return
}

func (q *HalftoneQRCode) embedCode(dst image.Image, src image.Image) image.Image {
	codeImage := imaging.Resize(src, q.MaskRectangle.Size().X, q.MaskRectangle.Size().Y, imaging.Lanczos)
	return imaging.Overlay(dst, codeImage, q.MaskRectangle.Min, 1)
}

func (q *HalftoneQRCode) isDataModule(x, y int) bool {
	return q.symbol.dataModule[y][x]
}

// New constructs a QRCode.
//
//	var q *cmd.QRCode
//	q, err := cmd.New("my content", cmd.Medium)
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

		ForegroundColor: color.Black,
		BackgroundColor: color.White,

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
func (q *HalftoneQRCode) ToString(inverseColor bool) string {
	bits := q.Bitmap()
	var buf bytes.Buffer
	for y := range bits {
		for x := range bits[y] {
			if bits[y][x] != inverseColor {
				buf.WriteString("  ")
			} else {
				buf.WriteString("██")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

// PNG returns the QR Code as a PNG image.
//
// size is both the image width and height in pixels. If size is too small then
// a larger image is silently returned. Negative values for size cause a
// variable sized image to be returned: See the documentation for Image().
func (q *HalftoneQRCode) PNG(size int) ([]byte, error) {
	img := q.Image(size)

	encoder := png.Encoder{CompressionLevel: png.BestCompression}

	var b bytes.Buffer
	err := encoder.Encode(&b, img)

	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
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
