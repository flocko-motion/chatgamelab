FROM postgres:18

# Install SSH client + gzip for backups
RUN apt-get update && \
    apt-get install -y --no-install-recommends openssh-client gzip && \
    rm -rf /var/lib/apt/lists/*

# Backup script
COPY docker/db/backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

# Keep default postgres entrypoint
