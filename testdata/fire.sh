alertname="${1:-TestAlert}"

curl -i -X POST http://localhost:9093/api/v2/alerts \
  -H "Content-Type: application/json" \
  -d "[
    {
      \"status\": \"firing\",
      \"labels\": {
        \"alertname\": \"${alertname}\",
        \"instance\": \"localhost:9090\",
        \"job\": \"prometheus\",
        \"icinga_example\": \"variable\",
        \"icinga_number_myNumber\": \"123\",
        \"icinga_string_myString\": \"foobar\",
        \"severity\": \"critical\"
      },
      \"annotations\": {
        \"summary\": \"This is a test alert\",
        \"description\": \"Manually triggered test alert for webhook testing\"
      },
      \"startsAt\": \"2026-01-01T00:00:00Z\",
      \"generatorURL\": \"http://localhost:9090/graph\"
    }
  ]"
