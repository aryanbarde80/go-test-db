FROM ubuntu:22.04

# Set non-interactive mode
ENV DEBIAN_FRONTEND=noninteractive

# Install MySQL, MongoDB, Golang, and Supervisor
RUN apt-get update && apt-get install -y \
    mysql-server \
    mongodb \
    golang-go \
    supervisor \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . /app/

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

# Supervisord config to run MySQL, MongoDB, and Go app
RUN echo '[supervisord]' > /etc/supervisor/conf.d/app.conf && \
    echo 'nodaemon=true' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:mysql]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/usr/bin/mysqld_safe' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:mongodb]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/usr/bin/mongod --bind_ip 0.0.0.0 --port 27017' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo '' >> /etc/supervisor/conf.d/app.conf && \
    echo '[program:app]' >> /etc/supervisor/conf.d/app.conf && \
    echo 'command=/app/inventory-app' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autostart=true' >> /etc/supervisor/conf.d/app.conf && \
    echo 'autorestart=true' >> /etc/supervisor/conf.d/app.conf

# Expose ports
EXPOSE 5000 3306 27017

# Start supervisor
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/app.conf"]
