package web

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	captchaLength = 5
	captchaTTL    = 5 * time.Minute
)

var captchaAlphabet = []byte("23456789ABCDEFGHJKLMNPQRSTUVWXYZ")

var captchaGlyphs = map[rune][7]string{
	'2': {"11110", "00001", "00001", "01110", "10000", "10000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"10010", "10010", "10010", "11111", "00010", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01111", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "11110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'J': {"00111", "00010", "00010", "00010", "00010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "11011", "10001"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}

type captchaPayload struct {
	Answer    string `json:"answer"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
}

// captchaManager keeps CAPTCHA challenges stateless. The answer is encrypted
// and authenticated with a key derived from SESSION_SECRET, so the same token
// can be rendered and verified by every application instance without storing
// the clear-text answer in the browser or database.
type captchaManager struct {
	key []byte
	now func() time.Time
}

func newCaptchaManager(secret string) *captchaManager {
	sum := sha256.Sum256([]byte("livira-login-captcha\x00" + secret))
	return &captchaManager{key: append([]byte(nil), sum[:]...), now: time.Now}
}

func (m *captchaManager) newChallenge() (token, answer string, err error) {
	answerBytes := make([]byte, captchaLength)
	for i := range answerBytes {
		index, randomErr := cryptoRandomInt(len(captchaAlphabet))
		if randomErr != nil {
			return "", "", randomErr
		}
		answerBytes[i] = captchaAlphabet[index]
	}
	now := m.now().UTC()
	payload, err := json.Marshal(captchaPayload{Answer: string(answerBytes), IssuedAt: now.Unix(), ExpiresAt: now.Add(captchaTTL).Unix()})
	if err != nil {
		return "", "", err
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", err
	}
	sealed := gcm.Seal(nonce, nonce, payload, []byte("LIVIRA-CAPTCHA-v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), string(answerBytes), nil
}

func (m *captchaManager) decode(token string) (captchaPayload, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 512 {
		return captchaPayload{}, errors.New("invalid captcha token")
	}
	sealed, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return captchaPayload{}, errors.New("invalid captcha token")
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return captchaPayload{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return captchaPayload{}, err
	}
	if len(sealed) < gcm.NonceSize()+gcm.Overhead() {
		return captchaPayload{}, errors.New("invalid captcha token")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte("LIVIRA-CAPTCHA-v1"))
	if err != nil {
		return captchaPayload{}, errors.New("invalid captcha token")
	}
	var payload captchaPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return captchaPayload{}, errors.New("invalid captcha token")
	}
	now := m.now().UTC().Unix()
	if len(payload.Answer) != captchaLength || payload.IssuedAt > now+60 || payload.ExpiresAt < now || payload.ExpiresAt-payload.IssuedAt > int64(10*time.Minute/time.Second) {
		return captchaPayload{}, errors.New("expired captcha token")
	}
	return payload, nil
}

func (m *captchaManager) verify(token, answer string) bool {
	payload, err := m.decode(token)
	if err != nil {
		return false
	}
	expected := sha256.Sum256([]byte(payload.Answer))
	actual := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(answer))))
	return subtle.ConstantTimeCompare(expected[:], actual[:]) == 1
}

func (m *captchaManager) renderPNG(token string) ([]byte, error) {
	payload, err := m.decode(token)
	if err != nil {
		return nil, err
	}
	const width, height, scale = 220, 68, 5
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	background := color.RGBA{R: 242, G: 248, B: 248, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			canvas.SetRGBA(x, y, background)
		}
	}

	// Light noise is drawn both behind and over the glyphs. The glyphs are
	// rasterized from a tiny built-in bitmap alphabet, so the answer never
	// appears as extractable text in the image response.
	for i := 0; i < 65; i++ {
		x, _ := cryptoRandomInt(width)
		y, _ := cryptoRandomInt(height)
		canvas.SetRGBA(x, y, color.RGBA{R: 122, G: 164, B: 166, A: 105})
	}
	characterColors := []color.RGBA{
		{R: 16, G: 59, B: 75, A: 255},
		{R: 8, G: 105, B: 99, A: 255},
		{R: 34, G: 76, B: 101, A: 255},
	}
	for index, character := range payload.Answer {
		glyph, ok := captchaGlyphs[character]
		if !ok {
			return nil, errors.New("captcha glyph unavailable")
		}
		jitterY, _ := cryptoRandomInt(7)
		skew, _ := cryptoRandomInt(5)
		inkIndex, _ := cryptoRandomInt(len(characterColors))
		baseX := 19 + index*39
		baseY := 10 + jitterY
		rowSkew := skew - 2
		for row, pattern := range glyph {
			rowOffset := (row - 3) * rowSkew / 3
			for column, bit := range pattern {
				if bit != '1' {
					continue
				}
				fillCaptchaRect(canvas, baseX+column*scale+rowOffset, baseY+row*scale, scale, scale, characterColors[inkIndex])
			}
		}
	}

	waveColors := []color.RGBA{
		{R: 68, G: 143, B: 139, A: 150},
		{R: 104, G: 132, B: 151, A: 125},
	}
	for wave := 0; wave < 2; wave++ {
		phase, _ := cryptoRandomInt(25)
		amplitude := 5 + wave*2
		center := 23 + wave*24
		previousY := center
		for x := 0; x < width; x++ {
			y := center + int(math.Sin((float64(x+phase)/19.0))*float64(amplitude))
			drawCaptchaLine(canvas, x-1, previousY, x, y, waveColors[wave])
			previousY = y
		}
	}
	for i := 0; i < 5; i++ {
		x1, _ := cryptoRandomInt(width)
		y1, _ := cryptoRandomInt(height)
		x2, _ := cryptoRandomInt(width)
		y2, _ := cryptoRandomInt(height)
		drawCaptchaLine(canvas, x1, y1, x2, y2, color.RGBA{R: 126, G: 159, B: 164, A: 80})
	}

	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&encoded, canvas); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func fillCaptchaRect(canvas *image.RGBA, x, y, width, height int, ink color.RGBA) {
	for offsetY := 0; offsetY < height; offsetY++ {
		for offsetX := 0; offsetX < width; offsetX++ {
			pointX, pointY := x+offsetX, y+offsetY
			if image.Pt(pointX, pointY).In(canvas.Bounds()) {
				canvas.SetRGBA(pointX, pointY, ink)
			}
		}
	}
}

func drawCaptchaLine(canvas *image.RGBA, x0, y0, x1, y1 int, ink color.RGBA) {
	deltaX, stepX := absInt(x1-x0), 1
	if x0 > x1 {
		stepX = -1
	}
	deltaY, stepY := -absInt(y1-y0), 1
	if y0 > y1 {
		stepY = -1
	}
	errorValue := deltaX + deltaY
	for {
		if image.Pt(x0, y0).In(canvas.Bounds()) {
			canvas.SetRGBA(x0, y0, ink)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		doubled := 2 * errorValue
		if doubled >= deltaY {
			errorValue += deltaY
			x0 += stepX
		}
		if doubled <= deltaX {
			errorValue += deltaX
			y0 += stepY
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func cryptoRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("invalid random range")
	}
	value, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func (s *Server) captchaPNG(w http.ResponseWriter, r *http.Request) {
	content, err := s.captcha.renderPNG(r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, "CAPTCHA tidak valid atau sudah kedaluwarsa", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) newCaptcha(w http.ResponseWriter, _ *http.Request) {
	token, _, err := s.captcha.newChallenge()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "CAPTCHA belum dapat dibuat"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "image_url": "/captcha.png?token=" + url.QueryEscape(token)})
}
