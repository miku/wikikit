package main

// An example streaming XML parser.

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

var titleFilter = flag.String("f", "", "only filter pages that contain the given text (no regex)")
var filter, _ = regexp.Compile("^file:.*|^talk:.*|^special:.*|^wikipedia:.*|^wiktionary:.*|^user:.*|^user_talk:.*")

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

func CanonicalizeTitle(title string) string {
	can := strings.ToLower(title)
	can = strings.Replace(can, " ", "_", -1)
	can = url.QueryEscape(can)
	return can
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: wptojson WIKIPEDIA-XML-DUMP")
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
					if strings.Contains(p.Title, *titleFilter) {
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
