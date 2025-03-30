package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/golang/freetype/truetype"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var killed bool

type symbolMatch struct {
	symbol     string
	position   image.Point
	confidence float32
	sizeX      int
	sizeY      int
	centerX    int
	centerY    int
	disabled   bool
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type DecodeRequest struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Image string `json:"image"`
}

type EncodeRequest struct {
	Text string `json:"text"`
}

type DecodeResponse struct {
	Phonetics string `json:"phonetics"`
	AlienText string `json:"alien"`
}

type EncodeResponse struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Image string `json:"image"`
}

func average(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	sum := 0
	for _, num := range nums {
		sum += num
	}
	return sum / len(nums)
}

func averageFloat(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	sum := float64(0)
	for _, num := range nums {
		sum += num
	}
	return sum / float64(len(nums))
}

func prepareBinaryImage(imgData []byte) (gocv.Mat, error) {
	imgMat, err := gocv.IMDecode(imgData, gocv.IMReadColor)
	if err != nil || imgMat.Empty() {
		return gocv.NewMat(), fmt.Errorf("error decoding image")
	}
	defer imgMat.Close()
	gray := gocv.NewMat()
	gocv.CvtColor(imgMat, &gray, gocv.ColorBGRToGray)

	binary := gocv.NewMat()

	gocv.Threshold(gray, &binary, 0, 255, gocv.ThresholdBinary|gocv.ThresholdOtsu)
	gray.Close()

	whiteCount := gocv.CountNonZero(binary)
	totalPixels := binary.Rows() * binary.Cols()
	blackCount := totalPixels - whiteCount

	if blackCount > whiteCount {
		inverted := gocv.NewMat()
		gocv.BitwiseNot(binary, &inverted)
		binary.Close()
		binary = inverted
	}

	return binary, nil
}

func measureMixedString(s string, asciiFace, ttfFace font.Face) int {
	var total fixed.Int26_6
	for _, r := range s {
		var adv fixed.Int26_6
		if r < 128 {
			if a, ok := asciiFace.GlyphAdvance(r); ok {
				adv = a
			}
		} else {
			if a, ok := ttfFace.GlyphAdvance(r); ok {
				adv = a
			}
		}
		total += adv
	}
	return total.Round()
}

func drawMixedString(dst draw.Image, x, y int, s string, asciiFace, ttfFace font.Face) {
	d := &font.Drawer{
		Dst: dst,
		Src: image.White,
	}
	currentX := x

	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		var currentFace font.Face
		if r < 128 {
			currentFace = asciiFace
		} else {
			currentFace = ttfFace
		}

		segment := ""
		for len(s) > 0 {
			r, size = utf8.DecodeRuneInString(s)
			if (r < 128) == (currentFace == asciiFace) {
				segment += string(r)
				s = s[size:]
			} else {
				break
			}
		}

		d.Face = currentFace
		d.Dot = fixed.Point26_6{
			X: fixed.I(currentX),
			Y: fixed.I(y),
		}
		d.DrawString(segment)
		currentX += d.MeasureString(segment).Round()
	}
}

