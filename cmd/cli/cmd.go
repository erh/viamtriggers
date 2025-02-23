package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/generic"
	"go.viam.com/utils/rpc"

	"github.com/erh/viamtriggers"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func readJSONFromFile(fn string, where any) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}

	jsonData, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, where)
}

func realMain() error {
	ctx := context.Background()
	logger := logging.NewLogger("cli")

	machineName := ""
	apiKey := ""
	apiKeyId := ""

	debug := false

	flag.BoolVar(&debug, "debug", debug, "")

	flag.StringVar(&machineName, "machine", machineName, "")
	flag.StringVar(&apiKey, "api-key", apiKey, "")
	flag.StringVar(&apiKeyId, "api-key-id", apiKeyId, "")

	flag.Parse()

	if debug {
		logger.SetLevel(logging.DEBUG)
	}

	conf := viamtriggers.Config{}
	err := readJSONFromFile(flag.Arg(0), &conf)
	if err != nil {
		return err
	}

	machine, err := client.New(
		ctx,
		machineName,
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials(
			apiKeyId,
			rpc.Credentials{
				Type:    rpc.CredentialsTypeAPIKey,
				Payload: apiKey,
			})),
	)
	if err != nil {
		return err
	}
	defer machine.Close(ctx)

	deps := resource.Dependencies{}

	names := machine.ResourceNames()
	for _, n := range names {
		r, err := machine.ResourceByName(n)
		if err != nil {
			return err
		}
		deps[n] = r
	}

	mm, err := viamtriggers.NewMovementMotion(ctx, deps, generic.Named("foo"), &conf, logger)
	defer mm.Close(ctx)

	time.Sleep(2 * time.Minute)

	return nil
}
