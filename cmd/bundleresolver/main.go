package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var version = "0.1.0"

// Field represents the output data fields.
type Field string

const (
	FieldName      Field = "name"
	FieldPublisher Field = "publisher"
	FieldURL       Field = "url"
)

var allowedFields = []Field{FieldName, FieldPublisher, FieldURL}
var fieldSet map[Field]struct{}

func init() {
	fieldSet = make(map[Field]struct{}, len(allowedFields))
	for _, f := range allowedFields {
		fieldSet[f] = struct{}{}
	}
	log.SetFlags(0)
}

func main() {
	var fieldsCSV string
	var showVersion bool
	var showHeader bool

	flag.StringVar(&fieldsCSV, "fields", "name,publisher,url", "Comma-separated list of fields to output (allowed: name,publisher,url)")
	flag.StringVar(&fieldsCSV, "f", "name,publisher,url", "Alias of --fields")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&showHeader, "header", true, "Print header row as first line (use --header=false to disable)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Bundle Resolver\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] < <input>\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nInput: lines of either numeric iOS App IDs or Android package names (with dots).\n")
	}
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	fields, err := parseFields(fieldsCSV)
	if err != nil {
		log.Fatalf("invalid --fields: %v", err)
	}

	if err := process(os.Stdin, os.Stdout, fields, showHeader); err != nil {
		log.Fatalf("error: %v", err)
	}
}

var (
	reIOS     = regexp.MustCompile(`^[0-9]+$`)
	reAndroid = regexp.MustCompile(`^[A-Za-z0-9_]+(\.[A-Za-z0-9_]+)+$`)
)

type record struct {
	Name      string
	Publisher string
	URL       string
}

func parseFields(csv string) ([]Field, error) {
	parts := strings.Split(csv, ",")
	res := make([]Field, 0, len(parts))
	seen := map[Field]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f := Field(p)
		if _, ok := fieldSet[f]; !ok {
			return nil, fmt.Errorf("unknown field %q", p)
		}
		// allow duplicates? Probably not useful; keep order but de-dup
		if seen[f] {
			continue
		}
		seen[f] = true
		res = append(res, f)
	}
	if len(res) == 0 {
		return nil, errors.New("no valid fields specified")
	}
	return res, nil
}

func process(r io.Reader, w io.Writer, fields []Field, header bool) error {
	s := bufio.NewScanner(r)
	// Print header immediately if requested so it's always the first line in output.
	if header {
		printHeader(w, fields)
	}
	for s.Scan() {
		raw := s.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			// Preserve alignment: output an empty row corresponding to the blank input line.
			printFields(w, record{}, fields)
			continue
		}
		rec, err := resolve(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve %q: %v\n", line, err)
			// Still emit placeholder row; rec may have URL (canonical) or be empty.
		}
		printFields(w, rec, fields)
	}
	return s.Err()
}

func printHeader(w io.Writer, fields []Field) {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = string(f)
	}
	fmt.Fprintln(w, strings.Join(names, "\t"))
}

func printFields(w io.Writer, rec record, fields []Field) {
	cols := make([]string, len(fields))
	for i, f := range fields {
		var val string
		switch f {
		case FieldName:
			val = rec.Name
		case FieldPublisher:
			val = rec.Publisher
		case FieldURL:
			val = rec.URL
		}
		cols[i] = sanitize(val)
	}
	fmt.Fprintln(w, strings.Join(cols, "\t"))
}

// sanitize removes tabs and newlines to preserve TSV integrity.
func sanitize(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// resolve decides platform and fetches metadata.
func resolve(id string) (record, error) {
	if reIOS.MatchString(id) {
		return fetchIOS(id)
	}
	if reAndroid.MatchString(id) {
		return fetchAndroid(id)
	}
	return record{}, fmt.Errorf("cannot detect platform for %q", id)
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func fetchIOS(appID string) (record, error) {
	lookup := func(country string) (record, error) {
		url := fmt.Sprintf("https://itunes.apple.com/lookup?id=%s", appID)
		if country != "" {
			url += "&country=" + country
		}
		resp, err := httpClient.Get(url)
		if err != nil {
			return record{}, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return record{}, fmt.Errorf("status %s", resp.Status)
		}
		var payload struct {
			ResultCount int `json:"resultCount"`
			Results     []struct {
				TrackName    string `json:"trackName"`
				SellerName   string `json:"sellerName"`
				TrackViewURL string `json:"trackViewUrl"`
			} `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return record{}, err
		}
		if payload.ResultCount == 0 || len(payload.Results) == 0 {
			return record{}, fmt.Errorf("not found")
		}
		res := payload.Results[0]
		urlOut := res.TrackViewURL
		if urlOut == "" {
			// if TrackViewURL missing we'll still build canonical later
		}
		// Normalize to canonical short form per README
		canonical := fmt.Sprintf("https://apps.apple.com/app/id%s", appID)
		return record{Name: res.TrackName, Publisher: res.SellerName, URL: canonical}, nil
	}

	// 1st try: no country (Apple often defaults to US)
	rec, err := lookup("")
	if err == nil {
		return rec, nil
	}
	// Fallback to jp (common case for JP-only apps)
	jpRec, errJP := lookup("jp")
	if errJP == nil {
		return jpRec, nil
	}
	// Return the original error but still provide constructed URL
	return record{URL: fmt.Sprintf("https://apps.apple.com/app/id%s", appID)}, err
}

func fetchAndroid(pkg string) (record, error) {
	storeURL := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s", pkg)
	resp, err := httpClient.Get(storeURL)
	if err != nil {
		return record{URL: storeURL}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return record{URL: storeURL}, fmt.Errorf("status %s", resp.Status)
	}
	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return record{URL: storeURL}, err
	}
	name := strings.TrimSpace(doc.Find("h1 span").First().Text())
	if name == "" { // fallback to title tag
		title := strings.TrimSpace(doc.Find("title").Text())
		if strings.Contains(title, " - Apps on Google Play") {
			name = strings.TrimSuffix(title, " - Apps on Google Play")
		}
	}
	publisher := strings.TrimSpace(doc.Find("div[itemprop='author'] a span").First().Text())
	if publisher == "" {
		// New Play Store layout fallback (may change frequently)
		publisher = strings.TrimSpace(doc.Find("a[href^='/store/apps/dev'] span").First().Text())
	}
	return record{Name: name, Publisher: publisher, URL: storeURL}, nil
}

// (removed) previous naive JSON extractor replaced with proper json.Decoder usage

// Guarantee deterministic ordering of allowedFields (unused yet but keep for future list printing)
func init() {
	// sort just in case
	sort.Slice(allowedFields, func(i, j int) bool { return allowedFields[i] < allowedFields[j] })
}