func wrapMixedText(text string, asciiFace, ttfFace font.Face, maxWidth int) []string {
	words := strings.Fields(text)
	var lines []string
	currentLine := ""

	for _, word := range words {
		if measureMixedString(word, asciiFace, ttfFace) > maxWidth {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			var currentWordPart string
			for _, r := range word {
				charStr := string(r)
				if measureMixedString(currentWordPart+charStr, asciiFace, ttfFace) <= maxWidth {
					currentWordPart += charStr
				} else {
					lines = append(lines, currentWordPart)
					currentWordPart = charStr
				}
			}
			currentLine = currentWordPart
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				testLine := currentLine + " " + word
				if measureMixedString(testLine, asciiFace, ttfFace) <= maxWidth {
					currentLine = testLine
				} else {
					lines = append(lines, currentLine)
					currentLine = word
				}
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

func renderTextToPNG(text, fontPath string) (string, error) {
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return "", fmt.Errorf("failed to read font file: %v", err)
	}
	ttf, err := truetype.Parse(fontData)
	if err != nil {
		return "", fmt.Errorf("failed to parse font: %v", err)
	}

	const fontSize = 32
	ttfFace := truetype.NewFace(ttf, &truetype.Options{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	asciiFace := basicfont.Face7x13

	const maxWidth = 800
	lines := wrapMixedText(text, asciiFace, ttfFace, maxWidth)

	ttfMetrics := ttfFace.Metrics()
	lineHeight := (ttfMetrics.Ascent + ttfMetrics.Descent).Ceil()
	lineSpacing := lineHeight + 4
	imgHeight := lineSpacing*len(lines) + 10

	rgba := image.NewRGBA(image.Rect(0, 0, maxWidth, imgHeight))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	y := ttfMetrics.Ascent.Ceil()
	for _, line := range lines {
		drawMixedString(rgba, 0, y, line, asciiFace, ttfFace)
		y += lineSpacing
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %v", err)
	}
	encodedStr := base64.StdEncoding.EncodeToString(buf.Bytes())
	return encodedStr, nil
}

func readImageToSymbols(imgdata []byte) (string, error) {

	inputGray, err := prepareBinaryImage(imgdata)
	if err != nil {
		log.Fatalf("Error preparing binary image: %v", err)
	}
	defer inputGray.Close()
	matches := []*symbolMatch{}

	templates, err := filepath.Glob("./train/*.png")
	if err != nil {
		return "", err
	}

	if len(templates) == 0 {
		fmt.Println("No template files found.")
		return "", err
	}

	scaleFound := false
	foundScale := float64(1)
	scales := []float64{}

	for _, tmplFilename := range templates {
		symbol := strings.TrimSuffix(strings.TrimPrefix(tmplFilename, "train/"), ".png")

		tmpl := gocv.IMRead(tmplFilename, gocv.IMReadGrayScale)
		if tmpl.Empty() {
			continue
		}
		defer tmpl.Close()

		bestScore := float32(-1.0)
		var bestScale float64

		scaleLower := 0.05
		scaleUpper := 10.0
		scaleStep := 0.1

		if len(scales) > 5 {
			avgLastScales := averageFloat(scales[len(scales)-3:])
			if avgLastScales/scales[len(scales)-1] > 0.95 && avgLastScales/scales[len(scales)-1] < 1.05 {
				scaleFound = true
				foundScale = avgLastScales
			}
		}

		if !scaleFound {
			for range 3 {
				for scale := scaleLower; scale <= scaleUpper; scale += scaleStep {
					scaledTemplate := gocv.NewMat()
					newWidth := int(float64(tmpl.Cols()) * scale)
					newHeight := int(float64(tmpl.Rows()) * scale)
					gocv.Resize(tmpl, &scaledTemplate, image.Point{X: newWidth, Y: newHeight}, 0, 0, gocv.InterpolationLinear)
					if scaledTemplate.Cols() > inputGray.Cols() || scaledTemplate.Rows() > inputGray.Rows() {
						scaledTemplate.Close()
						continue
					}
					result := gocv.NewMat()
					gocv.MatchTemplate(inputGray, scaledTemplate, &result, gocv.TmCcoeffNormed, gocv.NewMat())
					_, maxVal, _, _ := gocv.MinMaxLoc(result)

					if maxVal > bestScore {
						bestScore = maxVal
						bestScale = scale
					}
					result.Close()
					scaledTemplate.Close()
				}
				scaleLower = max(0.1, bestScale-scaleStep)
				scaleUpper = bestScale + scaleStep
				scaleStep = scaleStep / 4
			}
		} else {
			bestScale = foundScale
		}

		scales = append(scales, bestScale)
		scaledTemplate := gocv.NewMat()
		newWidth := int(float64(tmpl.Cols()) * bestScale)
		newHeight := int(float64(tmpl.Rows()) * bestScale)
		gocv.Resize(tmpl, &scaledTemplate, image.Point{X: newWidth, Y: newHeight}, 0, 0, gocv.InterpolationLinear)
		defer scaledTemplate.Close()

		if scaledTemplate.Cols() > inputGray.Cols() || scaledTemplate.Rows() > inputGray.Rows() {
			continue
		}

		result := gocv.NewMat()
		gocv.MatchTemplate(inputGray, scaledTemplate, &result, gocv.TmCcoeffNormed, gocv.NewMat())

		_, bestScore, _, _ = gocv.MinMaxLoc(result)

		baseThreshold := float32(0.68)
		matchThreshold := max(baseThreshold, bestScore*0.85)

		var matchLocations []*symbolMatch
		for y := range result.Rows() {
			for x := range result.Cols() {
				val := result.GetFloatAt(y, x)
				if val >= matchThreshold {
					matchLocations = append(matchLocations, &symbolMatch{symbol: symbol, confidence: val, position: image.Pt(x, y), sizeX: newWidth, sizeY: newHeight, disabled: false})
				}
			}
		}
		result.Close()

		matches = append(matches, matchLocations...)
	}

	rowPositions := []int{}
	heights := []int{}
	cols := map[int][]int{}

	for _, match := range matches {
		ratio := float64(match.sizeX) / float64(max(match.sizeY, 1))
		isSquare := ratio < 1.15 && ratio > 0.85
		centerY := match.position.Y + match.sizeY/2
		match.centerX = match.position.X + match.sizeX/2
		if isSquare {
			heights = append(heights, match.sizeY)
			found := false
			for _, r := range rowPositions {
				if math.Abs(float64(centerY)-float64(r)) < float64(match.sizeY)/4 {
					match.centerY = r
					found = true
					break
				}
			}
			if !found {
				match.centerY = centerY
				rowPositions = append(rowPositions, centerY)
				cols[centerY] = []int{}
			}
		} else {
			match.centerY = centerY
		}
	}

	avgHeight := average(heights)

	cleanedMatches := []*symbolMatch{}
	for _, match := range matches {
		found := false
		for _, r := range rowPositions {
			if math.Abs(float64(match.centerY)-float64(r)) < float64(match.sizeY)/4 {
				match.centerY = r
				found = true
				break
			}
		}
		if !found || math.Abs(float64(avgHeight-match.sizeY)) > float64(match.sizeY)/4 {
			match.disabled = true
			continue
		}
		cleanedMatches = append(cleanedMatches, match)
	}

	for _, match := range cleanedMatches {
		found := false
		for _, x := range cols[match.centerY] {
			if math.Abs(float64(match.centerX)-float64(x)) < float64(match.sizeX)/3 {
				match.centerX = x
				found = true
				break
			}
		}
		if !found {
			cols[match.centerY] = append(cols[match.centerY], match.centerX)
		}
	}

	toDisable := []*symbolMatch{}
	sortedByCenter := map[int]map[int][]*symbolMatch{}
	for _, match := range cleanedMatches {
		if row, exists := sortedByCenter[match.centerY]; exists {
			if cell, exists := row[match.centerX]; exists {
				row[match.centerX] = append(cell, match)
			} else {
				row[match.centerX] = []*symbolMatch{match}
			}
		} else {
			sortedByCenter[match.centerY] = map[int][]*symbolMatch{match.centerX: {match}}
		}
	}
	for _, row := range sortedByCenter {
		for _, cell := range row {
			slices.SortStableFunc(cell, func(i, j *symbolMatch) int {
				if float64(min(j.sizeX, i.sizeX))/float64(max(j.sizeX, i.sizeX)) < 0.25 {
					return j.sizeX - i.sizeX
				}
				return int((j.confidence - i.confidence) * 1000)
			})
			for _, match := range cell[1:] {
				match.disabled = true
			}
		}
	}

	for _, d := range toDisable {
		d.disabled = true
	}

	matchedSymbols := []*symbolMatch{}
	for _, match := range matches {
		if match.disabled {
			continue
		}
		matchedSymbols = append(matchedSymbols, match)
	}

	slices.SortStableFunc(matchedSymbols, func(i, j *symbolMatch) int {
		if i.centerY != j.centerY {
			return i.centerY - j.centerY
		} else if i.centerX != j.centerX {
			return i.centerX - j.centerX
		} else {
			return 0
		}
	})

	outStr := ""
	for _, sym := range matchedSymbols {
		outStr += sym.symbol
	}

	return outStr, nil
}

func jsonResponse(w http.ResponseWriter, v any) {
	enableCors(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func respondWithError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	jsonResponse(w, ErrorResponse{
		Error: err.Error(),
	})
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "https://hvzlien.sylvie.fyi")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
}

func translateAlienToSounds(alienText string) string {
	for a, p := range lookup {
		alienText = strings.ReplaceAll(alienText, a, p)
	}
	return alienText
}

func encodeAlienFromEnglish(humanText string) string {
	lowercase := strings.ToLower(humanText)
	re := regexp.MustCompile(`[\w']+|[^\s\w]+|[\s]+`)
	tokens := re.FindAllString(lowercase, -1)

	for i, token := range tokens {
		if ipa, exists := ipaTable[token]; exists {
			tokens[i] = ipa
		}
	}

	sounds := strings.Join(tokens, "")
	for c, r := range secondaryIPAMapping {
		sounds = strings.ReplaceAll(sounds, c, r)
	}

	alien := sounds
	for a, p := range reverseLookup {
		alien = strings.ReplaceAll(alien, a, p)
	}
	return alien
}

func encodeAlienFromFrench(humanText string) string {
	lowercase := strings.ToLower(humanText)
	re := regexp.MustCompile(`[\w']+|[^\s\w]+|[\s]+`)
	tokens := re.FindAllString(lowercase, -1)

	for i, token := range tokens {
		if ipa, exists := frenchTable[token]; exists {
			tokens[i] = ipa
		}
	}

	sounds := strings.Join(tokens, "")
	for c, r := range secondaryIPAMapping {
		sounds = strings.ReplaceAll(sounds, c, r)
	}

	alien := sounds
	for a, p := range reverseLookup {
		alien = strings.ReplaceAll(alien, a, p)
	}
	return alien
}

func Decode(w http.ResponseWriter, r *http.Request) {
	var decodeRequest DecodeRequest
	err := json.NewDecoder(r.Body).Decode(&decodeRequest)
	if err != nil {
		respondWithError(w, err)
		return
	}
	if killed {
		jsonResponse(w, DecodeResponse{Phonetics: "translations currently disabled", AlienText: "translations currently disabled"})
		return
	}
	var translated string
	switch decodeRequest.Type {
	case "text":
		translated = translateAlienToSounds(decodeRequest.Text)
		jsonResponse(w, DecodeResponse{Phonetics: translated, AlienText: decodeRequest.Text})
	case "image":
		imgBytes, err := base64.StdEncoding.DecodeString(decodeRequest.Image)
		if err != nil {
			respondWithError(w, err)
			return
		}
		symbols, err := readImageToSymbols(imgBytes)
		if err != nil {
			respondWithError(w, err)
			return
		}
		translated = translateAlienToSounds(symbols)
		jsonResponse(w, DecodeResponse{Phonetics: translated, AlienText: symbols})
	default:
		respondWithError(w, fmt.Errorf("not a valid decode request type"))
		log.Infof("got invalid decode request type: %s", decodeRequest.Type)
		return
	}

	log.Infof("got decode request type: %s, with content %s", decodeRequest.Type, translated)
}

func EncodeText(w http.ResponseWriter, r *http.Request) {
	var encodeRequest EncodeRequest
	err := json.NewDecoder(r.Body).Decode(&encodeRequest)
	if err != nil {
		respondWithError(w, err)
		return
	}
	if killed {
		jsonResponse(w, EncodeResponse{Text: "translations currently disabled"})
		return
	}
	jsonResponse(w, EncodeResponse{Text: encodeAlienFromEnglish(encodeRequest.Text)})
	log.Infof("got text encode request for: %s", encodeRequest.Text)
}

func EncodeImage(w http.ResponseWriter, r *http.Request) {
	var encodeRequest EncodeRequest
	err := json.NewDecoder(r.Body).Decode(&encodeRequest)
	if err != nil {
		respondWithError(w, err)
		return
	}
	if killed {
		jsonResponse(w, EncodeResponse{Text: "translations currently disabled"})
		return
	}
	translated := encodeAlienFromEnglish(encodeRequest.Text)
	imgBase64, err := renderTextToPNG(translated, "alien.ttf")
	if err != nil {
		respondWithError(w, err)
		return
	}
	jsonResponse(w, EncodeResponse{Image: imgBase64})
	log.Infof("got image encode request for: %s", encodeRequest.Text)
}

func alienToEmojis(alienText string, incudeDiscriminator bool) string {
	for symbol, mapping := range emojiNames {
		if incudeDiscriminator {
			alienText = strings.ReplaceAll(alienText, symbol, fmt.Sprintf("<:%s:%s>", mapping[0], mapping[1]))
		} else {
			alienText = strings.ReplaceAll(alienText, symbol, fmt.Sprintf(":%s:", mapping[0]))
		}
	}
	return alienText
}

func DiscordEnglishToAlienEmojis(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if option, ok := optionMap["text"]; ok {
		msgText := option.StringValue()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "translating human messsage...",
			},
		})

		translated := encodeAlienFromEnglish(msgText)
		emojified := alienToEmojis(translated, true)
		if len(emojified) > 2000 {
			emojified = fmt.Sprintf("output too long by %d chars", len(emojified)-2000)
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &emojified,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no text provided :(",
		},
	})
}

func DiscordEnglishToAlienEmojisFrench(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if option, ok := optionMap["text"]; ok {
		msgText := option.StringValue()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "translating human messsage...",
			},
		})

		translated := encodeAlienFromFrench(msgText)
		emojified := alienToEmojis(translated, true)
		if len(emojified) > 2000 {
			emojified = fmt.Sprintf("output too long by %d chars", len(emojified)-2000)
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &emojified,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no text provided :(",
		},
	})
}

