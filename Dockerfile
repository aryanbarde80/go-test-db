FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    mysql-server \
    supervisor \
    git \
    curl \
    wget \
    gnupg \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.21
RUN wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz && \
    rm go1.21.13.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

# Install MongoDB
RUN wget -qO - https://www.mongodb.org/static/pgp/server-7.0.asc | apt-key add - && \
    echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-7.0.list && \
    apt-get update && \
    apt-get install -y mongodb-org && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY go.mod go.sum ./
COPY . /app/

RUN go mod tidy
RUN go build -o inventory-app main.go

# Configure MySQL with sample data
RUN mkdir -p /var/run/mysqld && chown mysql:mysql /var/run/mysqld
RUN service mysql start && \
    mysql -e "CREATE DATABASE IF NOT EXISTS inventory_db;" && \
    mysql -e "CREATE USER 'appuser'@'localhost' IDENTIFIED BY 'password123';" && \
    mysql -e "GRANT ALL PRIVILEGES ON inventory_db.* TO 'appuser'@'localhost';" && \
    mysql -e "FLUSH PRIVILEGES;" && \
    mysql inventory_db -e "CREATE TABLE IF NOT EXISTS products (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), quantity INT, price DECIMAL(10,2));" && \
    mysql inventory_db -e "INSERT INTO products (name, quantity, price) VALUES ('Laptop', 10, 50000), ('Mouse', 25, 500), ('Keyboard', 15, 1200), ('Monitor', 8, 15000);"

RUN mkdir -p /data/db && chown -R mongodb:mongodb /data/db

# Supervisor config
RUN echo '[supervisord]' > /etc/supervisor/conf.d/app.conf && \
    echo 'nodaemon=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'user=root' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:mysql]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/usr/bin/mysqld_safe' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:mongodb]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/usr/bin/mongod --bind_ip 0.0.0.0 --port 27017 --dbpath /data/db' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:app]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/app/inventory-app' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf

EXPOSE 5000 3306 27017

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/app.conf"]
