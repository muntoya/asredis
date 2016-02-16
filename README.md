# asredis
go开发的高性能redis。使用pipelining解决在网络花费，并提供对sentinel集群和cluster的自动连接.

所有接口被封装为同步调用，除sub/pub外都是线程安全的。

需要开多个goroutine来达到最高性能，与pipelining原理类似。

单个连接中goroutine的数量需要大于pipelining队列长度，但过大上下文切换频分会影响性能。一般pipelining队列长度应该在20-50之间。
连接池中连接数要尽量小，为其他进程留下资源。

提供以下功能
- single connection，自动重连
- connection pool
- cluster
- sub/pub
- lua
