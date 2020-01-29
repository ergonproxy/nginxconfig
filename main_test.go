package main

import "context"

// make sure that vince started with configuration and calls kase after vince is
// up and running.
//
// This will make sure all resources are cleared/released before the function exits.
func runTest(v *vinceConfiguration, kase func(context.Context)) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go func() {
		err = startEverything(ctx, v, func() {
			ready <- struct{}{}
		})
	}()
	<-ready
	kase(ctx)
	return
}
