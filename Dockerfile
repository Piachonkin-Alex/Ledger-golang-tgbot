FROM ubuntu:20.04
ENV LANGUAGE="en"
COPY Go-Cript .
RUN  chmod +x Go-Cript
RUN  apt-get update && apt-get install -y ca-certificates && update-ca-certificates
EXPOSE 80/tcp
CMD ["./Go-Cript"]


