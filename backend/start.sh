#!/bin/bash
cd /root/trading-app/backend
export $(grep -v '^#' .env | xargs)
./trading-server
