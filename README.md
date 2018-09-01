# Qart #

[![Build Status](https://travis-ci.org/xrlin/qart.svg?branch=master)](https://travis-ci.org/xrlin/qart)
[![godoc](https://camo.githubusercontent.com/4953dcce3ef06016a8f872b20e3bf6cd65e99621/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f676f646f632d7265666572656e63652d3532373242342e737667)](https://godoc.org/github.com/xrlin/qart)

Package qart implements a QR Code encoder, basing on [go-qrcode](https://github.com/skip2/go-qrcode), with other funny features.

## Install

    go get -u github.com/xrlin/qart/...

A command-line tool `qart` will be built into `$GOPATH/bin/`.

## CLI

```bash
# create code with png
qart -m test.png -o out.png http://example.com

# create code with specified area of png
qart -m test.png -startX 100 -startY 100 -width 100 -oout.png http://example.com

# overlay code on source image
qart -m test.png -startX 100 -startY 100 -width 100 -embed true -o out.gif http://example.com

# create code with gif
qart -m illya.gif -o out.png http://example.com
```
More options can found by

```bash
qart -h
```

## Usage

```go
import "github.com/xrlin/qart"

q, err := qrcode.NewHalftoneCode(content, qrcode.Highest)
q.AddOption(qart.Option{Embed: false, MaskImagePath: "test.png"})
pointWidth := 3
// Get the image.Image represents the qr code
ret := q.CodeImage(pointWidth)

// Get the bytes of image
imgBytes, err := q.ImageData(pointWidth)

```

Read the godoc for more usages.

## DemoApp

[https://xrlin.github.io/qart-web](https://xrlin.github.io/qart-web)
## Examples

<p>
<img alt="example" src="https://raw.githubusercontent.com/xrlin/qart/master/screenshots/example1.png" width="200px"/>
<img alt="example" src="https://raw.githubusercontent.com/xrlin/qart/master/screenshots/example2.png" width="200px"/>
<img alt="example" src="https://raw.githubusercontent.com/xrlin/qart/master/screenshots/example3.gif" width="200px"/>
</p>
<img alt="example" src="https://raw.githubusercontent.com/xrlin/qart/master/screenshots/example4.gif" width="300px"/>

## Links

- [go-qrcode](https://github.com/skip2/go-qrcode) :sparkles:
- [pyqart](https://github.com/7sDream/pyqart) :clap:
- [Paper](http://vecg.cs.ucl.ac.uk/Projects/SmartGeometry/halftone_QR/halftoneQR_sigga13.html)
