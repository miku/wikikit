wikikit
=======

Convert Wikipedia XML dump into JSON.

    $ wikikit WIKIPEDIA-XML-DUMP

[![Gobuild download](http://gobuild.io/badge/github.com/miku/wikikit/download.png)](http://gobuild.io/download/github.com/miku/wikikit)

Output:

    {
       "redirect" : {
          "title" : ""
       },
       "text" : "{{Red ....",
       "ctitle" : "anarchism",
       "title" : "Anarchism"
    }

Extract category information only:

    $ wikikit -c Category WIKIPEDIA-XML-DUMP

Extract authority data only:

    $ wikikit -a WIKIPEDIA-XML-DUMP

De-literalize JSON text from wikidata pages/articles dumps:

    $ wikikit -d WIKIDATA-XML-DUMP
