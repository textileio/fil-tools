package client

import (
	"os"
	"testing"

	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"
	"github.com/textileio/powergate/health/rpc"
)

const (
	tmpDir = "/tmp/powergate/clienttest"
)

func TestMain(m *testing.M) {
	if err := os.RemoveAll(tmpDir); err != nil {
		panic(err)
	}
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			panic("can't create temp dir: " + err.Error())
		}
	}
	logging.SetAllLoggers(logging.LevelError)
	os.Exit(m.Run())
}

func TestCheck(t *testing.T) {
	c, done := setupHealth(t)
	defer done()
	res, err := c.Check(ctx)
	require.NoError(t, err)
	require.Empty(t, res.Messages)
	require.Equal(t, rpc.Status_STATUS_OK, res.Status)
}

func setupHealth(t *testing.T) (*Health, func()) {
	serverDone := setupServer(t, defaultServerConfig(t))
	conn, done := setupConnection(t)
	return &Health{client: rpc.NewRPCServiceClient(conn)}, func() {
		done()
		serverDone()
	}
}
