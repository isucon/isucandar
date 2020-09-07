# Isucandar

Utility packages for [ISUCON](http://isucon.net/) benchmarker.

## Packages

- `agent`: HTTP request agent.
    - Content-Encoding supports: gzip, deflate, brotli.
    - Always using HTTP/2 if you can.
    - Cache-Control supports like a browser.
- `failure`: Error collecter and wrap with error code.
    - Async error collection.
    - Wrap error code likes xerrors.
- `score`: Score collecter.
    - Async score collection.
