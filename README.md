# multitee
tee on steroids. You can customize and chain multiple tee processes to redirect stdin to
```
go build multitee.go
echo 'hello world' | multitee https://http-tee.herokuapp.com/request
```

* write a function `func(reader *io.PipeReader, done chan bool, errs chan error)` and pass it to `newTeeProcess`. 
* create a new `multitee` by passing the returned teeProcess and the `reader` from which contents are to be redirected
* `tee.multiplex()` will fan out contents of `reader` to all the processes

````golang
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
````

###What this does
* Creates a reader-writer pipe. 
* Creates a multiwriter with all those writers
* Uses io.Copy to copy contents from the main reader to all writers
* Each tee process(function) is injected with a corresponding reader end of the pipe.
* Uses that reader to read contents and do whatever

```
                                                   writer ==== reader -> processHTTP (read from reader for contents)
io.Copy(originalReader,multiwriter) -> write(bytes)
                                                   writer ==== reader -> processWhatever
```
