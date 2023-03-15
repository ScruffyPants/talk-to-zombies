# Winter is coming
This is a solution to Mysterium Network's "winter is coming" quest
called talk to zombies. The original task can be found: [here](https://github.com/mysteriumnetwork/winter-is-coming/blob/master/quests/Talk_to_Zombies.md#communication-channel-specification)

### The solution
Service is written in Go. It has three main domains:

* Communication - a wrapper for everything communication between business logic and user related
it is a generic wrapper for any implementation of bidirectional communication.
* Game - component for handling the game business logic, it listens to events from communication service
and executes commands appropriately. It also handles game instances.
* Player - component which bridges the connection between a communication connection and player inside a game.

Websockets were chosen for API as a bidirectional communication channel.
Application is covered using functional tests which start the service
start multiple games in parallel, have multiple players join each game,
fire missing shots and correct shots. All the outcomes are checked and validated
to make sure the expected behaviour is met.

All the requirements of the application were implemented with one small change:
to support multiple players in a game, a "JOIN" command was added + the "START"
command returns an ID of the game which can be used to JOIN that game.

### Running the service

The service should just run by executing `go run main.go` in the root of the project.
There are also some flags which can be adjusted, you can see all of them with 
`go run main.go --help`