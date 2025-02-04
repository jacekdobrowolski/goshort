import { check } from 'k6';
import http from 'k6/http';
import crypto from 'k6/crypto';
import encoding from 'k6/encoding';


function generateRandomUrl() {
    const domains = ['example.com', 'example.org', 'test.com', 'website.net'];
    const randomDomain = domains[Math.floor(Math.random() * domains.length)];
    const randomBytes = crypto.randomBytes(8);
    const urlSafeRandomPath = encoding.b64encode(randomBytes, "rawurl")

    return `http://${randomDomain}/${urlSafeRandomPath}`;
}

export let options = {
scenarios: {
    contacts: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { target: 300, duration: '5m' },
      ],
    },
  },
};

export default function () {
    const url = 'http://172.18.0.6/api/v1/links';
    const randomUrl = generateRandomUrl();
    const payload = JSON.stringify({ url: randomUrl });
    const params = {
        headers: { 'Content-Type': 'application/json' },
    };

    const res = http.post(url, payload, params);

    check(res, {
        'POST request should return 201': (r) => r.status === 201,
    });

    const responseBody = JSON.parse(res.body);
    const shortUrl = responseBody.short;

    check(responseBody, {
        'should have short URL': (r) => r.short !== undefined,
        'should have original URL': (r) => r.original === randomUrl,
    });

    const getResponse = http.get(shortUrl, {redirects: 0});

    check(getResponse, {
        'GET request to short URL should return 307': (r) => r.status === 307,
    });
}

