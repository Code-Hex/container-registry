.PHONY: clean
clean:
	rm -rf testdata/library
build:
	go build -o bin/registry -trimpath -ldflags "-w -s"
