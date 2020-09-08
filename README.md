# Isucandar

![test](https://github.com/rosylilly/isucandar/workflows/test/badge.svg)

Utility packages for [ISUCON](http://isucon.net/) benchmarker.

## Packages

- `agent`: HTTP request agent.
  - Content-Encoding supports: gzip, deflate, brotli.
  - Always using HTTP/2 if you can.
  - Cache-Control supports like a browser.
  - HTML parse and sub-resource fetching.
- `failure`: Error collecter and wrap with error code.
  - Async error collection.
  - Wrap error code likes xerrors.
  - Collect backtrace when error creation.
  - Clean up backtrace with customize.
- `score`: Score collecter.
  - Async score collection.
