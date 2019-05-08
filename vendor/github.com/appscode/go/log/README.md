# log

Simple [golang/glog](https://github.com/golang/glog) wrapper to works with contextual logs.

This wrapper only binds glog's Leveled logging to internal logging functions, and in logger
to work with contextes. Context and formatter are provided by the caller while initiated.
Log do not have any control over the context.

The purpose of this package is to bind the vLeveled log to predifined levels and work with
contexts value its provided.

Log levels  are defined with respect to VLevel as:
```
 Fatal   - 0
 Error   - 1
 Warning - 2
 Info    - 3
 Debug   - 4
```
 
More Leveled log can be done via
 **V(Level)**
 **If Level is lower than `Debug` level Level will be increased to `Debug` level.** 
 
 
## Simple Usage of Context
```go
 type Context string

 func (c Context) Format() string {
     return "desired format" + string(c)
 }

 l := log.New(Context("hello")))
 l.Infoln("Log Line")
```

Customized context is available with customized formats.

## Available Flags
```	
    -logtostderr=false
		Logs are written to standard error instead of to files.
	-alsologtostderr=false
		Logs are written to standard error as well as to files.
	-log_dir=""
		Log files will be written to this directory instead of the
		default temporary directory.

	Other flags provide aids to debugging.

	-log_backtrace_at=""
		When set to a file and line number holding a logging statement,
		such as
	-log_backtrace_at=gopherflakes.go:234
		a stack trace will be written to the Info log whenever execution
		hits that statement. (Unlike with -vmodule, the ".go" must be
		present.)
	-v=0
		Enable V-leveled logging at the specified level.
	-vmodule=""
		The syntax of the argument is a comma-separated list of pattern=N,
		where pattern is a literal file name (minus the ".go" suffix) or
		"glob" pattern and N is a V level. For instance,
	-vmodule=gopher*=3
		sets the V level to 3 in all Go files whose names begin "gopher".
    
    
    
    IMPORTANT: Setting -stderrthreshold flag can cause no logging as our main goal is to level
               log upon Vlevel instead of severity.
    
    -stderrthreshold=INFO
    		Log events at or above this severity are logged to standard
    		error as well as to files.    	
```

## Run Benchmark
```
go test -bench=.
```
