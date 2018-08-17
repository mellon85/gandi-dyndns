FROM alpine:3.8

RUN apk add --no-cache python3 && pip3 install --upgrade pip

COPY requirements.txt /requirements.txt
RUN pip3 install -r /requirements.txt

COPY conf.json /conf.json
COPY sync.py /sync.py

ENTRYPOINT ["/usr/bin/python3", "/sync.py"]
