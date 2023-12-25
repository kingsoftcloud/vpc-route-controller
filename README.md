# VPC路由控制器

## 1. 简介

  vpc-route-controller负责集群内每个节点上容器网段的vpc路由管理，监听Node的状态变化：
  
  * 当Node新加入集群时，为新增Node创建Vpc主机路由；
  
  * 当Node从集群中移除时，删除Node的Vpc主机路由记录。

  另外，默认每隔5分钟，检测集群内的所有Node相关路由记录。若不存在，则重新创建。

## 2. 编译构建镜像

### 2.1. calico
```sh
# git clone git@github.com:kingsoftcloud/calico.git
# cd calico/cni-plugin
# make image
# docker tag calico/cni:latest hub.kce.ksyun.com/ksyun/calico/cni:v3.24.6-for-selfbuild-cluster
# docker push hub.kce.ksyun.com/ksyun/calico/cni:v3.24.6-for-selfbuild-cluster
```
### 2.2. vpc-route-controller
```sh
# git clone git@github.com:kingsoftcloud/vpc-route-controller.git
# cd vpc-route-controller
```
修改Makefile，将“yourcipherkey”改为实际的密钥，如果SK无需加密，则设置为空即可。保存退出
```
CIPHER_KEY=$(shell echo "yourcipherkey")
```
假设KEY='404633a025a386e110d54242a48f885e'，则
```yaml
CIPHER_KEY=$(shell echo "404633a025a386e110d54242a48f885e")
```
假设不加密，则
```yaml
CIPHER_KEY=$(shell echo "")
```

然后，请将Makefile中镜像仓库地址修改为自己实际的地址
```yaml
BJKSYUNREPOSITORY:= hub.kce.ksyun.com/ksyun/vpc-route-controller
```

最后执行make命令
```sh
# make all
```

## 3. 配置aksk
示例：
```yaml
AK："AKTRQxqRY0SdCw31S46rrcMA"
SK："ODPedeQvrIo2BF6QkzkZ1HZdhkjH648cOF0fVXGt"
KEY: "404633a025a386e110d54242a48f885e"（32位）
```
### 3.1. 如果需要对SK加密，则执行以下操作：
注意：如果不需要对SK加密，请忽略此步

* 首先将KEY字符串转换16进制：

string转16进制命令：
```sh
# echo -n '404633a025a386e110d54242a48f885e' | xxd -p
3430343633336130323561333836653131306435343234326134386638383565
```
* 执行加密命令：
```sh
#echo -n "ODPedeQvrIo2BF6QkzkZ1HZdhkjH648cOF0fVXGt" |openssl enc -aes-256-cbc -e -a -K 3430343633336130323561333836653131306435343234326134386638383565 -iv 34303436333361303235613338366531
```
参数说明：

  -e 加密
  
  -a  加密后以base64编码
  
  -K 加密key （16进制）
  
  -iv iv值(固定长度：16位)   （16进制）取密钥key的前16位作为iv值

加密后字符串：
```sh
70aM3hAdVJMB/yJHOxIB3iHyST0aijaIQWoIXCo6yLgFRofS2lHs62Q0Z6wAhgY+
```

### 3.2. 创建secret，将AK、SK保存在其中
* 当对SK加密时：
```sh
# kubectl create secret generic kce-security-token --from-literal=ak='AKTRQxqRY0SdCw31S46rrcMA' --from-literal=sk='70aM3hAdVJMB/yJHOxIB3iHyST0aijaIQWoIXCo6yLgFRofS2lHs62Q0Z6wAhgY+' --from-literal=cipher='aes256+base64' -n kube-system
```
* 当不对SK加密时：
```sh
# kubectl create secret generic kce-security-token --from-literal=ak='AKTRQxqRY0SdCw31S46rrcMA' --from-literal=sk='ODPedeQvrIo2BF6QkzkZ1HZdhkjH648cOF0fVXGt' -n kube-system
```
## 4. 修改部署文件
使用deploy/calico_secret_aksk.yaml，将以下几个变量，修改为实际的值
```yaml
___POD_CIDR___：集群pod网络使用的cidr
___VPC_CIDR___：集群所在vpc的cidr
___REGION___：集群所在region，比如北京6，值为： cn-beijing-6
___VPC_ID___: 集群所在vpc的id
___CLUSTER_UUID___：集群的uuid
```
另外，请注意，如果Kubernetes版本在1.23(含）以上，请将yaml文件中的monitor_token修改为true
```yaml
monitor_token: "true"
```
最后，请将yaml文件中的镜像修改为实际的地址和tag

## 5. 部署calico和vpc-route-controller
```sh
# kubectl apply -f deploy/calico_secret_aksk.yaml
```
确保kube-system命名空间下的所有calico-cni和vpc-route-controller处于running状态

创建工作负载，能够跨节点访问，代表容器网络部署成功。

如果跨节点访问Pod不通，请您检查Pod所在节点上的iptables规则是否默认放行了FORWARD流量，如果默认DROP，请执行以下命令：
```sh
# iptables -t filter -P FORWARD ACCEPT
```
