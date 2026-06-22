package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"plugin-dynamic-conditions/internal/acproto"
	"plugin-dynamic-conditions/internal/config"
	"plugin-dynamic-conditions/internal/dynamic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: plugin-dynamic-conditions <config1.cfg> ...")
		os.Exit(1)
	}
	managers := make([]*acproto.Manager, 0, len(os.Args)-1)
	for _, path := range os.Args[1:] {
		params, err := config.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		manager := acproto.NewManager(params.Plugin)
		manager.AddPlugin(dynamic.NewPlugin(params.Weather))
		if err := manager.Connect(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		managers = append(managers, manager)
	}
	if len(managers) == 1 {
		fmt.Println("> Server plugin is running. Press Ctrl+C to close.")
	} else {
		fmt.Printf("> %d server plugins are running. Press Ctrl+C to close.\n", len(managers))
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	for _, m := range managers {
		m.Close()
	}
}
