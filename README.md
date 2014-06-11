wptoldj
=======

Convert Wikipedia XML dump into JSON.

    $ wptoldj WIKIPEDIA-XML-DUMP

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

    $ wptoldj.go -c Category WIKIPEDIA-XML-DUMP

Extract authority data only:

    $ wptoldj.go -a WIKIPEDIA-XML-DUMP
