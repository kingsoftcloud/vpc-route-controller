维护者：<lijunli@kingsoft.com>

1.简介
  将容器网络的vpc主机路由功能从flannel插件中剥离出来，作为一个独立的组件存在。
  vpc-route-controller监听Node的状态变化：
  1）当Node新加入集群时，为新增Node创建Vpc主机路由；
  2）当Node从集群中移除时，删除Node的Vpc主机路由记录。

  另外，默认每隔5分钟，检测集群内的所有Node相关路由记录。若不存在，则重新创建。


