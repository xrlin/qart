// go-HalftoneQRCode
// Copyright 2014 Tom Harwood

package qart

import (
	"strings"
	"testing"
)

func TestHalftoneQRCodeMaxCapacity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHalftoneQRCodeCapacity")
	}

	tests := []struct {
		string         string
		numRepetitions int
	}{
		{
			"0",
			7089,
		},
		{
			"A",
			4296,
		},
		{
			"#",
			2953,
		},
		// Alternate byte/numeric data types. Optimises to 2,952 bytes.
		{
			"#1",
			1476,
		},
	}

	for _, test := range tests {
		_, err := NewHalftoneCode(strings.Repeat(test.string, test.numRepetitions), Low)

		if err != nil {
			t.Errorf("%d x '%s' got %s expected success", test.numRepetitions,
				test.string, err.Error())
		}
	}

	for _, test := range tests {
		_, err := NewHalftoneCode(strings.Repeat(test.string, test.numRepetitions+1), Low)

		if err == nil {
			t.Errorf("%d x '%s' chars encodable, expected not encodable",
				test.numRepetitions+1, test.string)
		}
	}
}

func TestHalftoneQRCode_AddOption(t *testing.T) {
	hcode, _ := NewHalftoneCode("test", Low)
	hcode.AddOption(Option{MaskImagePath: "test.png"})
	if hcode.option.MaskImagePath != "test.png" {
		t.Errorf("set MaskImagePath to test.png but result is %s", hcode.option.MaskImagePath)
	}
}

func TestHalftoneQRCode_RemoveOption(t *testing.T) {
	hcode, _ := NewHalftoneCode("test", Low)
	hcode.option = &Option{MaskImagePath: "test.png"}
	hcode.RemoveOption(MaskImagePathOpt)
	if hcode.option.MaskImagePath != "" {
		t.Errorf("remove MaskImagePath(before: %s) but result is %s(after)", "test.png", hcode.option.MaskImagePath)
	}
}

func TestHalftoneQRCodeVersionCapacity(t *testing.T) {
	tests := []struct {
		version         int
		level           RecoveryLevel
		maxNumeric      int
		maxAlphanumeric int
		maxByte         int
	}{
		{
			1,
			Low,
			41,
			25,
			17,
		},
		{
			2,
			Low,
			77,
			47,
			32,
		},
		{
			2,
			Highest,
			34,
			20,
			14,
		},
		{
			40,
			Low,
			7089,
			4296,
			2953,
		},
		{
			40,
			Highest,
			3057,
			1852,
			1273,
		},
	}

	for i, test := range tests {
		numericData := strings.Repeat("1", test.maxNumeric)
		alphanumericData := strings.Repeat("A", test.maxAlphanumeric)
		byteData := strings.Repeat("#", test.maxByte)

		var n *HalftoneQRCode
		var a *HalftoneQRCode
		var b *HalftoneQRCode
		var err error

		n, err = NewHalftoneCode(numericData, test.level)
		if err != nil {
			t.Fatal(err.Error())
		}

		a, err = NewHalftoneCode(alphanumericData, test.level)
		if err != nil {
			t.Fatal(err.Error())
		}

		b, err = NewHalftoneCode(byteData, test.level)
		if err != nil {
			t.Fatal(err.Error())
		}

		if n.VersionNumber != test.version {
			t.Fatalf("Test #%d numeric has version #%d, expected #%d", i,
				n.VersionNumber, test.version)
		}

		if a.VersionNumber != test.version {
			t.Fatalf("Test #%d alphanumeric has version #%d, expected #%d", i,
				a.VersionNumber, test.version)
		}

		if b.VersionNumber != test.version {
			t.Fatalf("Test #%d byte has version #%d, expected #%d", i,
				b.VersionNumber, test.version)
		}
	}
}

func TestHalftoneQRCodeISOAnnexIExample(t *testing.T) {
	var q *HalftoneQRCode
	q, err := NewHalftoneCode("01234567", Medium)

	if err != nil {
		t.Fatalf("Error producing ISO Annex I Example: %s, expected success",
			err.Error())
	}

	const expectedMask int = 2

	if q.mask != 2 {
		t.Errorf("ISO Annex I example mask got %d, expected %d\n", q.mask,
			expectedMask)
	}
}

func BenchmarkHalftoneQRCodeURLSize(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewHalftoneCode("http://www.example.org", Medium)
	}
}

func BenchmarkHalftoneQRCodeMaximumSize(b *testing.B) {
	for n := 0; n < b.N; n++ {
		// 7089 is the maximum encodable number of numeric digits.
		NewHalftoneCode(strings.Repeat("0", 7089), Low)
	}
}
