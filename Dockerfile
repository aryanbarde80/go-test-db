FROM ubuntu:22.04

# Set non-interactive mode
ENV DEBIAN_FRONTEND=noninteractive

# Install MySQL, Supervisor, Dependencies
RUN apt-get update && apt-get install -y \
    mysql-server \
    supervisor \
    git \
    curl \
    wget \
    gnupg \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.21 manually
RUN wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz && \
    rm go1.21.13.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

# Install MongoDB from official repo
RUN wget -qO - https://www.mongodb.org/static/pgp/server-7.0.asc | apt-key add - && \
    echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-7.0.list && \
    apt-get update && \
    apt-get install -y mongodb-org && \
    rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy go.mod first
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy go.mod go.sum
COPY go.mod go.sum ./

# Copy source code
COPY . /app/

# Tidy dependencies
RUN go mod tidy

# Build Go app
RUN go build -o inventory-app main.go

# Configure MySQL
RUN mkdir -p /var/run/mysqld && chown mysql:mysql /var/run/mysqld
RUN service mysql start && \
    mysql -e "CREATE DATABASE IF NOT EXISTS inventory_db;" && \
    mysql -e "CREATE USER 'appuser'@'localhost' IDENTIFIED BY 'password123';" && \
    mysql -e "GRANT ALL PRIVILEGES ON inventory_db.* TO 'appuser'@'localhost';" && \
    mysql -e "FLUSH PRIVILEGES;"

# Configure MongoDB
RUN mkdir -p /data/db && chown -R mongodb:mongodb /data/db

# Create supervisor config
RUN echo '[supervisord]' > /etc/supervisor/conf.d/app.conf && \
    echo 'nodaemon=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'user=root' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:mysql]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/usr/bin/mysqld_safe' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'stdout_logfile=/var/log/mysql.log' >> /etc/supervisor/conf.d/app.conf && \
    echo 'stderr_logfile=/var/log/mysql-error.log' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:mongodb]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/usr/bin/mongod --bind_ip 0.0.0.0 --port 27017 --dbpath /data/db' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'stdout_logfile=/var/log/mongodb.log' >> /etc/supervisor/conf.d/app.conf && \
    echo 'stderr_logfile=/var/log/mongodb-error.log' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:app]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/app/inventory-app' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'stdout_logfile=/var/log/app.log' >> /etc/supervisor/conf.d/app.conf && \
    echo 'stderr_logfile=/var/log/app-error.log' >> /etc/supervisor/conf.d/app.conf

# Expose ports
EXPOSE 5000 3306 27017

# Start supervisor
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/app.conf"]
