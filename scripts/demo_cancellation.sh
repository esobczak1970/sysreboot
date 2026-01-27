#!/bin/bash
# Demonstration of sysreboot cancellation feature
# This script shows how to use --cancel and --status flags

set -e

echo "==================================================="
echo "sysreboot Cancellation Feature Demonstration"
echo "==================================================="
echo ""

# Build the binary
echo "1. Building sysreboot..."
go build -o sysreboot
echo "   ✓ Build successful"
echo ""

# Show version
echo "2. Checking version..."
./sysreboot --version
echo ""

# Show help with new flags
echo "3. Showing usage (note --cancel and --status flags)..."
./sysreboot --help 2>&1 | head -30
echo ""

# Test status when nothing is scheduled
echo "4. Checking status when no actions are scheduled..."
sudo ./sysreboot --status || true
echo ""

# Schedule a reboot with delay (in background)
echo "5. Scheduling a reboot with 60 minute delay..."
echo "   (Running in background as PID file test)"
sudo ./sysreboot --reboot --delay 60 --message "Test reboot" &
REBOOT_PID=$!
sleep 2  # Give it time to create PID file
echo "   ✓ Scheduled (Process PID: $REBOOT_PID)"
echo ""

# Check status
echo "6. Checking status of scheduled action..."
sudo ./sysreboot --status
echo ""

# Cancel the scheduled action
echo "7. Cancelling the scheduled action..."
sudo ./sysreboot --cancel
sleep 1
echo ""

# Verify cancellation
echo "8. Verifying cancellation (status should show none)..."
sudo ./sysreboot --status || true
echo ""

# Schedule with specific time
echo "9. Scheduling reboot for specific time (1 minute from now)..."
FUTURE_TIME=$(date -d '+1 minute' '+%H:%M' 2>/dev/null || date -v +1M '+%H:%M')
echo "   Scheduling for: $FUTURE_TIME"
sudo ./sysreboot --reboot --time "$FUTURE_TIME" &
REBOOT_PID=$!
sleep 2
echo "   ✓ Scheduled (Process PID: $REBOOT_PID)"
echo ""

# Check status again
echo "10. Checking status..."
sudo ./sysreboot --status
echo ""

# Cancel via signal (Ctrl+C simulation)
echo "11. Cancelling via signal (SIGTERM)..."
sudo kill -TERM $REBOOT_PID 2>/dev/null || true
sleep 1
echo "    ✓ Signal sent"
echo ""

# Final status check
echo "12. Final status check..."
sudo ./sysreboot --status || true
echo ""

echo "==================================================="
echo "Demonstration Complete!"
echo "==================================================="
echo ""
echo "Key Features Demonstrated:"
echo "  • PID file management for scheduled actions"
echo "  • --status flag to view pending actions"
echo "  • --cancel flag to cancel scheduled actions"
echo "  • Signal handling (Ctrl+C) for graceful cancellation"
echo "  • Works with both --delay and --time scheduling"
echo ""
echo "Usage Examples:"
echo "  sysreboot --reboot --delay 30       # Schedule with delay"
echo "  sysreboot --status                  # Check what's scheduled"
echo "  sysreboot --cancel                  # Cancel scheduled action"
echo "  [Press Ctrl+C while waiting]        # Cancel interactively"
echo ""
