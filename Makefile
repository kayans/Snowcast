all: snowcast_server snowcast_control snowcast_listener

clean:
	rm snowcast_server snowcast_control snowcast_listener

snowcast_server: ./cmd/server/main.go ./pkg/protocol/*.go ./pkg/kit/*.go
	go build -o $@ $<

snowcast_control: ./cmd/control/main.go ./pkg/protocol/*.go ./pkg/kit/*.go
	go build -o $@ $<

snowcast_listener: ./cmd/listener/main.go ./pkg/protocol/*.go ./pkg/kit/*.go
	go build -o $@ $<

server: snowcast_server
	./snowcast_server 16800 ./mp3/*

control: snowcast_control
	./snowcast_control localhost 16800 1234

listener: snowcast_listener
	./snowcast_listener 1234 | mpg123 -

test:
	make all
	./util/run_tests all

testfast:
	make all
	./util/run_tests --fail-fast all

testserver:
	make snowcast_server
	./util/run_tests --fail-fast server

testcontrol:
	make snowcast_control
	./util/run_tests --fail-fast control
