#!/bin/bash

# Hospital CMS Database Migration Script
# This script completely recreates the database from scratch

set -e  # Exit on any error

echo "🏥 Hospital CMS Database Migration Starting..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found!"
    echo "Please create a .env file with the following variables:"
    echo "DB_NAME=your_database_name"
    echo "DB_USERNAME=your_username"
    echo "DB_PASSWORD=your_password"
    echo "DB_HOST=your_host"
    echo "DB_PORT=your_port"
    exit 1
fi

# Load environment variables
echo "📋 Loading environment variables..."
# Use Bash export-all + source to preserve spaces and quoted values
set -a
. ./.env
set +a
export PGPASSWORD="$DB_PASSWORD"

# Validate required environment variables
required_vars=("DB_NAME" "DB_USERNAME" "DB_PASSWORD" "DB_HOST" "DB_PORT")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ Error: $var is not set in .env file"
        exit 1
    fi
done

echo "✅ Environment variables loaded successfully"
echo "   Database: $DB_NAME"
echo "   Host: $DB_HOST:$DB_PORT"
echo "   User: $DB_USERNAME"

# Check if schema.sql exists
if [ ! -f schema.sql ]; then
    echo "❌ Error: schema.sql file not found!"
    exit 1
fi

# Check if default_seeds.sql exists
if [ ! -f default_seeds.sql ]; then
    echo "❌ Error: default_seeds.sql file not found!"
    exit 1
fi

echo "📁 Schema and seed files found"

# Test database connection
echo "🔌 Testing database connection..."
if ! psql "host=$DB_HOST port=$DB_PORT user=$DB_USERNAME dbname=postgres" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "❌ Error: Cannot connect to PostgreSQL server"
    echo "Please check your database credentials and server status"
    exit 1
fi
echo "✅ Database connection successful"

# Step 2: Load schema
echo "📊 Loading database schema..."
if psql "host=$DB_HOST port=$DB_PORT user=$DB_USERNAME dbname=$DB_NAME" -f "schema.sql"; then
    echo "✅ Schema loaded successfully"
else
    echo "❌ Error: Failed to load schema"
    exit 1
fi

# Step 3: Load seed data
echo "🌱 Loading seed data..."
if psql "host=$DB_HOST port=$DB_PORT user=$DB_USERNAME dbname=$DB_NAME" -f "default_seeds.sql"; then
    echo "✅ Seed data loaded successfully"
else
    echo "❌ Error: Failed to load seed data"
    exit 1
fi

# Step 4: Verify installation
echo "🔍 Verifying database installation..."
table_count=$(psql "host=$DB_HOST port=$DB_PORT user=$DB_USERNAME dbname=$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')

if [ "$table_count" -gt 0 ]; then
    echo "✅ Database verification successful"
    echo "   Tables created: $table_count"
else
    echo "❌ Error: No tables found in database"
    exit 1
fi

# Step 5: Show database summary
echo "📈 Database Summary:"
echo "   Database Name: $DB_NAME"
echo "   Tables Created: $table_count"
echo "   Admin User: admin@saglik.com (password: admin123)"
echo "   Default Options: Loaded"
echo "   Sample Data: Loaded"

echo ""
echo "🎉 Hospital CMS Database Migration Completed Successfully!"
echo ""
echo "📝 Next Steps:"
echo "   1. Update the admin password from 'admin123' to a secure password"
echo "   2. Configure your application to use these database credentials"
echo "   3. Test your application connection"
echo ""
echo "🔗 Connection String:"
echo "   postgresql://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
