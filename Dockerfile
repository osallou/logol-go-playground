FROM debian:sid-slim

MAINTAINER Olivier Sallou <olivier.sallou@irisa.fr>

RUN apt-get update && apt-get install -y cassiopee

COPY logolClient /usr/bin/
COPY logolMessage /usr/bin/
COPY logolResult /usr/bin/
COPY logolCassie /usr/bin/
