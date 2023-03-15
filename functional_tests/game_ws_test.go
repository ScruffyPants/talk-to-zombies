package functional_tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

const (
	numberOfMissedShots   = 5
	concurrentGames       = 10
	numberOfJoinedPlayers = 4
)

var (
	timeoutBetweenConcurrentGameTestStart = 10 * time.Millisecond
	timeoutBetweenShootCommands           = 10 * time.Millisecond
	timeoutBetweenJoinCommands            = 10 * time.Millisecond
)

func TestGameWebsocketAPI(t *testing.T) {
	wg := sync.WaitGroup{}

	for i := 0; i < concurrentGames; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			testSingleWebsocketGame(t)
		}()

		time.Sleep(timeoutBetweenConcurrentGameTestStart)
	}

	wg.Wait()

}

func testSingleWebsocketGame(t *testing.T) {
	wsConnection := makeWSConnection(t)

	defer func() {
		require.NoError(t, wsConnection.Close(websocket.StatusNormalClosure, "disconnect"))
	}()

	name := uuid.NewString()

	gameID := testStartGame(t, wsConnection, name)

	testJoinGame(t, gameID)

	testMissedShots(t, wsConnection, name)

	xCord, yCord := testWalkMessage(t, wsConnection)

	testHitShot(t, wsConnection, name, xCord, yCord)
}

func makeWSConnection(t *testing.T) *websocket.Conn {
	c, _, err := websocket.Dial(context.Background(), fmt.Sprintf("ws://%s", testWSAddress), nil)
	require.NoError(t, err)

	return c
}

func testStartGame(t *testing.T, wsConnection *websocket.Conn, name string) string {
	testWriteWSMessageWithTimeout(t, wsConnection, fmt.Sprintf("START %s", name))
	message := testReadWSMessageWithTimeout(t, wsConnection)

	splitMessage := strings.Split(message, " ")
	require.Len(t, splitMessage, 2)

	assert.Equal(t, "GAME", splitMessage[0])
	assert.NotEmpty(t, splitMessage[1])

	return splitMessage[1]
}

func testJoinGame(t *testing.T, gameID string) {
	wg := sync.WaitGroup{}

	for i := 0; i < numberOfJoinedPlayers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			wsConnection := makeWSConnection(t)
			defer func() {
				require.NoError(t, wsConnection.Close(websocket.StatusNormalClosure, "disconnect"))
			}()

			name := uuid.NewString()

			testWriteWSMessageWithTimeout(t, wsConnection, fmt.Sprintf("JOIN %s %s", gameID, name))

			testMissedShots(t, wsConnection, name)
		}()

		time.Sleep(timeoutBetweenJoinCommands)
	}

	wg.Wait()
}

func testMissedShots(t *testing.T, wsConnection *websocket.Conn, name string) {
	for i := 0; i < numberOfMissedShots; i++ {
		testWriteWSMessageWithTimeout(t, wsConnection, fmt.Sprintf("SHOOT 500 500"))
		gotDifferentMessage := false

		for {
			message := testReadWSMessageWithTimeout(t, wsConnection)

			splitMessage := strings.Split(message, " ")
			require.Greater(t, len(splitMessage), 0)

			if splitMessage[0] != "BOOM" {
				if gotDifferentMessage {
					t.Fatal("failed to get missed shot response")
				}

				// Dealing with possibility of getting WALK (or different) message
				// before getting response to BOOM message, however assuming the next
				// message will be response to the BOOM message
				gotDifferentMessage = true
				continue
			}

			require.Len(t, splitMessage, 3)
			assert.Equal(t, "BOOM", splitMessage[0])
			assert.Equal(t, name, splitMessage[1])
			assert.Equal(t, "0", splitMessage[2])

			break
		}

		time.Sleep(timeoutBetweenShootCommands)
	}
}

func testWalkMessage(t *testing.T, wsConnection *websocket.Conn) (int, int) {
	message := testReadWSMessageWithTimeout(t, wsConnection)

	splitMessage := strings.Split(message, " ")
	require.Len(t, splitMessage, 4)

	assert.Equal(t, "WALK", splitMessage[0])

	xCord, err := strconv.Atoi(splitMessage[2])
	require.NoError(t, err)

	yCord, err := strconv.Atoi(splitMessage[3])
	require.NoError(t, err)

	return xCord, yCord
}

func testHitShot(t *testing.T, wsConnection *websocket.Conn, name string, xCord int, yCord int) {
	testWriteWSMessageWithTimeout(t, wsConnection, fmt.Sprintf("SHOOT %d %d", xCord, yCord))

	message := testReadWSMessageWithTimeout(t, wsConnection)

	splitMessage := strings.Split(message, " ")
	require.Len(t, splitMessage, 4)

	assert.Equal(t, "BOOM", splitMessage[0])
	assert.Equal(t, name, splitMessage[1])
	assert.Equal(t, "1", splitMessage[2])
}

func testWriteWSMessageWithTimeout(t *testing.T, wsConnection *websocket.Conn, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := wsConnection.Write(ctx, websocket.MessageText, []byte(message))
	require.NoError(t, err)
}

func testReadWSMessageWithTimeout(t *testing.T, wsConnection *websocket.Conn) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, messageBytes, err := wsConnection.Read(ctx)
	require.NoError(t, err)

	return string(messageBytes)
}
