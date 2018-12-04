# 代码下载与编译

编译好可执行文件：
    
    gelchain

# 运行脚本启动网络

cd $GOPATH/src/github.com/green-element-chain/gelchain/scripts/network

make [build|up|down]

`说明:`

    参数介绍:
        build：         编译可执行二进制文件
        up：            启动初始化网络节点，支持新老版本
        down：          删除网络节点，所有节点容器都会被删除掉
    
    网络节点配置文件，可以根据需要修改：
        $GOPATH/src/github.com/green-element-chain/gelchain/scripts/network/config/env.json
        
        文件配置解释：
            {
                "system":"ubuntu:16.04",    //监控节点镜像的操作系统
                "localhost":"192.168.56.4", //本地IP地址，脚本判断是否本机使用
                "user": { "name":"root", "passwd":"Energy@123" }, //分布式节点部署使用的用户名和密码
                "datapath":"/home/share/chaindata", //节点数据目录路径
                "setup": {
                    "port": [
                        {
                            "db":"/home/share/gelchain_data",
                            "ports":"8545:8545 46656:46656 47757:26657 48858:26658 19190:19190",
                            "loglevel":"info",
                            "debug":0
                        }
                    ],
                    "node": {
                        "init": [
                            /*网络初始启动节点，type为类型：1为共识节点，0为普通节点*/
                            "peer1=192.168.56.4,type=1",
                            "peer2=192.168.56.4,type=1",
                            "peer3=192.168.56.4,type=1",
                            "peer4=192.168.56.4,type=1"
                        ],
                        "add": {
                            "from": { "node":"peer1", "data":"peer5" }, //恢复损坏的节点使用
                            "host": [
                                "peer6=192.168.56.4,type=1",
                                "peer7=192.168.56.4,type=0"
                            ]
                        }
                    }
                }
            }
