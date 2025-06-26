package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/kljensen/snowball"
	"github.com/ledongthuc/pdf"
)

var stopwords = map[string]bool{
	"el": true, "la": true, "de": true, "y": true, "que": true, "en": true,
	"a": true, "los": true, "se": true, "del": true, "las": true, "por": true,
	"un": true, "para": true, "con": true, "no": true, "una": true,
	// Add more stopwords as needed
}

func extractTextFromPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	text, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	buf := new(strings.Builder)
	_, err = io.Copy(buf, text)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
func preprocess(text string) []string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "\n", " ")
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || r == 'á' || r == 'é' || r == 'í' || r == 'ó' || r == 'ú' || r == 'ñ')
	})

	var result []string
	for _, word := range tokens {
		if stopwords[word] {
			continue
		}
		stemmed, _ := snowball.Stem(word, "spanish", true)
		result = append(result, stemmed)
	}
	return result
}

func tfidf(sentences [][]string) ([][]float64, map[string]int, map[int]int) {
	vocab := map[string]int{}
	for _, sent := range sentences {
		for _, word := range sent {
			if _, exists := vocab[word]; !exists {
				vocab[word] = len(vocab)
			}
		}
	}

	docCount := len(sentences)
	df := make(map[int]int)
	vectors := make([][]float64, docCount)

	for i, sent := range sentences {
		vec := make([]float64, len(vocab))
		termSeen := map[int]bool{}
		for _, word := range sent {
			idx := vocab[word]
			vec[idx]++
			termSeen[idx] = true
		}
		for idx := range termSeen {
			df[idx]++
		}
		vectors[i] = vec
	}

	for _, vec := range vectors {
		for j := range vec {
			if vec[j] > 0 {
				vec[j] *= math.Log(float64(docCount) / float64(1+df[j]))
			}
		}
		normalize(vec)
	}

	return vectors, vocab, df
}

func normalize(vec []float64) {
	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for i := range vec {
		vec[i] /= norm
	}
}

func cosine(a, b []float64) float64 {
	sumProd, sumA, sumB := 0.0, 0.0, 0.0
	for i := range a {
		sumProd += a[i] * b[i]
		sumA += a[i] * a[i]
		sumB += b[i] * b[i]
	}
	if sumA == 0 || sumB == 0 {
		return 0
	}
	return sumProd / (math.Sqrt(sumA) * math.Sqrt(sumB))
}

func splitIntoSentences(text string) []string {
	abbreviations := map[string]bool{"art.": true, "arts.": true, "arts": true, "etc.": true, "sr.": true, "sra.": true, "dr.": true}
	raw := strings.Split(text, ".")
	var sentences []string
	for i := range raw {
		s := strings.TrimSpace(raw[i])
		if i > 0 && len(sentences) > 0 {
			prev := strings.TrimSpace(sentences[len(sentences)-1])
			if len(prev) > 0 && abbreviations[strings.ToLower(prev[strings.LastIndex(prev, " ")+1:])+"."] {
				sentences[len(sentences)-1] += ". " + s
				continue
			}
		}
		sentences = append(sentences, s)
	}
	return sentences
}

func main() {
	text, err := extractTextFromPDF("test.pdf")
	if err != nil {
		log.Fatal(err)
	}

	sentences := splitIntoSentences(text)
	filtered := make([]string, 0, len(sentences))
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 20 {
			filtered = append(filtered, s)
		}
	}

	var processed [][]string
	for _, s := range filtered {
		processed = append(processed, preprocess(s))
	}

	vectors, vocab, df := tfidf(processed)

	fmt.Println("Pregunta en español:")
	reader := bufio.NewReader(os.Stdin)
	query, _ := reader.ReadString('\n')
	query = strings.TrimSpace(query)

	qVec := make([]float64, len(vocab))
	for _, word := range preprocess(query) {
		if idx, ok := vocab[word]; ok {
			qVec[idx]++
		}
	}

	totalDocs := float64(len(processed))
	for _, idx := range vocab {
		if qVec[idx] > 0 {
			qVec[idx] *= math.Log(totalDocs / float64(1+df[idx]))
		}
	}
	normalize(qVec)

	type result struct {
		Score float64
		Index int
		Text  string
	}
	var results []result
	for i, v := range vectors {
		results = append(results, result{cosine(v, qVec), i, filtered[i]})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	fmt.Println("Respuesta relevante:")
	if len(results) > 0 && results[0].Score > 0.2 {
		fmt.Printf("- %.2f: %s\n", results[0].Score, strings.TrimSpace(results[0].Text))
	}
}
