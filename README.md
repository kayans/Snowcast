# Snowcast
This project is a simplified version of an Internet radio station that includes a server and client. The duty of the server is to keep track of subscribed clients and send them streaming data at a precise rate (16KiB/s). Clients provide a simple user's interface and are capable of joining/switching/leaving stations, and even "playing" the music.


## Major Design Decisions
### Interface
Commands and replys are treated as messages in this project. They all can mashalling structures to bytes, and unmarshalling structures from bytes. So, it's quite easy to write some functions to send and receive messages.


### Handshake
In the first place, a `Hello` command followed by a `Welcome` reply is called a handshake. Before the server and the client can start real constructive communication, they need to complete the handshake. To simplify the code, both the server and the client will handle a `Hello` command and a `Welcome` reply only during the handshake. After that, they will be regarded as invalid commands or unknown replys.


### Concurrency
* start a goroutines for each clien connectiosn
* create separate goroutines to handle keyboard input
* use WaitGroup to wait for all connections to close
* use RWMutex to make sure only one goroutine can modify the client list at a time
* use channels to send messages

## Extra Credit
* Add a command which requests a listing of what each of the stations is currently playing

## Server CLI
`p` -> print to stdout a list of its stations along with the listeners that are connected to each one

`p <file>` -> write the list of stations to the specified file

`q` close all connections and exit 


## Client CLI
`q` -> close all connections and exit

`stations` -> requests a listing of what each of the stations is currently playing


## Makefile
### Build
`make all` -> build the server, the client and the control

`make snowcast_server` -> build the server

`make snowcast_control` -> build the control

`make clesnowcast_listeneran` -> build listener

### Clean
`make clean` -> remove old file

### Run
`make server` -> build and run the server with default arguments

`make control` -> build run the control with default arguments

`make listener` -> build run the listener with default arguments

### Test
`make test` -> test the project using the built-in tester

`make testfast` -> test the project using the built-in tester in fail-fast mode

`make testserver` -> test the server using the built-in tester in fail-fast mode

`make testcontrol` -> test the control using the built-in tester in fail-fast mode

