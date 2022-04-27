#!/bin/sh
#
#{"actuators":[ {"pin":8, "trigger": 1} ]}
#
curl -X PUT -H "Content-Type: application/json" -d '{"endpoint_type":"rest","endpoint":"http://192.168.12.5:8999/konnected","token":"notyet", "sensors":[{"pin":1},{"pin":2},{"pin":5},{"pin":6},{"pin":7}] }' http://192.168.12.252:14996/settings
