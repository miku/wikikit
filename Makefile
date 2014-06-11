all:
	go fmt wikikit.go
	go build wikikit.go

clean:
	rm -f wikikit
