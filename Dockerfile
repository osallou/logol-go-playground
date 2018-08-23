FROM debian:sid-slim

MAINTAINER Olivier Sallou <olivier.sallou@irisa.fr>

RUN apt-get update && apt-get install -y cassiopee

COPY logolClient /usr/bin/
COPY logolMessage /usr/bin/
COPY logolResult /usr/bin/
COPY logolCassie /usr/bin/

# localhost:6379
ENV LOGOL_REDIS_ADDR=${LOGOL_REDIS_ADDR}
# amqp://guest:guest@localhost:5672
ENV LOGOL_RABBITMQ_ADDR=${LOGOL_RABBITMQ_ADDR}

ENV LOGOL_DEBUG={$LOGOL_DEBUG}
