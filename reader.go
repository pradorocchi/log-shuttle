package main

import (
	"bufio"
	"io"
	"net"
	"time"
)

const (
	UNIXGRAM_BUFFER_SIZE = 10000 //Make this a little smaller than logplex's max (10240), so we have room for headers
)

type LogLine struct {
	line    []byte
	when    time.Time
	rfc3164 bool
}

type Reader struct {
	Outbox chan *LogLine
}

func NewReader(frontBuff int) *Reader {
	r := new(Reader)
	r.Outbox = make(chan *LogLine, frontBuff)
	return r
}

func (rdr *Reader) readFromUnixgram(input *net.UnixConn, out chan<- *LogLine) {
	msg := make([]byte, UNIXGRAM_BUFFER_SIZE)
	for {
		numRead, err := input.Read(msg)
		if err != nil { // TODO: Do this better of just log.Fatal
			input.Close()
			return
		}

		//make a new []byte of the right length and copy our read message into it
		thisMsg := make([]byte, numRead)
		copy(thisMsg, msg)

		out <- &LogLine{thisMsg, time.Now(), true}
	}
}

func (rdr *Reader) ReadUnixgram(input *net.UnixConn, stats *ProgramStats, closeChan <-chan bool) {
	in := make(chan *LogLine)
	go rdr.readFromUnixgram(input, in)
	for {
		select {
		case msg := <-in:
			rdr.Outbox <- msg
			stats.Reads.Add(1)
		case <-closeChan:
			return
		}
	}
}

func (rdr *Reader) Read(input io.ReadCloser, stats *ProgramStats) error {
	rdrIo := bufio.NewReader(input)

	for {
		line, err := rdrIo.ReadBytes('\n')

		if err != nil {
			input.Close()
			return err
		}

		logLine := &LogLine{line, time.Now(), false}

		rdr.Outbox <- logLine
		stats.Reads.Add(1)
	}
	return nil
}
