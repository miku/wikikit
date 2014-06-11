// Convert Wikipedia XML dump to JSON or extract categories
package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const AppVersion = "1.0.6"

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

type Page struct {
	Title          string   `xml:"title" json:"title"`
	CanonicalTitle string   `xml:"ctitle" json:"ctitle"`
	Redir          Redirect `xml:"redirect" json:"redirect"`
	Text           string   `xml:"revision>text" json:"text"`
}

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

func main() {

	version := flag.Bool("v", false, "prints current version and exits")
	extractCategories := flag.String("c", "", "only extract categories TSV(page, category)")
	extractAuthorityData := flag.Bool("a", false, "only extract authority data (Normdaten)")
	decodeWikiData := flag.Bool("d", false, "decode the text key value")
	filter, _ := regexp.Compile("^file:.*|^talk:.*|^special:.*|^wikipedia:.*|^wiktionary:.*|^user:.*|^user_talk:.*")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExtract and convert things from wikipedia/wikidata XML dumps.\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *extractCategories != "" && *extractAuthorityData {
		fmt.Println("It's either -a or -c")
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
	inputFile := flag.Args()[0]

	xmlFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)
	var inElement string
	categoryPattern := regexp.MustCompile(`\[\[` + *extractCategories + `:([^\[]+)\]\]`)
	authorityDataPattern := regexp.MustCompile(`(?mi){{Normdaten[^}]*}}`)

	// for wikidata
	var container interface{}

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

				// Do some stuff with the page.
				p.CanonicalTitle = CanonicalizeTitle(p.Title)
				m := filter.MatchString(p.CanonicalTitle)
				if !m && p.Redir.Title == "" {
					if *extractCategories != "" {
						result := categoryPattern.FindAllStringSubmatch(p.Text, -1)
						for _, value := range result {
							// replace anything after a |
							category := strings.TrimSpace(value[1])
							firstIndex := strings.Index(category, "|")
							if firstIndex != -1 {
								category = category[0:firstIndex]
							}
							fmt.Printf("%s\t%s\n", p.Title, category)
						}
					} else if *extractAuthorityData {
						result := authorityDataPattern.FindString(p.Text)
						if result != "" {
							// https://cdn.mediacru.sh/JsdjtGoLZBcR.png
							result = strings.Replace(result, "\t", "", -1)
							fmt.Printf("%s\t%s\n", p.Title, result)
						}
					} else if *decodeWikiData {
						json.Unmarshal([]byte(p.Text), &container)
						parsed := WikidataPage{Title: p.Title,
							CanonicalTitle: p.CanonicalTitle,
							Content:        container,
							Redir:          p.Redir}
						b, err := json.Marshal(parsed)
						if err != nil {
							os.Exit(2)
						}
						fmt.Println(string(b))
					} else {
						b, err := json.Marshal(p)
						if err != nil {
							os.Exit(2)
						}
						fmt.Println(string(b))
					}
				}
			}
		default:
		}
	}
}