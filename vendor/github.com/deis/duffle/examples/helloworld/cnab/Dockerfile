FROM alpine:latest

COPY app/run /cnab/app/run
COPY bundle.json /cnab/bundle.json
COPY Dockerfile cnab/Dockerfile

CMD [ "/cnab/app/run" ]