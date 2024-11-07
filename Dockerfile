FROM python:3.13-alpine
COPY --from=golang:1.23-alpine /usr/local/go/ /usr/local/go/
ENV PATH="/usr/local/go/bin:${PATH}"
COPY . /app
WORKDIR /app
RUN pip install requests
RUN pip install flask
CMD ["go", "run", "main.go"]