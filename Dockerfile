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
