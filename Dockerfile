FROM golang:1.19
USER root


ARG CHANGE_SOURCE=true
RUN if [ ${CHANGE_SOURCE} = true ]; then \
    # Change application source from deb.debian.org to aliyun source
    touch /etc/apt/sources.list && \
    sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apt/sources.list && \
    sed -i 's/security.debian.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apt/sources.list && \
    sed -i 's/security-cdn.debian.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apt/sources.list \
;fi

RUN apt-get update -yqq && \
    apt-get install build-essential -y && \
    apt-get install flex -y && \
    apt-get install bison -y

RUN wget https://www.tcpdump.org/release/libpcap-1.10.1.tar.gz  && \
    tar zxvf  libpcap-1.10.1.tar.gz && \
    cd libpcap-1.10.1 && \
    ./configure && make && make install

RUN cp -rf /usr/local/lib/* /usr/lib

RUN apt-get install -y nmap masscan
RUN apt-get install -y apt-utils
RUN apt-get install libasound2  -y
RUN apt-get install ttf-wqy-microhei ttf-wqy-zenhei xfonts-wqy -y

RUN mkdir /app
WORKDIR /app
ADD ./google-chrome-stable_current_amd64.deb /tmp/google-chrome-stable_current_amd64.deb
RUN dpkg -i /tmp/google-chrome-stable_current_amd64.deb || :
RUN apt --fix-broken install -y

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
# RUN go mod tidy



# EXPOSE 18000

CMD ["tail","-f"]
# CMD ["/app/worker"]