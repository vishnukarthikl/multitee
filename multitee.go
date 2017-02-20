package main

import "bufio"
import (
  "os"
  "io"
  "fmt"
  "io/ioutil"
  "net/http"
  "strings"
)

type multiTee struct {
  reader    *bufio.Reader
  processes []*teeProcess
}

func (tee *multiTee) multiplex() (error) {
  done := make(chan bool)
  errs := make(chan error)
  defer close(done)
  defer close(errs)
  go tee.copyToWriters(errs)
  for _, process := range tee.processes {
    process.launch(done, errs)
  }

  for c := 0; c < len(tee.processes); c++ {
    select {
    case err := <-errs:
      return err
    case <-done:
    }
  }
  return nil
}

func (tee *multiTee) getWriters() []*io.PipeWriter {
  writers := make([]*io.PipeWriter, 0)
  for _, process := range tee.processes {
    writers = append(writers, process.writer)
  }
  return writers
}

func (tee *multiTee) copyToWriters(errs chan error) {
  newWriters := make([]io.Writer, 0)
  for _, writer := range tee.getWriters() {
    defer writer.Close()
    newWriters = append(newWriters, writer)
  }

  mw := io.MultiWriter(newWriters...)
  _, err := io.Copy(mw, tee.reader)
  if err != nil {
    errs <- err
  }
}

type teeProcess struct {
  reader  *io.PipeReader
  writer  *io.PipeWriter
  process func(*io.PipeReader, chan bool, chan error)
}

func (teeProcess *teeProcess) launch(done chan bool, errs chan error) {
  go teeProcess.process(teeProcess.reader, done, errs)
}

func main() {
  reader := bufio.NewReader(os.Stdin)
  processes := []*teeProcess{newTeeProcess(processHTTP), newTeeProcess(processStandardOutput)}
  tee := newTee(reader, processes)
  err := tee.multiplex()
  if err != nil {
    fmt.Printf("\nERROR: %v\n", err)
  } else {
    fmt.Println("\nDONE")
  }
}

func processHTTP(reader *io.PipeReader, done chan bool, errs chan error) {
  url := os.Args[1]
  var bytes []byte
  var err error
  if bytes, err = ioutil.ReadAll(reader); err != nil {
    errs <- err
  } else {
    // ideally it should be
    // http.Post(url, "meh", reader)
    // but the contents are empty from the PipeReader
    // I tried figuring out what the issue was but could not
    if res, err := http.Post(url, "meh", strings.NewReader(string(bytes))); err == nil {
      fmt.Printf("\nHTTP\n%v\n", res.Status)
      done <- true
    } else {
      errs <- err
    }
  }
}

func processStandardOutput(reader *io.PipeReader, done chan bool, errs chan error) {
  if _, err := io.Copy(os.Stdout, reader); err == nil {
    done <- true
  } else {
    errs <- err
  }
}

func newTee(reader *bufio.Reader, processes []*teeProcess) (*multiTee) {
  return &multiTee{reader, processes}
}

func newTeeProcess(f func(*io.PipeReader, chan bool, chan error)) (*teeProcess) {
  r, w := io.Pipe()
  return &teeProcess{r, w, f}
}