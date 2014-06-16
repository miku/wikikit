// Convert Wikipedia XML dump to JSON or extract categories
// Example inputs:
// wikidata: http://dumps.wikimedia.org/wikidatawiki/20140612/wikidatawiki-20140612-pages-articles.xml.bz2
// wikipedia:  http://dumps.wikimedia.org/huwiki/latest/huwiki-latest-pages-articles.xml.bz2
package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
)

const AppVersion = "1.1.1"

// Here is an example article from the Wikipedia XML dump
//
// <page>
// 	<title>Apollo 11</title>
//      <redirect title="Foo bar" />
// 	...
// 	<revision>
// 	...
// 	  <text xml:space="preserve">
// 	  {{Infobox Space mission
// 	  |mission_name=&lt;!--See above--&gt;
// 	  |insignia=Apollo_11_insignia.png
// 	...
// 	  </text>
// 	</revision>
// </page>
//
// Note how the tags on the fields of Page and Redirect below
// describe the XML schema structure.

type Redirect struct {
	Title string `xml:"title,attr" json:"title"`
}

// A page as it occurs on Wikipedia
type Page struct {
	Title          string   `xml:"title" json:"title"`
	CanonicalTitle string   `xml:"ctitle" json:"ctitle"`
	Redir          Redirect `xml:"redirect" json:"redirect"`
	Text           string   `xml:"revision>text" json:"text"`
}

// A page as it occurs on Wikidata, content will be turned from a string
// into a substructure with -d switch
type WikidataPage struct {
	Title          string      `xml:"title" json:"title"`
	CanonicalTitle string      `xml:"ctitle" json:"ctitle"`
	Redir          Redirect    `xml:"redirect" json:"redirect"`
	Content        interface{} `json:"content"`
}

func CanonicalizeTitle(title string) string {
	can := strings.ToLower(title)
	can = strings.Replace(can, " ", "_", -1)
	can = url.QueryEscape(can)
	return can
}

func categoryExtractor(in chan *Page,
	out chan *string,
	filter *regexp.Regexp,
	categoryPattern *regexp.Regexp) {
	var pp *Page
	for {
		// get the page pointer
		pp = <-in
		if pp == nil {
			break
		}
		// get the page
		p := *pp

		// do some stuff with the page
		p.CanonicalTitle = CanonicalizeTitle(p.Title)
		m := filter.MatchString(p.CanonicalTitle)
		if !m && p.Redir.Title == "" {

			// specific to category extraction
			result := categoryPattern.FindAllStringSubmatch(p.Text, -1)
			for _, value := range result {
				// replace anything after a |
				category := strings.TrimSpace(value[1])
				firstIndex := strings.Index(category, "|")
				if firstIndex != -1 {
					category = category[0:firstIndex]
				}

				line := fmt.Sprintf("%s\t%s", p.Title, category)
				out <- &line
			}
		}
	}
}

func authorityDataExtractor(in chan *Page,
	out chan *string,
	filter *regexp.Regexp,
	authorityDataPattern *regexp.Regexp) {
	var pp *Page
	for {
		// get the page pointer
		pp = <-in
		if pp == nil {
			break
		}
		// get the page
		p := *pp

		// do some stuff with the page
		p.CanonicalTitle = CanonicalizeTitle(p.Title)
		m := filter.MatchString(p.CanonicalTitle)
		if !m && p.Redir.Title == "" {

			// specific to category extraction
			result := authorityDataPattern.FindString(p.Text)
			if result != "" {
				// https://cdn.mediacru.sh/JsdjtGoLZBcR.png
				result = strings.Replace(result, "\t", "", -1)
				// fmt.Printf("%s\t%s\n", p.Title, result)
				line := fmt.Sprintf("%s\t%s", p.Title, result)
				out <- &line
			}
		}
	}
}

