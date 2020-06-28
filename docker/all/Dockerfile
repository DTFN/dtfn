FROM webbshi/dtfn-base
MAINTAINER webb.shi <bob.shi@energy-blockchain.com>

ENV PATH /root/.go/bin:/root/go/bin:$PATH
ENV GOPATH /root/go

RUN cd /root/go/src/github.com/DTFN/ && \
    git clone git@github.com:DTFN/dtfn.git && \
    cd /root/go/src/github.com/DTFN/dtfn

RUN cd /root/go/src/github.com/DTFN/dtfn/ && \
    make bls_ubuntu

RUN cd /root/go/src/github.com/DTFN/dtfn/ && \
    make develop_ubuntu

RUN cd /root/go/src/github.com/DTFN/dtfn/ && \
    make develop_build

