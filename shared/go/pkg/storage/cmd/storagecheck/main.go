// Command storagecheck exercises the storage package end-to-end against a
// real S3-compatible bucket and reports each step's result. Use it to verify
// OCI namespace/region/bucket/credentials are correctly configured before
// deploying any service.
//
// Usage:
//
//	export OCI_S3_ENDPOINT=https://axwcv3hbytve.compat.objectstorage.ap-singapore-1.oraclecloud.com
//	export OCI_S3_REGION=ap-singapore-1
//	export OCI_S3_BUCKET=free_bucket
//	export OCI_S3_ACCESS_KEY=...
//	export OCI_S3_SECRET_KEY=...
//	go run github.com/ecommerce/shared/go/pkg/storage/cmd/storagecheck
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ecommerce/shared/go/pkg/storage"
)

// 1×1 transparent PNG, ~70 bytes — small enough to upload anywhere.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorBlue  = "\033[34m"
	colorGray  = "\033[90m"
)

func step(name string) {
	fmt.Printf("%s→%s %s ... ", colorBlue, colorReset, name)
}

func ok(detail string) {
	if detail != "" {
		fmt.Printf("%sOK%s %s%s%s\n", colorGreen, colorReset, colorGray, detail, colorReset)
	} else {
		fmt.Printf("%sOK%s\n", colorGreen, colorReset)
	}
}

func fail(err error) {
	fmt.Printf("%sFAIL%s\n  %v\n", colorRed, colorReset, err)
	os.Exit(1)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, err := storage.NewFromEnv(os.Getenv)
	if err != nil {
		fmt.Println(err)
		fmt.Println("\nSet the OCI_S3_* env vars and try again. See storage package docs.")
		os.Exit(1)
	}
	fmt.Printf("%sBucket:%s   %s\n", colorGray, colorReset, cfg.Bucket)
	fmt.Printf("%sRegion:%s   %s\n", colorGray, colorReset, cfg.Region)
	fmt.Printf("%sEndpoint:%s %s\n\n", colorGray, colorReset, cfg.Endpoint)

	step("constructing storage client")
	client, err := storage.New(ctx, *cfg)
	if err != nil {
		fail(err)
	}
	ok("")

	tenantID := "storagecheck"
	suffix := randomHex(6)
	key := fmt.Sprintf("storagecheck/%s.png", suffix)

	step("Put (1x1 PNG)")
	if err := client.Put(ctx, tenantID, key, bytes.NewReader(tinyPNG), "image/png"); err != nil {
		fail(err)
	}
	ok(fmt.Sprintf("uploaded tenants/%s/%s (%d bytes)", tenantID, key, len(tinyPNG)))

	step("Exists")
	exists, err := client.Exists(ctx, tenantID, key)
	if err != nil {
		fail(err)
	}
	if !exists {
		fail(fmt.Errorf("Exists returned false right after Put"))
	}
	ok("object visible")

	step("Get + verify bytes")
	rc, err := client.Get(ctx, tenantID, key)
	if err != nil {
		fail(err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		fail(err)
	}
	if !bytes.Equal(got, tinyPNG) {
		fail(fmt.Errorf("Get returned %d bytes, expected %d (corruption?)", len(got), len(tinyPNG)))
	}
	ok(fmt.Sprintf("%d bytes match", len(got)))

	step("PresignPut (15m)")
	presign, err := client.PresignPut(ctx, tenantID, "presigned/"+suffix+".png", "image/png", 15*time.Minute)
	if err != nil {
		fail(err)
	}
	ok(fmt.Sprintf("URL %s..., expires %s",
		truncate(presign.URL, 70), presign.ExpiresAt.Format(time.RFC3339)))

	step("PUT through presigned URL (browser path)")
	req, _ := http.NewRequestWithContext(ctx, presign.Method, presign.URL, bytes.NewReader(tinyPNG))
	for k, v := range presign.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(fmt.Errorf("HTTP request failed: %w", err))
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fail(fmt.Errorf("HTTP %d from presigned PUT: %s", resp.StatusCode, truncate(string(body), 200)))
	}
	ok(fmt.Sprintf("HTTP %d", resp.StatusCode))

	step("Verify presigned object lands in bucket")
	exists2, err := client.Exists(ctx, tenantID, "presigned/"+suffix+".png")
	if err != nil {
		fail(err)
	}
	if !exists2 {
		fail(fmt.Errorf("presigned upload didn't materialize in bucket"))
	}
	ok("")

	step("PresignGet")
	presignGet, err := client.PresignGet(ctx, tenantID, key, 5*time.Minute)
	if err != nil {
		fail(err)
	}
	getResp, err := http.Get(presignGet.URL)
	if err != nil {
		fail(err)
	}
	getBody, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if getResp.StatusCode != 200 || !bytes.Equal(getBody, tinyPNG) {
		fail(fmt.Errorf("presigned GET returned status %d, %d bytes", getResp.StatusCode, len(getBody)))
	}
	ok("retrieved correct bytes")

	step("PublicURL")
	pub := client.PublicURL(tenantID, key)
	ok(pub)

	step("Cleanup (delete both objects)")
	if err := client.Delete(ctx, tenantID, key); err != nil {
		fail(err)
	}
	if err := client.Delete(ctx, tenantID, "presigned/"+suffix+".png"); err != nil {
		fail(err)
	}
	ok("")

	step("Exists returns false after Delete")
	stillThere, err := client.Exists(ctx, tenantID, key)
	if err != nil {
		fail(err)
	}
	if stillThere {
		fail(fmt.Errorf("object still present after Delete"))
	}
	ok("")

	fmt.Printf("\n%sAll checks passed.%s Storage is wired correctly.\n", colorGreen, colorReset)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
