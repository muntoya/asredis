# asredis
go开发的redis异步库。解决同步redis库在网络延迟上浪费掉的性能。
所有接口被封装为同步调用，除sub/pub外都是线程安全的。需要开多个goroutine来达到最高性能，与pipelining原理类似。
提供以下功能
- single connection，自动重连
- connection pool
- sub/pub
- lua
