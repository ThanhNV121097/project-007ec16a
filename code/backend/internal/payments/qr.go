package payments

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// qrRows draws the payload as a QR code: one string per row of modules, "1" dark
// and "0" light, square, without the quiet zone (the page draws its own margin).
// The error correction is medium. The payload is the one the shop built, so the
// picture a customer scans is the text the tests read.
func qrRows(payload string) ([]string, error) {
	code, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("encode the QR code: %w", err)
	}
	code.DisableBorder = true
	bitmap := code.Bitmap()
	rows := make([]string, len(bitmap))
	for y, line := range bitmap {
		var b strings.Builder
		for _, dark := range line {
			if dark {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		rows[y] = b.String()
	}
	return rows, nil
}
