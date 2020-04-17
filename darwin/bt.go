package darwin

/*
// See cutil.go for C compiler flags.
#import "bt.h"
*/
import "C"

import (
	"fmt"
	"log"
)

type BTState struct {
	Enabled bool
	Msg     string
}

func StartBTLoop(ch chan BTState) error {
	C.bt_start()

	// Block until initial state change.  If Bluetooth is disabled, fail
	// immediately.
	state := <-ch
	if !state.Enabled {
		return fmt.Errorf("failed to start Bluetooth client: %s", state.Msg)
	}

	// Listen for and log subsequent state changes in the background.
	go func() {
		for {
			state, ok := <-ch
			if !ok {
				return
			}

			log.Printf("Bluetooth state change: %s", state.Msg)
		}
	}()

	return nil
}

func StopBTLoop() {
	C.bt_stop()
}
