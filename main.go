package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"gitlab.com/oleg.balunenko/logs-converter/config"
	"gitlab.com/oleg.balunenko/logs-converter/converter"
	"gitlab.com/oleg.balunenko/logs-converter/mongo"
)

func main() {

	cfg := config.LoadConfig()

	dbCollection := mongo.Connect(cfg)

	if cfg.DropDB {
		if errDrop := dbCollection.DropCollection(); errDrop != nil {
			log.Fatalf("Failed to drop the collection [%+v.%+v]:%v", dbCollection, dbCollection.Database, errDrop)
		}

	}

	resChan := make(chan *converter.LogModel)
	for l, format := range cfg.LogsFilesList {

		go converter.Start(l, format, resChan)

	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	signal.Notify(signals, syscall.SIGTERM)
	var storedModelsCnt, failedToStoreCnt, totalRecCnt uint64

	for {

		select {
		case <-signals:
			log.Infof("Got UNIX signal, shutting down")
			mongo.CloseConnection(dbCollection)

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 0, 0, ' ', tabwriter.Debug|tabwriter.AlignRight)
			_, err := fmt.Fprintf(w, "Execution statistics:\n"+
				"Total models recived\tStored in DB\tFailed to store in DB\n"+
				"%d\t%d\t%d", totalRecCnt, storedModelsCnt, failedToStoreCnt)
			if err != nil {
				log.Fatalf("Failed to print statistic: %v", err)
			}
			//fmt.Fprintln(w)
			if err := w.Flush(); err != nil {
				log.Fatalf("Failed to flush statistic writer: %v", err)
			}

			return

		case data := <-resChan:

			atomic.AddUint64(&totalRecCnt, 1)
			log.Debugf("Received model: %+v", data)
			log.Infof("Current amount of recieved models to store is: [%d]", totalRecCnt)
			errStore := mongo.StoreModel(data, dbCollection)
			if errStore != nil {
				log.Errorf("Failed to store model...: %v", errStore)
				atomic.AddUint64(&failedToStoreCnt, 1)
			} else {
				atomic.AddUint64(&storedModelsCnt, 1)

			}

		}
	}

}