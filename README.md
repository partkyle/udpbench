UDP Benchmarks
==============

Several different methods of UDP writing in Go

```
[mordekyle:~]$ go test github.com/partkyle/udp -bench .
testing: warning: no tests to run
PASS
BenchmarkDialEachTime                     50000      165875 ns/op
BenchmarkDialOnce                        500000        7249 ns/op
BenchmarkBuffered                       5000000         481 ns/op
BenchmarkSyscall                          50000      131669 ns/op
BenchmarkPersistantSyscall               500000       17572 ns/op
BenchmarkSyscallConnWrapper               10000      101010 ns/op
BenchmarkSyscallConnWrapperPersistant    500000        9638 ns/op
BenchmarkSyscallConnWrapperBuffered     5000000         515 ns/op
BenchmarkWriteTo                          50000      168094 ns/op
BenchmarkWriteToPersistant               500000        4963 ns/op
BenchmarkWriteToPersistantBuffered      5000000         472 ns/op
```
