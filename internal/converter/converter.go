// Package converter provides functionality for processing logs.
package converter

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/oleg-balunenko/logs-converter/internal/models"
)

// Start starts converting of logfile
func Start(logName string, format string, mustExist bool, follow bool, resultChan chan *models.LogModel,
	errorsChan chan error, wg *sync.WaitGroup) {
	log.Infof("Starting tailing and converting file [%s] with logs format [%s]", logName, format)

	defer wg.Done()

	t, err := tail.TailFile(logName, tail.Config{
		Follow:    follow,
		MustExist: mustExist,
	})
	if err != nil {
		msg := fmt.Sprintf("failed to tail file [%s]", logName)
		errorsChan <- errors.Wrap(err, msg)

		return
	}

	var cnt uint64

	for line := range t.Lines {
		cnt++

		log.Debugf("File:[%s] Line tailed: [%v]", logName, line)

		model, err := processLine(logName, line.Text, format, cnt)

		if err != nil {
			errorsChan <- errors.Wrap(err, fmt.Sprintf("Failed to process line [%s]", line.Text))
		}

		log.Debugf("Go routine for file [%s] sending model to chanel", logName)

		resultChan <- model

		log.Debugf("Go routine for file [%s] sent model to chanel", logName)
	}
}

func processLine(logName string, line string, format string, lineNumber uint64) (*models.LogModel, error) {
	const (
		minLength   = 1
		timePos     = 0
		msgPos      = 1
		timeWithMsg = 2
	)

	lineElements := strings.Split(line, " | ")

	if len(lineElements) <= minLength {
		log.Errorf("processLine: [%s]: Line [%d] has wrong log structure: %s", logName, lineNumber, line)
		return nil, fmt.Errorf("[%s]: Line [%d] has wrong log structure: %s", logName, lineNumber, line)
	}

	logTime, err := parseTime(lineElements[timePos], format)
	if err != nil {
		return nil, err
	}

	msg := lineElements[msgPos]
	// For case when log message contain additional " | " to not miss other part of message
	if len(lineElements) > timeWithMsg {
		msg = strings.Join(lineElements[1:], " | ")
	}

	md := &models.LogModel{
		LogTime:   logTime,
		LogMsg:    msg,
		FileName:  logName,
		LogFormat: format,
	}

	return md, nil
}

// parseTime parses logTime string as format that was passed and return time.Time representation of it
func parseTime(logTimeStr string, format string) (time.Time, error) {
	var layout string

	switch format {
	case firstFormat:
		layout = firstFormatLayout
	case secondFormat:
		layout = secondFormatLayout
	default:
		return time.Time{}, errors.Errorf("unexpected time format received (%s)", logTimeStr)
	}

	logTime, err := time.Parse(layout, logTimeStr)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to parse logTime [%s] as format [%s]",
			logTimeStr, format)
	}

	return logTime, nil
}
