#!/bin/bash
echo "Testing heredoc syntax"
cat << 'INNER_EOF'
This is a test of nested heredocs
INNER_EOF
