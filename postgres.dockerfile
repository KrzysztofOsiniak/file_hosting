FROM postgres:17.4-bookworm

RUN apt-get update &&  \ 
  apt-get -y install postgresql-17-cron && \ 
  apt-get clean \ 
  && rm -rf /var/lib/apt/lists/*

# Load extension for cron jobs
# Check if file contains desired config
RUN if [ -z $(grep "shared_preload_libraries='pg_cron'" /usr/share/postgresql/postgresql.conf.sample) ]; then \
  echo "shared_preload_libraries='pg_cron'" >> /usr/share/postgresql/postgresql.conf.sample; \
  echo "cron.database_name='app'" >> /usr/share/postgresql/postgresql.conf.sample; \
fi
