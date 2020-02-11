all:
	go generate
	go build

clean:
	rm -rf ownyourtrakt
	rm -rf rice-box.go
