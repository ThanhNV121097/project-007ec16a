package payments

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	// napasGUID identifies NAPAS's national transfer service in the merchant account tag.
	napasGUID = "A000000727"
	// serviceTransfer is the service code of a transfer to an account.
	serviceTransfer = "QRIBFTTA"
	// maxNote is the longest transfer note the payload carries.
	maxNote = 25
	// maxAmount is the largest amount tag 54 holds: 13 digits.
	maxAmount = 9_999_999_999_999
)

// tlv is one EMVCo field: a two-digit tag, a two-digit length, the value.
func tlv(tag, value string) string { return fmt.Sprintf("%s%02d%s", tag, len(value), value) }

// crc16 is CRC-16/CCITT-FALSE: polynomial 0x1021, initial value 0xFFFF, no
// reflection, no final xor. The CRC of "123456789" is 0x29B1.
func crc16(s string) uint16 {
	crc := uint16(0xFFFF)
	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func allBytes(s string, ok func(byte) bool) bool {
	for i := 0; i < len(s); i++ {
		if !ok(s[i]) {
			return false
		}
	}
	return true
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isAlnum(b byte) bool { return isDigit(b) || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

func isPrintable(b byte) bool { return b >= 0x20 && b < 0x7f }

// vietQR builds the VietQR text of a transfer to one account: the bank's BIN, the
// account, the amount in whole VND and the note, closed by the checksum. The
// builder refuses what the settings never produce (a BIN that is not six digits,
// an account that is not 1 to 19 letters or digits) and what the page must never
// ask for (no amount, a note over 25 characters), so a malformed payload is never
// drawn.
func vietQR(bin, account string, amount int64, note string) (string, error) {
	switch {
	case len(bin) != 6 || !allBytes(bin, isDigit):
		return "", errors.New("the bank's BIN is not six digits")
	case len(account) < 1 || len(account) > 19 || !allBytes(account, isAlnum):
		return "", errors.New("the account number is not 1 to 19 letters or digits")
	case amount < 1 || amount > maxAmount:
		return "", errors.New("the amount does not fit the payload")
	case len(note) < 1 || len(note) > maxNote || !allBytes(note, isPrintable):
		return "", errors.New("the note is empty, over 25 characters or not plain ASCII")
	}
	merchant := tlv("00", napasGUID) + tlv("01", tlv("00", bin)+tlv("01", account)) + tlv("02", serviceTransfer)
	body := tlv("00", "01") + tlv("01", "12") + tlv("38", merchant) + tlv("53", "704") +
		tlv("54", strconv.FormatInt(amount, 10)) + tlv("58", "VN") + tlv("62", tlv("08", note)) + "6304"
	return body + fmt.Sprintf("%04X", crc16(body)), nil
}
