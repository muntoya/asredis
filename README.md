# asredis
go开发的redis异步库。解决同步redis库在网络延迟上浪费掉的性能，并提供对sentinel集群和cluster的自动连接.

所有接口被封装为同步调用，除sub/pub外都是线程安全的。需要开多个goroutine来达到最高性能，与pipelining原理类似。

提供以下功能
- single connection，自动重连
- connection pool
- master-slave pool，自动向sentinel获取全部服务信息
- cluster
- sub/pub
- lua