func DiscordEnglishToAlienEmojisRaw(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if option, ok := optionMap["text"]; ok {
		msgText := option.StringValue()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "translating human messsage...",
			},
		})

		translated := encodeAlienFromEnglish(msgText)

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &translated,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no text provided :(",
		},
	})
}

func DiscordAlienUnicodeToEnglish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if option, ok := optionMap["text"]; ok {
		msgText := option.StringValue()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "translating alien messsage...",
			},
		})

		translated := translateAlienToSounds(msgText)

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &translated,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no text provided :(",
		},
	})
}

func KillHttp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	killed = true
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "http disabled",
		},
	})
}

func AliveHttp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	killed = false
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "http enabled",
		},
	})
}

func DiscordEmojiToEnglish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if option, ok := optionMap["text"]; ok {
		msgText := option.StringValue()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "translating alien messsage...",
			},
		})

		for symbol, emoji := range emojiNames {
			msgText = strings.ReplaceAll(msgText, fmt.Sprintf("<:%s:%s>", emoji[0], emoji[1]), symbol)
		}
		translated := translateAlienToSounds(msgText)

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &translated,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no text provided :(",
		},
	})
}

func DiscordDecodeImage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	if option, ok := optionMap["image"]; ok {
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[option.Value.(string)].URL
		res, err := http.DefaultClient.Get(attachmentUrl)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "could not download image",
				},
			})
			return
		}

		imgBytes, err := io.ReadAll(res.Body)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "could not download image",
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "translating alien messsage...",
			},
		})

		symbols, err := readImageToSymbols(imgBytes)
		if err != nil {
			msg := "failed to parse symbols"
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			return
		}
		translated := translateAlienToSounds(symbols)

		msg := fmt.Sprintf("`%s`", translated)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no image :(",
		},
	})
}

