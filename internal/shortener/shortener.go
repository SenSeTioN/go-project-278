// Package shortener генерирует короткие имена (short_name) для ссылок.
// Используется криптографический источник случайности из crypto/rand,
// чтобы исключить предсказуемость генерируемых идентификаторов.
package shortener

import (
	"crypto/rand"
	"math/big"
)

// charset — base62-алфавит (26 строчных + 26 заглавных + 10 цифр), безопасный для использования в URL-сегментах без дополнительного экранирования.
const (
	charset       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	DefaultLength = 8
)

// Generate возвращает случайную строку заданной длины из алфавита charset.
func Generate(length int) (string, error) {
	if length <= 0 {
		length = DefaultLength
	}
	b := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}
