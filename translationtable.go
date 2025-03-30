package main

import (
	"os"
	"strings"
)

var emojiNames = map[string][2]string{
	"☊": {"hvz_36", "1354569163733078071"},
	"☋": {"hvz_41", "1354569154367062016"},
	"☌": {"hvz_29", "1354569230535758105"},
	"☍": {"hvz_18", "1354569345707016324"},
	"☎": {"hvz_28", "1354569233270308935"},
	"☏": {"hvz_27", "1354569235875238112"},
	"☚": {"hvz_16", "1354569349368905789"},
	"☛": {"hvz_15", "1354569351252017182"},
	"☜": {"hvz_14", "1354569352703381604"},
	"☝": {"hvz_13", "1354569354355806389"},
	"☞": {"hvz_10", "1354569383241973774"},
	"☟": {"hvz_19", "1354569344239276063"},
	"☀": {"hvz_2", "1354574504973832332"},
	"☁": {"hvz_1", "1354574506089385988"},
	"☂": {"hvz_11", "1354569381388226560"},
	"☃": {"hvz_39", "1354569157433229482"},
	"☄": {"hvz_37", "1354569161245851859"},
	"★": {"hvz_40", "1354569155914895450"},
	"☆": {"hvz_35", "1354569166278885517"},
	"☇": {"hvz_20", "1354569342620008498"},
	"☈": {"hvz_33", "1354569170867716217"},
	"☉": {"hvz_9", "1354569384693207080"},
	"☐": {"hvz_26", "1354569237901082816"},
	"☑": {"hvz_31", "1354569218565345290"},
	"☒": {"hvz_30", "1354569222151471315"},
	"☓": {"hvz_4", "1354569395615305910"},
	"☔": {"hvz_25", "1354569240484647134"},
	"☕": {"hvz_24", "1354569242082541670"},
	"☖": {"hvz_23", "1354569245769601305"},
	"☗": {"hvz_21", "1354569341219373317"},
	"☘": {"hvz_22", "1354569333388607539"},
	"☙": {"hvz_7", "1354569388048777358"},
	"☠": {"hvz_17", "1354569347762225323"},
	"☡": {"hvz_6", "1354569391190053034"},
	"☢": {"hvz_38", "1354569158985121792"},
	"☣": {"hvz_5", "1354569393094525149"},
	"☤": {"hvz_3", "1354569397531836500"},
	"☥": {"hvz_12", "1354569379869622353"},
	"☦": {"hvz_8", "1354569386467524741"},
	"☧": {"hvz_32", "1354569172813615258"},
	"☨": {"hvz_34", "1354569168531492906"},
}

var lookup = map[string]string{
	"☂": " ",

	"☀": "ˌ",
	"☁": "ˈ",

	"☑": "i",
	"☒": "ɪ",
	"☌": "ɛ",
	"☃": "a",
	"★": "æ",
	"☋": "ə",
	"☢": "ɐ",
	"☟": "u",
	"☇": "o",
	"☍": "ɜ",

	"☏": "ɡ",
	"☞": "t",
	"☚": "p",
	"☎": "f",
	"☡": "v",
	"☉": "ð",
	"☜": "s",
	"☤": "z",
	"☖": "m",
	"☗": "n",
	"☘": "ŋ",
	"☐": "h",
	"☕": "l",
	"☛": "r",
	"☣": "w",
	"☆": "b",
	"☦": "θ",
	"☔": "k",
	"☠": "ʊ",
	"☈": "d",
	"☙": "ʌ",
	"☥": "ʒ",
	"☝": "ʃ",

	"☊": "e",
	"☓": "j",

	"☄": "ɑ",

	"☨": "ʧ",
	"☧": "ʤ",
}
var reverseLookup map[string]string
var ipaTable map[string]string
var frenchTable map[string]string

var secondaryIPAMapping = map[string]string{
	"ɫ": "l",
	"ɔ": "o",
	"ɹ": "r",
	"g": "ɡ",
	"ɝ": "r",
	"y": "i",
	"q": "k",
}

func loadFrench(path string) error {
	filedata, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filestring := string(filedata)
	frenchTable = map[string]string{}
	for _, line := range strings.Split(filestring, "\n") {
		chunks := strings.SplitN(line, "\t", 2)
		if len(chunks) > 1 {
			word := chunks[0]
			options := strings.Split(chunks[1], ", ")
			option := strings.TrimSuffix(strings.TrimPrefix(options[0], "/"), "/")
			frenchTable[word] = option
		}
	}
	return nil
}

func loadIPA(path string) error {
	filedata, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filestring := string(filedata)
	ipaTable = map[string]string{}
	for _, line := range strings.Split(filestring, "\n") {
		chunks := strings.SplitN(line, "\t", 2)
		if len(chunks) > 1 {
			word := chunks[0]
			options := strings.Split(chunks[1], ", ")
			option := strings.TrimSuffix(strings.TrimPrefix(options[0], "/"), "/")
			ipaTable[word] = option
		}
	}
	return nil
}
