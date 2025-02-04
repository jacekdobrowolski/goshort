FROM grafana/k6:latest

COPY tests/add_and_get/test.js /test.js

CMD ["k6", "run", "/test.js"]

