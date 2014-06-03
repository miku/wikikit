all:
	go fmt wptoldj.go
	go build wptoldj.go

clean:
	rm -f wptoldj
