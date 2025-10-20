#!/bin/bash

# Load DB_NAME from environment
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

sudo -u postgres psql -d "$DB_NAME" -f schema.sql
sudo -u postgres psql -d "$DB_NAME" -f default_seeds.sql
