.PHONY: clean
clean:
	rm -rf testdata/*
build:
	go build -o bin/registry -trimpath -ldflags "-w -s"
