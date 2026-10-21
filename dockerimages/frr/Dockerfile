FROM frrouting/frr:latest

USER root

RUN apk update
RUN apk add --no-cache openssh iproute2 iputils bash vim tcpdump

RUN mkdir -p /run/sshd
RUN ssh-keygen -A

RUN sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
RUN sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config
RUN echo 'root:root' | chpasswd

CMD ["/bin/sh", "-c", "/usr/sbin/sshd && /usr/lib/frr/docker-start"]
