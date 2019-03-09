FROM centos:latest

RUN rm /etc/yum.repos.d/*
COPY remirror.repo /etc/yum.repos.d/
# fastmirror takes a while to return so slows things down
RUN  sed -i 's/^enabled=1/enabled=0/' /etc/yum/pluginconf.d/fastestmirror.conf
RUN time yum install -y docker
