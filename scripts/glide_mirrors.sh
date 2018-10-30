#!/usr/bin/env bash
if [ -f "/root/.glide/mirrors.yaml" ];then
echo "glide mirror.yaml exist"
else
touch /root/.glide/mirrors.yaml
fi

glide mirror set https://golang.org/x/crypto https://github.com/golang/crypto --vcs git

glide mirror set https://golang.org/x/net https://github.com/golang/net --vcs git

glide mirror set https://golang.org/x/sys https://github.com/golang/sys --vcs git

glide mirror set https://golang.org/x/text https://github.com/golang/text --vcs git

glide mirror set https://google.golang.org/grpc https://github.com/grpc/grpc-go --vcs git

glide mirror set https://google.golang.org/genproto https://github.com/google/go-genproto --vcs git