func runDiscord() {
	discord, err := discordgo.New("Bot <BotTokenGoHere>")
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	discord.Open()
	defer discord.Close()
	guildID := "762409528779210823"
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "gneep",
			Description: "decode alien text from image",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "image",
					Description: "image to decode",
					Required:    true,
				},
			},
		},
		{
			Name:        "gnarp",
			Description: "encodes human words as alien discord emojis",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "text to encode",
					Required:    true,
				},
			},
		},
		{
			Name:        "gnarp-fr",
			Description: "encodes human words as alien discord emojis (french)",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "text to encode (french)",
					Required:    true,
				},
			},
		},
		{
			Name:        "gnarp-raw",
			Description: "encodes human words as alien (copy-able as text)",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "text to encode",
					Required:    true,
				},
			},
		},
		{
			Name:        "glorp",
			Description: "decodes alien raw text as human sounds",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "alien text (rendered as unicode emojis)",
					Required:    true,
				},
			},
		},
		{
			Name:        "glarp",
			Description: "decodes alien discord emojis to human sounds",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "text",
					Description: "alien emojis",
					Required:    true,
				},
			},
		},
		{
			Name:        "kill",
			Description: "disables http server",
		},
		{
			Name:        "alive",
			Description: "enables http server",
		},
	}
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"gneep":     DiscordDecodeImage,
		"gnarp":     DiscordEnglishToAlienEmojis,
		"gnarp-fr":  DiscordEnglishToAlienEmojisFrench,
		"gnarp-raw": DiscordEnglishToAlienEmojisRaw,
		"glorp":     DiscordAlienUnicodeToEnglish,
		"glarp":     DiscordEmojiToEnglish,
		"kill":      KillHttp,
		"alive":     AliveHttp,
	}
	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := discord.ApplicationCommandCreate(discord.State.User.ID, guildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	fmt.Println("bot running")

	for {
		time.Sleep(time.Second * 1)
	}
}

func main() {
	reverseLookup = map[string]string{}
	for a, p := range lookup {
		reverseLookup[p] = a
	}

	http.HandleFunc("/api/v1/decode", Decode)
	http.HandleFunc("/api/v1/encode/text", EncodeText)
	http.HandleFunc("/api/v1/encode/image", EncodeImage)
	fs := http.FileServer(http.Dir("./frontend/dist"))
	http.Handle("/", fs)

	loadIPA("ipa/en_US.txt")
	loadFrench("ipa/fr_FR.txt")

	go runDiscord()
	fmt.Println("starting server...")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
