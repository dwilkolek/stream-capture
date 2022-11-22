FROM ubuntu:latest
WORKDIR /usr/src/app
RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get install -y ffmpeg

COPY app app
RUN chmod +x app

ARG ffmpeg="ffmpeg"
ENV FFMPEG=$ffmpeg

ARG streamUrl
ENV STREAM_URL=$streamUrl
ARG cron
ENV CRON=$cron

ARG rec_timeout
ENV REC_TIMEOUT=$rec_timeout

ARG ftp
ENV FTP=$ftp

CMD ["./app"]

