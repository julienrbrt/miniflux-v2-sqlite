#!/bin/bash

# Miniflux SQLite Edition - Example Usage
# This script demonstrates how to set up and run Miniflux with SQLite

set -e

echo "🚀 Miniflux SQLite Edition Setup Example"
echo "========================================"

# Configuration
DATABASE_FILE="./miniflux.db"
ADMIN_USERNAME="admin"
ADMIN_PASSWORD="admin123"
SERVER_PORT="8080"

# Clean up any existing database for demo
if [ -f "$DATABASE_FILE" ]; then
    echo "🗑️  Removing existing database for fresh start..."
    rm "$DATABASE_FILE"
fi

# Build the application
echo "🔨 Building Miniflux SQLite..."
go build -o miniflux-sqlite .

# Set environment variables
export DATABASE_URL="$DATABASE_FILE"
export LISTEN_ADDR="localhost:$SERVER_PORT"
export BASE_URL="http://localhost:$SERVER_PORT"

echo "📊 Using SQLite database: $DATABASE_FILE"
echo "🌐 Server will run on: http://localhost:$SERVER_PORT"

# Run database migrations
echo "🗄️  Running database migrations..."
./miniflux-sqlite -migrate

# Create admin user
echo "👤 Creating admin user..."
echo "Username: $ADMIN_USERNAME"
echo "Password: $ADMIN_PASSWORD"
echo "$ADMIN_PASSWORD" | ./miniflux-sqlite -create-admin "$ADMIN_USERNAME"

echo ""
echo "✅ Setup complete!"
echo ""
echo "📋 Database Information:"
echo "   - File: $DATABASE_FILE"
echo "   - Size: $(du -h "$DATABASE_FILE" | cut -f1)"
echo "   - Tables: $(sqlite3 "$DATABASE_FILE" "SELECT COUNT(*) FROM sqlite_master WHERE type='table';")"
echo ""
echo "🔑 Admin Credentials:"
echo "   - Username: $ADMIN_USERNAME"
echo "   - Password: $ADMIN_PASSWORD"
echo ""
echo "🚀 Starting Miniflux server..."
echo "   Open your browser to: http://localhost:$SERVER_PORT"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start the server
./miniflux-sqlite