func wikidataEncoder(in chan *Page,
	out chan *string,
	filter *regexp.Regexp) {

	var container interface{}
	var pp *Page

	for {
		// get the page pointer
		pp = <-in
		if pp == nil {
			break
		}
		// get the page
		p := *pp

		// do some stuff with the page
		p.CanonicalTitle = CanonicalizeTitle(p.Title)
		m := filter.MatchString(p.CanonicalTitle)
		if !m && p.Redir.Title == "" {
			dec := json.NewDecoder(strings.NewReader(p.Text))
			dec.UseNumber()

			if err := dec.Decode(&container); err == io.EOF {
				break
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				continue
			}

			parsed := WikidataPage{Title: p.Title,
				CanonicalTitle: p.CanonicalTitle,
				Content:        container,
				Redir:          p.Redir}

			b, err := json.Marshal(parsed)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(2)
			}
			// fmt.Println(string(b))
			line := string(b)
			out <- &line
		}
	}
}

func vanillaConverter(in chan *Page,
	out chan *string,
	filter *regexp.Regexp) {
	var pp *Page
	for {
		// get the page pointer
		pp = <-in
		if pp == nil {
			break
		}
		// get the page
		p := *pp

		// do some stuff with the page
		p.CanonicalTitle = CanonicalizeTitle(p.Title)
		m := filter.MatchString(p.CanonicalTitle)
		if !m && p.Redir.Title == "" {

			b, err := json.Marshal(p)
			if err != nil {
				os.Exit(2)
			}
			line := string(b)
			out <- &line
		}
	}
}

func collect(lines chan *string) {
	for line := range lines {
		fmt.Println(*line)
	}
}

// Collect output and write to file
func FileCollector(lines chan *string, filename string) {
	output, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := output.Close(); err != nil {
			panic(err)
		}
	}()
	// 4M buffer size
	w := bufio.NewWriter(output)
	for line := range lines {
		_, err = w.WriteString(*line + "\n")
		if err != nil {
			panic(err)
		}
	}
	w.Flush()
}

func main() {

	version := flag.Bool("v", false, "prints current version and exits")
	extractCategories := flag.String("c", "", "only extract categories TSV(page, category), argument is the prefix, e.g. Kategorie or Category, ... ")
	extractAuthorityData := flag.String("a", "", "only extract authority data (Normdaten, Authority control, ...)")
	decodeWikiData := flag.Bool("d", false, "decode the text key value")
	numWorkers := flag.Int("w", runtime.NumCPU(), "number of workers")

	filter, _ := regexp.Compile("^file:.*|^talk:.*|^special:.*|^wikipedia:.*|^wiktionary:.*|^user:.*|^user_talk:.*")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExtract and convert things from wikipedia/wikidata XML dumps.\n")
		fmt.Fprintf(os.Stderr, "\nVersion: %s\n\n", AppVersion)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *extractCategories != "" && *extractAuthorityData != "" {
		fmt.Fprintln(os.Stderr, "it's either -a or -c")
		os.Exit(1)
	}

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	runtime.GOMAXPROCS(*numWorkers)

	inputFile := flag.Args()[0]

	xmlFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer xmlFile.Close()

	// xml decoder
	decoder := xml.NewDecoder(xmlFile)
	var inElement string

	// category pattern depends on the language, e.g. Kategorie or Category, ...
	categoryPattern := regexp.MustCompile(`\[\[` + *extractCategories + `:([^\[]+)\]\]`)
	// Authority data (German only for now)
	authorityDataPattern := regexp.MustCompile(`(?mi){{` + *extractAuthorityData + `[^}]*}}`)

	// the parsed XML pages channel
	pages := make(chan *Page)
	// the strings output channel
	lines := make(chan *string)

	// start the collector
	go collect(lines)

	// start some appropriate workers
	for i := 0; i < *numWorkers; i++ {
		if *extractCategories != "" {
			go categoryExtractor(pages, lines, filter, categoryPattern)
		} else if *extractAuthorityData != "" {
			go authorityDataExtractor(pages, lines, filter, authorityDataPattern)
		} else if *decodeWikiData {
			go wikidataEncoder(pages, lines, filter)
		} else {
			go vanillaConverter(pages, lines, filter)
		}
	}

	for {
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			// If we just read a StartElement token
			inElement = se.Name.Local
			// ...and its name is "page"
			if inElement == "page" {
				var p Page
				// decode a whole chunk of following XML into the
				// variable p which is a Page (se above)
				decoder.DecodeElement(&p, &se)
				pages <- &p
			}
		default:
		}
	}

	// kill workers
	for n := 0; n < *numWorkers; n++ {
		pages <- nil
	}
	close(lines)

}
