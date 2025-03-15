FROM golang:1.21-alpine
    
WORKDIR /app
    
COPY go.mod ./
COPY go.sum ./
RUN go mod download
    
COPY . .
    
    # Adicione flags de debug para ver o que está acontecendo
RUN ls -la
RUN pwd
RUN go build -v -o auction cmd/auction/main.go
RUN ls -la
    
    # Certifique-se de que o executável existe e tem permissões corretas
RUN chmod +x auction
    
    # Use um ENTRYPOINT relativo ao WORKDIR
ENTRYPOINT ["./auction"]