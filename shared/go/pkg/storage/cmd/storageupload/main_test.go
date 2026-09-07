package main

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

// makeTestPNG's output is uploaded and then reported as proof that storage
// works, so bytes that are not a decodable PNG would make the tool certify a
// corrupt object. Round-trip the encode rather than trusting its length.
func TestMakeTestPNGRoundTrips(t *testing.T) {
	got, err := makeTestPNG()
	if err != nil {
		t.Fatalf("makeTestPNG: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("decode %d encoded bytes: %v", len(got), err)
	}

	if b := img.Bounds(); b.Dx() != 200 || b.Dy() != 100 {
		t.Errorf("bounds = %dx%d, want 200x100", b.Dx(), b.Dy())
	}

	// The orange band is what makes the object recognizable in the OCI
	// Console preview; a blank image would still decode.
	orange := color.RGBA{0xff, 0x6b, 0x00, 0xff}
	r, g, b, a := img.At(100, 50).RGBA()
	wr, wg, wb, wa := orange.RGBA()
	if r != wr || g != wg || b != wb || a != wa {
		t.Errorf("pixel(100,50) = %v, want %v", img.At(100, 50), orange)
	}
}
