# asredis
go开发的高性能redis。使用pipelining解决在网络花费，并提供对sentinel集群和cluster的自动连接.

所有接口被封装为同步调用，除sub/pub外都是线程安全的。需要开多个goroutine来达到最高性能，与pipelining原理类似。

提供以下功能
- single connection，自动重连
- connection pool
- cluster
- sub/pub
- lua
