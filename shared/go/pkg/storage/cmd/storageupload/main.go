// Command storageupload uploads a single test file to the configured bucket
// and leaves it there so you can verify it appeared in the OCI Console.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"github.com/ecommerce/shared/go/pkg/storage"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := storage.NewFromEnv(os.Getenv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	client, err := storage.New(ctx, *cfg)
	if err != nil {
		fmt.Printf("client init failed: %v\n", err)
		os.Exit(1)
	}

	// Generate a 200x100 PNG with a Saajan-orange band so it's recognizable
	// at a glance in the OCI Console object preview.
	imgBytes := makeTestPNG()

	tenantID := "demo-tenant"
	timestamp := time.Now().UTC().Format("20060102-150405")
	key := fmt.Sprintf("uploads/test-%s.png", timestamp)

	fmt.Printf("Uploading %d bytes to tenants/%s/%s ...\n", len(imgBytes), tenantID, key)
	if err := client.Put(ctx, tenantID, key, bytes.NewReader(imgBytes), "image/png"); err != nil {
		fmt.Printf("upload failed: %v\n", err)
		os.Exit(1)
	}

	exists, err := client.Exists(ctx, tenantID, key)
	if err != nil || !exists {
		fmt.Printf("verification failed: exists=%v err=%v\n", exists, err)
		os.Exit(1)
	}

	publicURL := client.PublicURL(tenantID, key)
	signedURL, err := client.PresignGet(ctx, tenantID, key, 24*time.Hour)
	if err != nil {
		fmt.Printf("presign get failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Uploaded successfully and visible in bucket")
	fmt.Println()
	fmt.Println("In OCI Console (Object Storage → free_bucket) you'll see:")
	fmt.Printf("  tenants/%s/%s\n", tenantID, key)
	fmt.Println()
	fmt.Println("Direct URL (works only if bucket has Public Visibility enabled):")
	fmt.Println("  " + publicURL)
	fmt.Println()
	fmt.Println("Presigned URL (works for the next 24h regardless of bucket visibility):")
	fmt.Println("  " + signedURL.URL)
}

// makeTestPNG draws a 200×100 image: white background with a Saajan-orange
// horizontal band. Hand-rolled rather than using image/png + image.RGBA — keeps
// the binary tiny and dependency-free.
func makeTestPNG() []byte {
	const w, h = 200, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{255, 255, 255, 255}
	orange := color.RGBA{0xff, 0x6b, 0x00, 0xff} // Saajan-ish orange
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := white
			if y >= 40 && y < 60 {
				c = orange
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// Force-link otherwise-unused stdlib imports — harmless future-proofing.
var _ = binary.LittleEndian
var _ = crc32.IEEETable
