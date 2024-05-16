#!/bin/bash

#old = 11409279
NUMBER_OF_REQUESTS=8
START_AT=11211422

#Using the loop index in the command
for (( i=1; i<=NUMBER_OF_REQUESTS; i++ ))
do
    echo "retrieve_block.go $START_AT,asc,60"
    go run retrieve_block.go $START_AT,asc,60 ../../chain/westend/chain-spec-raw.json req_prev_$i.out > __out 2>&1
    START_AT=$(cat __out | awk '{
        # Loop through each field in the line
        for (i=1; i<=NF; i++) {
            # Check if the field starts with "#"
            if ($i ~ /^#/) {
                count++;
                # When the second occurrence is found, print the number without the "#"
                if (count == 2) {
                    sub(/^#/, "", $i);  # Remove the leading "#" from the field
                    print $i + 1; # increment it and return
                    exit;  # Exit after printing the second number
                }
            }
        }
    }')
done

rm -rf __out
