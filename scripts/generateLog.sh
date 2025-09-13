mkdir -p /home/shared/logs

{
  # Add some known patterns for testing
  for i in {1..1000}; do
    echo "ERROR: Something went wrong $i"
    echo "INFO: Process completed successfully $i"
    echo "DEBUG: Connection established $i"
  done
  
  # Fill the rest with base64 encoded random data
  dd if=/dev/urandom bs=1M count=59 2>/dev/null | base64
} > /home/shared/logs/fake.log