package knirvmodels

import (
	"os"
	"os/signal"
)

func signalNotify(ch chan<- os.Signal, sigs ...os.Signal) { signal.Notify(ch, sigs...) }
